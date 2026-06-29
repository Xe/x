package sigv4

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

// These tests sign requests with the real aws-sdk-go-v2 signer and then verify
// them with this package. That makes them a genuine round-trip against the
// reference implementation rather than a test of our canonicalization against
// our own assumptions: if our canonical request disagrees with the SDK's by a
// single byte, the recomputed signature won't match and the test fails.

const (
	testKey    = "AKIDEXAMPLE"
	testSecret = "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	testRegion = "us-east-1"
	testSvc    = "execute-api"
)

// A verifier whose clock sits 5s after the canonical signing time, well inside
// the default 15-minute skew window.
func newVerifier() *Verifier {
	return &Verifier{
		Region:  testRegion,
		Service: testSvc,
		Lookup: LookuperFunc(func(id string) (string, error) {
			if id == testKey {
				return testSecret, nil
			}
			return "", ErrUnknownKey
		}),
		Now: func() time.Time { return time.Date(2026, 6, 29, 12, 0, 5, 0, time.UTC) },
	}
}

// signWithSDK signs req in place exactly as a real client would: it computes
// the payload hash, sets X-Amz-Content-Sha256, and lets the SDK choose the
// signed-headers set and write the Authorization header.
func signWithSDK(t *testing.T, req *http.Request, body []byte) {
	t.Helper()
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)

	creds := aws.Credentials{AccessKeyID: testKey, SecretAccessKey: testSecret}
	signingTime := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)

	err := v4.NewSigner().SignHTTP(
		req.Context(), creds, req, payloadHash, testSvc, testRegion, signingTime,
	)
	if err != nil {
		t.Fatalf("SDK sign: %v", err)
	}
}

func TestRoundTrip_GET(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/v1/things?b=2&a=1", nil)
	signWithSDK(t, req, nil)

	got, err := newVerifier().Verify(req)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got != testKey {
		t.Fatalf("key = %q, want %q", got, testKey)
	}
}

func TestRoundTrip_POSTWithBody(t *testing.T) {
	body := []byte(`{"hello":"world"}`)
	req := httptest.NewRequest(http.MethodPost, "https://api.example.com/v1/submit", strings.NewReader(string(body)))
	signWithSDK(t, req, body)

	if _, err := newVerifier().Verify(req); err != nil {
		t.Fatalf("verify: %v", err)
	}

	// Verify must leave the body intact for downstream handlers.
	rest, _ := io.ReadAll(req.Body)
	if string(rest) != string(body) {
		t.Fatalf("body not reset: got %q", rest)
	}
}

