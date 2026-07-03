// Package sigv4 verifies incoming HTTP requests that were signed with AWS
// Signature Version 4 (the AWS4-HMAC-SHA256 scheme). It is the server-side
// counterpart to what the AWS SDKs do when they sign outgoing requests.
//
// Verification works by recomputing the signature from the request and the
// caller's secret, then comparing it against the presented signature in
// constant time. The set of signed headers is taken from the Authorization
// header itself, which is the authoritative list of what the client hashed.
package sigv4

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/twitchtv/twirp"
)

const (
	algorithm       = "AWS4-HMAC-SHA256"
	terminator      = "aws4_request"
	amzTimeFormat   = "20060102T150405Z"
	shortDateFormat = "20060102"
)

// Payload-hash sentinels defined by AWS SigV4. They are shared with the iamsts
// middleware so both sides agree on one definition and one comparison rule:
// UnsignedPayload matches exactly (AWS defines it case-sensitively), while
// StreamingPayload is matched case-insensitively because it is only ever used
// to reject requests, and rejecting more is the safe direction.
const (
	UnsignedPayload  = "UNSIGNED-PAYLOAD"
	StreamingPayload = "STREAMING-AWS4-HMAC-SHA256-PAYLOAD"
)

// Errors returned by Verify. Callers typically map ErrUnauthorized and
// ErrUnknownKey to 403 and the rest to 400.
var (
	ErrMissingAuth          = errors.New("sigv4: missing or malformed Authorization")
	ErrMissingSignedHost    = errors.New("sigv4: host must appear in SignedHeaders")
	ErrUnknownKey           = errors.New("sigv4: unknown access key id")
	ErrClockSkew            = errors.New("sigv4: request time outside allowed skew")
	ErrScopeMismatch        = errors.New("sigv4: credential scope does not match")
	ErrStreamingUnsupported = errors.New("sigv4: streaming payloads are not supported")
	ErrBodyHash             = errors.New("sigv4: body does not match x-amz-content-sha256")
	ErrUnauthorized         = errors.New("sigv4: signature mismatch")
	ErrBodyTooLarge         = errors.New("sigv4: request body exceeds limit")
	ErrNotConfigured        = errors.New("sigv4: Verifier.Lookup is not set")
)

// Verifier validates SigV4-signed requests for a single region/service.
type Verifier struct {
	// Region and Service must match the credential scope in the request.
	Region  string
	Service string

	// Lookup resolves an access key id to its secret. Return ErrUnknownKey
	// for unknown keys. This is the one piece you must supply.
	Lookup Lookuper

	// MaxClockSkew bounds how far the request's X-Amz-Date may be from now.
	// Defaults to 15 minutes (matching AWS) when zero.
	MaxClockSkew time.Duration

	// DisablePathEscaping selects S3-style canonicalization, where the path
	// is used as-is rather than being URI-encoded a second time. Set true
	// when emulating S3; leave false for every other AWS service.
	DisablePathEscaping bool

	// MaxBodySize caps how many bytes of the body will be buffered to verify
	// the payload hash. Zero means unlimited. Requests that exceed it are
	// rejected with ErrBodyTooLarge.
	MaxBodySize int64

	// Now is overridable for tests. Defaults to time.Now.
	Now func() time.Time
}

// Middleware returns net/http middleware that rejects unsigned or invalid
// requests with 401/403 and otherwise passes them through. On success it
// stores the verified access key id in the request context.
func (v *Verifier) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keyID, err := v.Verify(r)
		if err != nil {
			slog.DebugContext(r.Context(), "cannot serve request", "err", err, "method", r.Method, "path", r.URL.Path)
			switch {
			case errors.Is(err, ErrMissingAuth):
				err = twirp.WrapError(twirp.Unauthenticated.Error("no authentication header present"), err)
			case errors.Is(err, ErrScopeMismatch),
				errors.Is(err, ErrClockSkew), errors.Is(err, ErrBodyTooLarge),
				errors.Is(err, ErrStreamingUnsupported), errors.Is(err, ErrMissingSignedHost):
				// These sentinels describe the caller's own request and carry no
				// internal detail, so surface them: a client cannot correct clock
				// skew it can't distinguish from a scope mismatch.
				err = twirp.WrapError(twirp.InvalidArgument.Error(err.Error()), err)
			case errors.Is(err, ErrUnknownKey), errors.Is(err, ErrUnauthorized),
				errors.Is(err, ErrBodyHash):
				err = twirp.WrapError(twirp.PermissionDenied.Error("invalid authentication header"), err)
			default:
				// Unexpected errors (e.g. the key store being down) are server
				// faults; log the cause but never echo it to the caller.
				slog.ErrorContext(r.Context(), "sigv4 verification failed unexpectedly", "err", err)
				err = twirp.WrapError(twirp.Internal.Error("internal error"), err)
			}
			twirp.WriteError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(withKeyID(r.Context(), keyID)))
	})
}

