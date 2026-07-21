package sigv4a

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/twitchtv/twirp"
	"within.website/x/web/middleware/internal/awssig"
)

const (
	terminator      = awssig.Terminator
	amzTimeFormat   = awssig.AmzTimeFormat
	shortDateFormat = awssig.ShortDateFormat
)

// Payload-hash sentinels. UnsignedPayload matches exactly (AWS defines it
// case-sensitively); any STREAMING-* sentinel is rejected inside
// awssig.ResolvePayloadHash.
const (
	UnsignedPayload  = awssig.UnsignedPayload
	StreamingPayload = "STREAMING-AWS4-ECDSA-P256-SHA256-PAYLOAD"
)

// Errors returned by Verify. Callers typically map ErrUnauthorized and
// ErrUnknownKey to 403 and the rest to 400.
var (
	ErrMissingAuth          = errors.New("sigv4a: missing or malformed Authorization")
	ErrMalformedAuth        = errors.New("sigv4a: malformed authentication header")
	ErrMissingSignedHost    = errors.New("sigv4a: host must appear in SignedHeaders")
	ErrMissingRegionSet     = errors.New("sigv4a: x-amz-region-set must appear in SignedHeaders")
	ErrUnknownKey           = errors.New("sigv4a: unknown access key id")
	ErrClockSkew            = errors.New("sigv4a: request time outside allowed skew")
	ErrScopeMismatch        = errors.New("sigv4a: credential scope does not match")
	ErrStreamingUnsupported = awssig.ErrStreamingUnsupported
	ErrBodyHash             = awssig.ErrBodyHash
	ErrUnauthorized         = errors.New("sigv4a: signature mismatch")
	ErrBodyTooLarge         = awssig.ErrBodyTooLarge
	ErrNotConfigured        = errors.New("sigv4a: neither Verifier.Lookup nor Verifier.KeyLookup is set")
)

// DefaultMaxBodySize is the byte cap applied to request bodies when
// Verifier.MaxBodySize is left zero. 10 MiB is large enough for the
// control-plane requests this middleware protects while bounding the
// memory an attacker can force it to allocate.
const DefaultMaxBodySize int64 = 10 << 20 // 10 MiB

// Verifier validates SigV4A-signed requests for a single region/service.
type Verifier struct {
	// Region must be covered by the request's signed X-Amz-Region-Set;
	// Service must match the credential scope.
	Region  string
	Service string

	// Lookup resolves an access key id to its secret. Return ErrUnknownKey
	// for unknown keys. This is the one piece you must supply.
	Lookup Lookuper

	// KeyLookup resolves an access key id to its ECDSA public key. When set
	// it takes precedence over Lookup, and the verifier never sees the raw
	// secret — this is how services that must not hold secrets verify
	// locally (see web/middleware/sigv4a/iamsts). Exactly one of Lookup or
	// KeyLookup must be set.
	KeyLookup PublicKeyLookuper

	// MaxClockSkew bounds how far the request's X-Amz-Date may be from now.
	// Defaults to 15 minutes (matching AWS) when zero.
	MaxClockSkew time.Duration

	// DisablePathEscaping selects S3-style canonicalization, where the path
	// is used as-is rather than being URI-encoded a second time. Set true
	// when emulating S3; leave false for every other AWS service.
	DisablePathEscaping bool

	// MaxBodySize caps how many bytes of the body will be buffered to verify
	// the payload hash. Zero means DefaultMaxBodySize (10 MiB). Requests that
	// exceed it are rejected with ErrBodyTooLarge.
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
			twirp.WriteError(w, TwirpError(r.Context(), err))
			return
		}
		next.ServeHTTP(w, r.WithContext(withKeyID(r.Context(), keyID)))
	})
}

// TwirpError maps a verification error to the twirp error middlewares write
// to clients. Sentinels that describe the caller's own request keep their
// message; key and signature failures collapse to one opaque message so a
// probe cannot distinguish unknown, disabled, and mis-signed credentials.
// Unexpected errors are logged with their cause and surfaced as an opaque
// internal error.
func TwirpError(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, ErrMissingAuth):
		return twirp.WrapError(twirp.Unauthenticated.Error("no authentication header present"), err)
	case errors.Is(err, ErrMalformedAuth):
		return twirp.WrapError(twirp.InvalidArgument.Error("malformed authentication header"), err)
	case errors.Is(err, ErrScopeMismatch),
		errors.Is(err, ErrClockSkew), errors.Is(err, ErrBodyTooLarge),
		errors.Is(err, ErrStreamingUnsupported), errors.Is(err, ErrMissingSignedHost),
		errors.Is(err, ErrMissingRegionSet):
		// These sentinels describe the caller's own request and carry no
		// internal detail, so surface them: a client cannot correct clock
		// skew it can't distinguish from a scope mismatch.
		return twirp.WrapError(twirp.InvalidArgument.Error(err.Error()), err)
	case errors.Is(err, ErrUnknownKey), errors.Is(err, ErrUnauthorized),
		errors.Is(err, ErrBodyHash):
		return twirp.WrapError(twirp.PermissionDenied.Error("invalid authentication header"), err)
	default:
		// Unexpected errors (e.g. the key store being down) are server
		// faults; log the cause but never echo it to the caller.
		slog.ErrorContext(ctx, "sigv4a verification failed unexpectedly", "err", err)
		return twirp.WrapError(twirp.Internal.Error("internal error"), err)
	}
}