// Exercises the non-S3 double-encoding branch: the wire path /a%20b/c is
// encoded a second time when building the canonical URI.
func TestRoundTrip_PathWithSpace(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/a%20b/c", nil)
	signWithSDK(t, req, nil)

	if _, err := newVerifier().Verify(req); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

// The signed x-amz-content-sha256 proves the client signed *some* hash; it
// does not prove the body we received matches. Swapping the body after signing
// must be rejected.
func TestTamperedBody(t *testing.T) {
	body := []byte(`{"amount":1}`)
	req := httptest.NewRequest(http.MethodPost, "https://api.example.com/pay", strings.NewReader(string(body)))
	signWithSDK(t, req, body)

	req.Body = io.NopCloser(strings.NewReader(`{"amount":1000000}`))

	if _, err := newVerifier().Verify(req); err == nil {
		t.Fatal("expected rejection of tampered body")
	}
}

func TestTamperedSignature(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/", nil)
	signWithSDK(t, req, nil)
	req.Header.Set("Authorization", flipLastDigit(req.Header.Get("Authorization")))

	if _, err := newVerifier().Verify(req); err != ErrUnauthorized {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

func TestClockSkew(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/", nil)
	signWithSDK(t, req, nil)

	v := newVerifier()
	v.Now = func() time.Time { return time.Date(2026, 6, 29, 13, 0, 0, 0, time.UTC) } // +1h
	if _, err := v.Verify(req); err != ErrClockSkew {
		t.Fatalf("err = %v, want ErrClockSkew", err)
	}
}

func TestUnknownKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/", nil)
	signWithSDK(t, req, nil)

	v := newVerifier()
	v.Lookup = LookuperFunc(func(string) (string, error) { return "", ErrUnknownKey })
	if _, err := v.Verify(req); err != ErrUnknownKey {
		t.Fatalf("err = %v, want ErrUnknownKey", err)
	}
}

// A signature minted for one region/service must not verify against another,
// even though the math would otherwise check out.
func TestWrongScope(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/", nil)
	signWithSDK(t, req, nil)

	v := newVerifier()
	v.Region = "eu-west-1"
	if _, err := v.Verify(req); err != ErrScopeMismatch {
		t.Fatalf("err = %v, want ErrScopeMismatch", err)
	}
}

// TestMiddleware checks the full http.Handler path, including status mapping.
func TestMiddleware(t *testing.T) {
	var gotKey string
	h := newVerifier().Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey, _ = KeyID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	t.Run("valid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://api.example.com/ok", nil)
		signWithSDK(t, req, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", rec.Code)
		}
		if gotKey != testKey {
			t.Fatalf("context key = %q, want %q", gotKey, testKey)
		}
	})

	t.Run("unsigned", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://api.example.com/ok", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})
}

// Streaming payloads are not supported (chunk signatures would need to be
// verified), so the marker must be rejected rather than silently accepted.
func TestStreamingRejected(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/", nil)
	signWithSDK(t, req, nil)
	req.Header.Set("X-Amz-Content-Sha256", "STREAMING-AWS4-HMAC-SHA256-PAYLOAD")

	if _, err := newVerifier().Verify(req); err != ErrStreamingUnsupported {
		t.Fatalf("err = %v, want ErrStreamingUnsupported", err)
	}
}

// host must be a signed header; otherwise a captured request can be replayed
// against any other host sharing the same region/service verifier.
func TestUnsignedHostRejected(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/", nil)
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 "+
		"Credential=AKIDEXAMPLE/20260629/us-east-1/execute-api/aws4_request, "+
		"SignedHeaders=x-amz-date, Signature=0000")

	if _, err := newVerifier().Verify(req); err != ErrMissingSignedHost {
		t.Fatalf("err = %v, want ErrMissingSignedHost", err)
	}
}

// A Verifier without a Lookup must return an error, not panic.
func TestNilLookuper(t *testing.T) {
	v := &Verifier{Region: testRegion, Service: testSvc}
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/", nil)
	if _, err := v.Verify(req); err != ErrNotConfigured {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

// Non-sentinel errors (e.g. a backing-store outage) must surface as 500, not be
// misreported as a 403 authorization denial.
func TestUnknownErrorMapsTo500(t *testing.T) {
	v := newVerifier()
	v.Lookup = LookuperFunc(func(string) (string, error) {
		return "", errors.New("database is down")
	})
	h := v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("downstream handler should not be reached")
	}))

	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/", nil)
	signWithSDK(t, req, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

// The MaxBodySize cap applies even when the payload is declared unsigned.
func TestUnsignedPayloadRespectsMaxBodySize(t *testing.T) {
	body := strings.Repeat("x", 64)
	v := newVerifier()
	v.MaxBodySize = 16

	req := httptest.NewRequest(http.MethodPost, "https://api.example.com/",
		strings.NewReader(body))
	signWithSDK(t, req, []byte(body))
	req.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")

	if _, err := v.Verify(req); err != ErrBodyTooLarge {
		t.Fatalf("err = %v, want ErrBodyTooLarge", err)
	}
}

func flipLastDigit(auth string) string {
	b := []byte(auth)
	last := len(b) - 1
	if b[last] == '0' {
		b[last] = '1'
	} else {
		b[last] = '0'
	}
	return string(b)
}