// Verify checks a single request. On success it returns the access key id of
// the caller. The request body is buffered and reset so downstream handlers
// can read it normally.
func (v *Verifier) Verify(r *http.Request) (string, error) {
	if v.Lookup == nil {
		return "", ErrNotConfigured
	}

	sr, err := parseAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		return "", err
	}

	payloadHash, err := v.resolvePayloadHash(r)
	if err != nil {
		return "", err
	}

	return v.verify(r, sr, payloadHash)
}

// VerifySignature verifies a SigV4 signature over request material without
// possessing the body. payloadHash is placed in the canonical request verbatim
// — it must be exactly what the client signed, sentinel case included; the
// caller must have already confirmed the received body hashes to it — this
// method never reads a body. It is the entry point for central STS validation,
// where the body never reaches the verifier (the verifying service checks it
// locally instead).
//
// host is taken from the host argument, not a Host header, matching how the
// canonical "host" header is derived. headers must carry authorization and
// x-amz-date (or date). payloadHash must be non-empty.
//
// On success it returns the access key id of the caller.
func (v *Verifier) VerifySignature(method, path, query, host string, headers http.Header, payloadHash string) (string, error) {
	if v.Lookup == nil {
		return "", ErrNotConfigured
	}

	sr, err := parseAuthHeader(headers.Get("Authorization"))
	if err != nil {
		return "", err
	}

	if payloadHash == "" {
		return "", errors.New("sigv4: payload hash required to verify a signature")
	}
	if strings.EqualFold(payloadHash, StreamingPayload) {
		return "", ErrStreamingUnsupported
	}

	// Build the URL directly rather than via url.Parse: a forwarded path like
	// "//foo/bar" would parse as a scheme-relative authority, silently moving
	// "foo" into the host and canonicalizing the wrong path.
	unescaped, err := url.PathUnescape(path)
	if err != nil {
		return "", fmt.Errorf("%w: bad path", ErrMissingAuth)
	}

	r := &http.Request{
		Method: method,
		URL:    &url.URL{Path: unescaped, RawPath: path, RawQuery: query},
		Host:   host,
		Header: headers,
	}
	// canonicalHeaderValue derives content-length from r.ContentLength, not the
	// header map, so mirror the forwarded value into the field when a client
	// signed it.
	if cl := headers.Get("Content-Length"); cl != "" {
		if n, err := strconv.ParseInt(cl, 10, 64); err == nil {
			r.ContentLength = n
		}
	}

	return v.verify(r, sr, payloadHash)
}

// verify is the shared core run after the Authorization header is parsed and the
// canonical payload hash is known. It pins the credential scope, requires a
// signed host header, enforces the clock-skew window, resolves the signing
// secret, and compares the recomputed signature in constant time. It never
// touches r.Body.
func (v *Verifier) verify(r *http.Request, sr *signedRequest, payloadHash string) (string, error) {
	now := time.Now
	if v.Now != nil {
		now = v.Now
	}
	skew := v.MaxClockSkew
	if skew == 0 {
		skew = 15 * time.Minute
	}

	// Pin the scope. Without this, a signature valid for some other
	// region/service would be accepted here.
	if sr.scope.region != v.Region || sr.scope.service != v.Service {
		return "", ErrScopeMismatch
	}

	// AWS always signs the host header; requiring it binds the signature to
	// the actual host so a captured request cannot be replayed against a
	// different vhost or port.
	if !slices.Contains(sr.signedHeaders, "host") {
		return "", ErrMissingSignedHost
	}

	amzDate := r.Header.Get("X-Amz-Date")
	if amzDate == "" {
		// Some clients sign the standard Date header instead; convert its
		// RFC 7231 IMF-fixdate form into the AWS timestamp format.
		if d := r.Header.Get("Date"); d != "" {
			dt, derr := http.ParseTime(d)
			if derr != nil {
				return "", fmt.Errorf("%w: bad Date", ErrMissingAuth)
			}
			amzDate = dt.UTC().Format(amzTimeFormat)
		}
	}
	t, err := time.Parse(amzTimeFormat, amzDate)
	if err != nil {
		return "", fmt.Errorf("%w: bad X-Amz-Date", ErrMissingAuth)
	}
	if t.Format(shortDateFormat) != sr.scope.date {
		return "", ErrScopeMismatch
	}
	if d := now().Sub(t); d > skew || d < -skew {
		return "", ErrClockSkew
	}

	secret, err := v.Lookup.Lookup(sr.accessKeyID)
	if err != nil {
		return "", err
	}

	canonReq := v.canonicalRequest(r, sr, payloadHash)
	scopeStr := strings.Join([]string{sr.scope.date, sr.scope.region, sr.scope.service, terminator}, "/")
	hashed := sha256.Sum256([]byte(canonReq))
	stringToSign := strings.Join([]string{
		algorithm,
		amzDate,
		scopeStr,
		hex.EncodeToString(hashed[:]),
	}, "\n")

	key := signingKey(secret, sr.scope.date, sr.scope.region, sr.scope.service)
	want := hex.EncodeToString(hmacSHA256(key, []byte(stringToSign)))

	if subtle.ConstantTimeCompare([]byte(want), []byte(sr.signature)) != 1 {
		return "", ErrUnauthorized
	}
	return sr.accessKeyID, nil
}