// Verify checks a single request. On success it returns the access key id of
// the caller. The request body is buffered and reset so downstream handlers
// can read it normally.
func (v *Verifier) Verify(r *http.Request) (string, error) {
	if v.Lookup == nil && v.KeyLookup == nil {
		return "", ErrNotConfigured
	}

	sr, err := parseAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		return "", err
	}

	maxBodySize := v.MaxBodySize
	if maxBodySize == 0 {
		maxBodySize = DefaultMaxBodySize
	}
	payloadHash, err := awssig.ResolvePayloadHash(r, maxBodySize)
	if err != nil {
		return "", err
	}

	return v.verify(r, sr, payloadHash)
}

// verify is the shared core run after the Authorization header is parsed and the
// canonical payload hash is known. It pins the credential scope, requires a
// signed host header, enforces the clock-skew window, resolves the signing
// public key, and verifies the ECDSA signature. It never touches r.Body.
func (v *Verifier) verify(r *http.Request, sr *signedRequest, payloadHash string) (string, error) {
	now := time.Now
	if v.Now != nil {
		now = v.Now
	}
	skew := v.MaxClockSkew
	if skew == 0 {
		skew = 15 * time.Minute
	}

	// Pin the scope. Without this, a signature valid for some other service
	// would be accepted here. Region is not part of the SigV4A credential
	// scope; it is checked below through the signed X-Amz-Region-Set.
	if sr.scope.service != v.Service {
		return "", ErrScopeMismatch
	}

	// AWS always signs the host header; requiring it binds the signature to
	// the actual host so a captured request cannot be replayed against a
	// different vhost or port.
	if !slices.Contains(sr.signedHeaders, "host") {
		return "", ErrMissingSignedHost
	}

	// The region set feeds the scope decision, so it must be signed —
	// otherwise a relay could rewrite the audience of a captured signature.
	if !slices.Contains(sr.signedHeaders, "x-amz-region-set") {
		return "", ErrMissingRegionSet
	}
	if !regionSetMatches(r.Header.Get("X-Amz-Region-Set"), v.Region) {
		return "", ErrScopeMismatch
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

	var pub *ecdsa.PublicKey
	if v.KeyLookup != nil {
		pub, err = v.KeyLookup.LookupPublicKey(r.Context(), sr.accessKeyID)
	} else {
		var secret string
		secret, err = v.Lookup.Lookup(sr.accessKeyID)
		if err == nil {
			var priv *ecdsa.PrivateKey
			priv, err = DeriveKeyPair(sr.accessKeyID, secret)
			if err == nil {
				pub = &priv.PublicKey
			}
		}
	}
	if err != nil {
		return "", err
	}
	if pub == nil {
		// A lookuper that returns no key and no error must read as
		// unknown-key, never reach VerifyASN1: it dereferences pub.Curve
		// unconditionally and would panic on a nil key, turning a lookuper
		// bug into a DoS of this auth middleware.
		return "", ErrUnknownKey
	}

	canonReq := v.canonicalRequest(r, sr, payloadHash)
	scopeStr := strings.Join([]string{sr.scope.date, sr.scope.service, terminator}, "/")
	hashed := sha256.Sum256([]byte(canonReq))
	stringToSign := strings.Join([]string{
		algorithm,
		amzDate,
		scopeStr,
		hex.EncodeToString(hashed[:]),
	}, "\n")

	sig, err := hex.DecodeString(sr.signature)
	if err != nil {
		return "", ErrUnauthorized
	}
	digest := sha256.Sum256([]byte(stringToSign))
	if !ecdsa.VerifyASN1(pub, digest[:], sig) {
		return "", ErrUnauthorized
	}
	return sr.accessKeyID, nil
}

func (v *Verifier) canonicalRequest(r *http.Request, sr *signedRequest, payloadHash string) string {
	headers := append([]string(nil), sr.signedHeaders...)
	sort.Strings(headers)
	return awssig.BuildCanonicalRequest(r, headers, payloadHash, v.DisablePathEscaping)
}

type credentialScope struct {
	date    string
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
			return nil, ErrMalformedAuth
		}
		switch kv[0] {
		case "Credential":
			if cred != "" {
				return nil, ErrMalformedAuth
			}
			cred = kv[1]
		case "SignedHeaders":
			if signed != "" {
				return nil, ErrMalformedAuth
			}
			signed = kv[1]
		case "Signature":
			if sig != "" {
				return nil, ErrMalformedAuth
			}
			sig = kv[1]
		default:
			return nil, ErrMalformedAuth
		}
	}
	if cred == "" || signed == "" || sig == "" {
		return nil, ErrMalformedAuth
	}

	cp := strings.Split(cred, "/")
	if len(cp) != 4 || cp[3] != terminator {
		return nil, fmt.Errorf("%w: bad credential scope", ErrMalformedAuth)
	}
	return &signedRequest{
		accessKeyID:   cp[0],
		scope:         credentialScope{date: cp[1], service: cp[2]},
		signedHeaders: strings.Split(signed, ";"),
		signature:     sig,
	}, nil
}

// Credential is the parsed Credential= component of a SigV4A Authorization
// header: the access key id plus the literal, unnormalized scope strings.
type Credential struct {
	AccessKeyID string
	Date        string
	Service     string
}

// ParseCredential extracts the credential scope from an AWS4-ECDSA-P256-SHA256
// Authorization header value. It returns ErrMissingAuth for anything
// malformed, matching Verify.
func ParseCredential(authorization string) (*Credential, error) {
	sr, err := parseAuthHeader(authorization)
	if err != nil {
		return nil, err
	}
	return &Credential{
		AccessKeyID: sr.accessKeyID,
		Date:        sr.scope.date,
		Service:     sr.scope.service,
	}, nil
}
