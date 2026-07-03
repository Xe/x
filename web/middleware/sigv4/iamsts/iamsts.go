// Package iamsts authenticates HTTP requests signed with AWS Signature Version
// 4 by delegating signature verification to a central STS service over Twirp.
//
// It is the "central validation" counterpart to the local sigv4 middleware:
// the verifying service never holds signing secrets. Instead it forwards the
// material it actually received (method, path, query, host, headers) to STS,
// which validates the signature and returns the caller identity. Secrets never
// cross the wire.
//
// Responsibility is split deliberately:
//   - STS verifies the signature, the credential scope, the clock window, and
//     that the key/user are enabled. It confirms the signature covers the
//     declared payload hash.
//   - This middleware confirms that the bytes it actually received hash to the
//     declared x-amz-content-sha256. STS never sees the body, so this check
//     must run where the body is — otherwise a body swapped after signing would
//     go undetected.
//
// Configure the STS client's underlying *http.Client with mTLS so STS can
// authenticate the verifier, and so the channel carrying the forwarded headers
// is trusted.
package iamsts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/timestamppb"

	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
	"within.website/x/web/middleware/sigv4"
)

// Errors returned by Verify. ErrBodyHash and ErrBodyTooLarge are local checks;
// any signature/key/scope/clock failure arrives as a Twirp error from STS.
var (
	ErrBodyHash     = errors.New("iamsts: body does not match x-amz-content-sha256")
	ErrBodyTooLarge = errors.New("iamsts: request body exceeds limit")
)

// Verifier authenticates requests by asking STS to validate their signature.
type Verifier struct {
	// Client is the Twirp client for the STS service. Build it with
	// stsv1.NewSTSServiceProtobufClient (or NewVerifier) and point its
	// underlying *http.Client at STS over mTLS.
	Client stsv1.STSService

	// MaxBodySize caps the bytes buffered to verify the payload hash. Zero
	// means unlimited.
	MaxBodySize int64
}

// NewVerifier returns a Verifier that validates requests against the STS
// service at baseURL. Configure mTLS and timeouts on httpClient; it must not
// be nil (use http.DefaultClient for non-production).
func NewVerifier(baseURL string, httpClient *http.Client, maxBodySize int64) *Verifier {
	return &Verifier{
		Client:      stsv1.NewSTSServiceProtobufClient(baseURL, httpClient),
		MaxBodySize: maxBodySize,
	}
}

// Middleware wraps next so every request is authenticated against STS. On
// success the verified caller is stored in the request context (see Caller).
func (v *Verifier) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caller, err := v.Verify(r)
		if err != nil {
			slog.Debug("can't verify request", "err", err)
			twirp.WriteError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(withCaller(r.Context(), caller)))
	})
}

// Verify authenticates r by asking STS to validate its signature, then confirms
// the received body matches the declared payload hash. On success it returns the
// caller identity. The request body is buffered and reset so downstream
// handlers can read it; on STS failure the body is left untouched.
func (v *Verifier) Verify(r *http.Request) (*Identity, error) {
	headers := toSTHeaders(r.Header)
	// content-length is signed from the ContentLength field, not the header map,
	// and net/http strips it from r.Header on the server side — forward it
	// explicitly so STS can reconstruct the canonical request for body-bearing
	// requests.
	if r.ContentLength > 0 {
		headers = append(headers, &stsv1.Header{Name: "content-length", Value: strconv.FormatInt(r.ContentLength, 10)})
	}

	resp, err := v.Client.GetCallerIdentity(r.Context(), &stsv1.GetCallerIdentityReq{
		Method:  r.Method,
		Path:    r.URL.EscapedPath(),
		Query:   r.URL.RawQuery,
		Host:    r.Host,
		Headers: headers,
	})
	if err != nil {
		return nil, err
	}

	if err := v.verifyBody(r); err != nil {
		return nil, toTwirpErr(err)
	}

	return &Identity{
		AccessKeyID: resp.GetAccessKeyId(),
		User:        resp.GetUser(),
		SignedAt:    resp.GetSignedAt(),
	}, nil
}

// toTwirpErr maps the local body checks to Twirp error codes so a forged or
// oversized body is reported as a client error rather than masked as a server
// fault (which twirp.WriteError would otherwise turn a plain error into).
func toTwirpErr(err error) error {
	switch {
	case errors.Is(err, ErrBodyHash):
		return twirp.NewError(twirp.PermissionDenied, err.Error())
	case errors.Is(err, ErrBodyTooLarge):
		return twirp.NewError(twirp.InvalidArgument, err.Error())
	default:
		return err
	}
}

// verifyBody confirms the received body hashes to the declared
// x-amz-content-sha256. It is skipped for unsigned/streaming payloads and when
// no hash was declared (STS is the authority on whether that is acceptable).
// The body is reset on every return path so it stays re-readable.
func (v *Verifier) verifyBody(r *http.Request) error {
	declared := strings.TrimSpace(r.Header.Get("X-Amz-Content-Sha256"))
	if declared == "" || declared == sigv4.UnsignedPayload || strings.EqualFold(declared, sigv4.StreamingPayload) {
		return nil
	}

	var reader io.Reader = r.Body
	if v.MaxBodySize > 0 {
		reader = io.LimitReader(r.Body, v.MaxBodySize+1)
	}
	body, err := io.ReadAll(reader)
	r.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return err
	}
	if v.MaxBodySize > 0 && int64(len(body)) > v.MaxBodySize {
		return ErrBodyTooLarge
	}

	sum := sha256.Sum256(body)
	if subtle.ConstantTimeCompare([]byte(strings.ToLower(declared)), []byte(hex.EncodeToString(sum[:]))) != 1 {
		return ErrBodyHash
	}
	return nil
}

// Identity is the verified caller stored in the request context on success.
type Identity struct {
	AccessKeyID string
	User        *iamv1.User
	SignedAt    *timestamppb.Timestamp
}

type ctxKey struct{}

func withCaller(ctx context.Context, c *Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, c)
}

// Caller returns the verified identity stored by Middleware, if any.
func Caller(ctx context.Context) (*Identity, bool) {
	c, ok := ctx.Value(ctxKey{}).(*Identity)
	return c, ok
}

// toSTHeaders flattens the received headers into STS Header pairs. Names are
// lowercased; a repeated header yields one entry per value so multi-valued
// headers keep their order. Host is excluded (Go keeps it on r.Host) and is
// forwarded via the request's Host field instead.
func toSTHeaders(h http.Header) []*stsv1.Header {
	out := make([]*stsv1.Header, 0, len(h))
	for name, vals := range h {
		name = strings.ToLower(name)
		for _, val := range vals {
			out = append(out, &stsv1.Header{Name: name, Value: val})
		}
	}
	return out
}