// resolvePayloadHash returns the hash string to place in the canonical
// request. When the client sent a concrete hash in x-amz-content-sha256 we
// also re-hash the buffered body and confirm they match, otherwise the
// signed hash proves nothing about the bytes we actually received.
func (v *Verifier) resolvePayloadHash(r *http.Request) (string, error) {
	declared := r.Header.Get("X-Amz-Content-Sha256")

	// Chunked streaming integrity lives in per-chunk signatures in the body
	// framing, which this package does not verify. Reject rather than accept a
	// payload whose integrity is unverified.
	if strings.EqualFold(declared, StreamingPayload) {
		return "", ErrStreamingUnsupported
	}

	// Buffer the body (capped at MaxBodySize) and always reset it so that
	// downstream handlers — and direct Verify callers on error paths — see a
	// re-readable stream rather than a half-consumed one.
	body, err := v.readAndLimitBody(r)
	if err != nil {
		return "", err
	}

	if declared == UnsignedPayload {
		return UnsignedPayload, nil
	}

	sum := sha256.Sum256(body)
	computed := hex.EncodeToString(sum[:])

	if declared == "" {
		// No content hash was signed; fall back to the body hash.
		return computed, nil
	}
	if subtle.ConstantTimeCompare([]byte(strings.ToLower(declared)), []byte(computed)) != 1 {
		return "", ErrBodyHash
	}
	// Use the verified lowercase hash in the canonical request; AWS requires
	// lowercase hex and some clients emit uppercase.
	return computed, nil
}

// readAndLimitBody reads the request body, enforcing MaxBodySize when set, and
// resets r.Body to a reader over the bytes that were read. It resets r.Body on
// every return path so the request body is always re-readable afterwards.
func (v *Verifier) readAndLimitBody(r *http.Request) ([]byte, error) {
	var reader io.Reader = r.Body
	if v.MaxBodySize > 0 {
		reader = io.LimitReader(r.Body, v.MaxBodySize+1)
	}
	body, err := io.ReadAll(reader)
	// The original stream is consumed regardless; surface whatever was read.
	r.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if v.MaxBodySize > 0 && int64(len(body)) > v.MaxBodySize {
		return nil, ErrBodyTooLarge
	}
	return body, nil
}

func (v *Verifier) canonicalRequest(r *http.Request, sr *signedRequest, payloadHash string) string {
	headers := append([]string(nil), sr.signedHeaders...)
	sort.Strings(headers)

	var ch strings.Builder
	for _, h := range headers {
		ch.WriteString(h)
		ch.WriteByte(':')
		ch.WriteString(canonicalHeaderValue(r, h))
		ch.WriteByte('\n')
	}

	return strings.Join([]string{
		r.Method,
		v.canonicalURI(r),
		canonicalQuery(r.URL.Query(), "X-Amz-Signature"),
		ch.String(),
		strings.Join(headers, ";"),
		payloadHash,
	}, "\n")
}

func (v *Verifier) canonicalURI(r *http.Request) string {
	path := r.URL.EscapedPath()
	if path == "" {
		return "/"
	}
	if v.DisablePathEscaping {
		// S3: the on-the-wire encoded path is used directly.
		return path
	}
	// Everything else: encode the already-encoded path a second time.
	return awsURIEncode(path, false)
}

type credentialScope struct {
	date    string
	region  string
	service string
}

type signedRequest struct {
	accessKeyID   string
	scope         credentialScope
	signedHeaders []string
	signature     string
}

func parseAuthHeader(h string) (*signedRequest, error) {
	if !strings.HasPrefix(h, algorithm) {
		return nil, ErrMissingAuth
	}
	rest := strings.TrimSpace(strings.TrimPrefix(h, algorithm))

	var cred, signed, sig string
	for _, part := range strings.Split(rest, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 || kv[0] == "" || kv[1] == "" {
			return nil, ErrMissingAuth
		}
		switch kv[0] {
		case "Credential":
			cred = kv[1]
		case "SignedHeaders":
			signed = kv[1]
		case "Signature":
			sig = kv[1]
		default:
			return nil, ErrMissingAuth
		}
	}
	if cred == "" || signed == "" || sig == "" {
		return nil, ErrMissingAuth
	}

	cp := strings.Split(cred, "/")
	if len(cp) != 5 || cp[4] != terminator {
		return nil, fmt.Errorf("%w: bad credential scope", ErrMissingAuth)
	}
	return &signedRequest{
		accessKeyID:   cp[0],
		scope:         credentialScope{date: cp[1], region: cp[2], service: cp[3]},
		signedHeaders: strings.Split(signed, ";"),
		signature:     sig,
	}, nil
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func signingKey(secret, date, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte(terminator))
}
