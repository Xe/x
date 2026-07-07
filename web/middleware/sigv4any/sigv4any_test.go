package sigv4any

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"

	"within.website/x/web/middleware/authctx"
	"within.website/x/web/middleware/sigv4"
	"within.website/x/web/middleware/sigv4a"
)

// These tests sign requests with the real aws-sdk-go-v2 signer (classic) and
// the sigv4a.Signer (SigV4A), then drive them through sigv4any.Verifier the
// same way sigv4/sigv4_test.go round-trips against the reference
// implementation. The two verifiers share one test credential, mirroring how
// iamd's dual verifier resolves both algorithms from the same DAO secret.

const (
	testKey    = "AKIDEXAMPLE"
	testSecret = "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	testRegion = "us-east-1"
	testSvc    = "execute-api"
)

// signingTime is the instant requests are signed at; the verifiers' clocks
// sit 5s later, well inside the default 15-minute skew window.
var signingTime = time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)

func verifyNow() time.Time { return signingTime.Add(5 * time.Second) }

// newVerifier builds a sigv4any.Verifier over classic and SigV4A verifiers
// sharing the one test credential. Either leg can be disabled to exercise the
// nil-verifier configs, and observed, when non-nil, records every algorithm
// Observe is called with.
func newVerifier(disableV4, disableV4A bool, observed *[]string) *Verifier {
	v := &Verifier{}
	if !disableV4 {
		v.V4 = &sigv4.Verifier{
			Region:  testRegion,
			Service: testSvc,
			Lookup: sigv4.LookuperFunc(func(id string) (string, error) {
				if id == testKey {
					return testSecret, nil
				}
				return "", sigv4.ErrUnknownKey
			}),
			Now: verifyNow,
		}
	}
	if !disableV4A {
		v.V4A = &sigv4a.Verifier{
			Region:  testRegion,
			Service: testSvc,
			Lookup: sigv4a.LookuperFunc(func(id string) (string, error) {
				if id == testKey {
					return testSecret, nil
				}
				return "", sigv4a.ErrUnknownKey
			}),
			Now: verifyNow,
		}
	}
	if observed != nil {
		v.Observe = func(algorithm string) { *observed = append(*observed, algorithm) }
	}
	return v
}

// signClassic signs req with the real aws-sdk-go-v2 v4 signer under akid,
// exactly as sigv4/sigv4_test.go does.
func signClassic(t *testing.T, req *http.Request, akid string) {
	t.Helper()
	sum := sha256.Sum256(nil)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)

	creds := aws.Credentials{AccessKeyID: akid, SecretAccessKey: testSecret}
	if err := v4.NewSigner().SignHTTP(req.Context(), creds, req, payloadHash, testSvc, testRegion, signingTime); err != nil {
		t.Fatalf("SDK sign: %v", err)
	}
}

// signV4A signs req with sigv4a.Signer under akid, pinning its clock to
// signingTime so it lines up with the verifier's skew window.
func signV4A(t *testing.T, req *http.Request, akid string) {
	t.Helper()
	signer, err := sigv4a.NewSigner(akid, testSecret, testRegion, testSvc)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	signer.Now = func() time.Time { return signingTime }
	if err := signer.Sign(req, nil); err != nil {
		t.Fatalf("Sign: %v", err)
	}
}

func flipLastHexDigit(auth string) string {
	b := []byte(auth)
	last := len(b) - 1
	if b[last] == '0' {
		b[last] = '1'
	} else {
		b[last] = '0'
	}
	return string(b)
}

func TestMiddleware(t *testing.T) {
	cases := []struct {
		name         string
		disableV4    bool
		disableV4A   bool
		sign         func(t *testing.T, req *http.Request) // nil means leave unsigned
		mutateReq    func(req *http.Request)
		wantStatus   int
		wantObserved string
		wantKeyID    string // empty means no keyID assertion
	}{
		{
			name:         "sigv4a signed request succeeds",
			sign:         func(t *testing.T, req *http.Request) { signV4A(t, req, testKey) },
			wantStatus:   http.StatusNoContent,
			wantObserved: "sigv4a",
			wantKeyID:    testKey,
		},
		{
			name:         "classic signed request succeeds",
			sign:         func(t *testing.T, req *http.Request) { signClassic(t, req, testKey) },
			wantStatus:   http.StatusNoContent,
			wantObserved: "sigv4",
			wantKeyID:    testKey,
		},
		{
			name:         "unsigned request rejected",
			wantStatus:   http.StatusUnauthorized,
			wantObserved: "none",
		},
		{
			name: "unrecognized auth scheme rejected",
			mutateReq: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer nope")
			},
			wantStatus:   http.StatusUnauthorized,
			wantObserved: "none",
		},
		{
			name: "tampered classic signature rejected",
			sign: func(t *testing.T, req *http.Request) { signClassic(t, req, testKey) },
			mutateReq: func(req *http.Request) {
				req.Header.Set("Authorization", flipLastHexDigit(req.Header.Get("Authorization")))
			},
			wantStatus:   http.StatusForbidden,
			wantObserved: "sigv4",
		},
		{
			name: "tampered sigv4a signature rejected",
			sign: func(t *testing.T, req *http.Request) { signV4A(t, req, testKey) },
			mutateReq: func(req *http.Request) {
				req.Header.Set("Authorization", flipLastHexDigit(req.Header.Get("Authorization")))
			},
			wantStatus:   http.StatusForbidden,
			wantObserved: "sigv4a",
		},
		{
			name:         "unknown key rejected under classic",
			sign:         func(t *testing.T, req *http.Request) { signClassic(t, req, "AKIDNOPE") },
			wantStatus:   http.StatusForbidden,
			wantObserved: "sigv4",
		},
		{
			name:         "unknown key rejected under sigv4a",
			sign:         func(t *testing.T, req *http.Request) { signV4A(t, req, "AKIDNOPE") },
			wantStatus:   http.StatusForbidden,
			wantObserved: "sigv4a",
		},
		{
			name:         "V4 disabled: classic signed request falls through to none",
			disableV4:    true,
			sign:         func(t *testing.T, req *http.Request) { signClassic(t, req, testKey) },
			wantStatus:   http.StatusUnauthorized,
			wantObserved: "none",
		},
		{
			name:         "V4 disabled: sigv4a signed request still succeeds",
			disableV4:    true,
			sign:         func(t *testing.T, req *http.Request) { signV4A(t, req, testKey) },
			wantStatus:   http.StatusNoContent,
			wantObserved: "sigv4a",
			wantKeyID:    testKey,
		},
		{
			name:         "V4A disabled: sigv4a signed request falls through to none",
			disableV4A:   true,
			sign:         func(t *testing.T, req *http.Request) { signV4A(t, req, testKey) },
			wantStatus:   http.StatusUnauthorized,
			wantObserved: "none",
		},
		{
			name:         "V4A disabled: classic signed request still succeeds",
			disableV4A:   true,
			sign:         func(t *testing.T, req *http.Request) { signClassic(t, req, testKey) },
			wantStatus:   http.StatusNoContent,
			wantObserved: "sigv4",
			wantKeyID:    testKey,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var observed []string
			v := newVerifier(tc.disableV4, tc.disableV4A, &observed)

			var gotKeyID string
			h := v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotKeyID, _ = authctx.KeyID(r.Context())
				w.WriteHeader(http.StatusNoContent)
			}))

			req := httptest.NewRequest(http.MethodGet, "https://api.example.com/ok", nil)
			if tc.sign != nil {
				tc.sign(t, req)
			}
			if tc.mutateReq != nil {
				tc.mutateReq(req)
			}

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if len(observed) != 1 || observed[0] != tc.wantObserved {
				t.Fatalf("observed = %v, want [%q]", observed, tc.wantObserved)
			}
			if tc.wantKeyID != "" && gotKeyID != tc.wantKeyID {
				t.Fatalf("keyID = %q, want %q", gotKeyID, tc.wantKeyID)
			}
		})
	}
}

// TestMiddleware_NilObserve confirms a Verifier with Observe unset does not
// panic on the happy path, since Middleware must call v.observe
// unconditionally on every branch.
func TestMiddleware_NilObserve(t *testing.T) {
	v := newVerifier(false, false, nil)
	h := v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/ok", nil)
	signClassic(t, req, testKey)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}
