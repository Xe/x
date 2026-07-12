package sigv4a

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"within.website/x/web/middleware/internal/awssig"
)

// authField extracts one comma-separated field ("Credential",
// "SignedHeaders", "Signature") from an Authorization header value.
func authField(t *testing.T, auth, field string) string {
	t.Helper()
	for _, part := range strings.Split(auth, " ") {
		part = strings.TrimSuffix(strings.TrimSpace(part), ",")
		if v, ok := strings.CutPrefix(part, field+"="); ok {
			return v
		}
	}
	t.Fatalf("no %s= in %q", field, auth)
	return ""
}

// TestSigner_Vectors signs each AWS test-suite request and checks every
// signing artifact against the vector: the canonical request and the
// Credential/SignedHeaders fields byte-for-byte, and the signature by ECDSA
// verification against the vector's public key over the vector's own
// string-to-sign. Signature bytes themselves are never compared: ECDSA is
// randomized (the suite itself ships two different valid signatures per
// vector).
func TestSigner_Vectors(t *testing.T) {
	for _, dir := range vectorDirs(t) {
		t.Run(dir, func(t *testing.T) {
			vc := loadVectorContext(t, dir)
			r, body := parseVectorRequest(t, readVectorFile(t, dir, "request.txt"))
			sum := sha256.Sum256(body)
			payloadHash := hex.EncodeToString(sum[:])

			s, err := NewSigner(vc.Credentials.AccessKeyID, vc.Credentials.SecretAccessKey, vc.Region, vc.Service)
			if err != nil {
				t.Fatalf("NewSigner: %v", err)
			}
			s.Now = func() time.Time { return vc.signingTime(t) }

			wantSigned := authField(t, readVectorFile(t, dir, "header-signed-request.txt"), "SignedHeaders")
			signedHeaders := strings.Split(wantSigned, ";")
			if err := s.sign(r, signedHeaders, payloadHash); err != nil {
				t.Fatalf("sign: %v", err)
			}

			sorted := append([]string(nil), signedHeaders...)
			sort.Strings(sorted)
			if got, want := awssig.BuildCanonicalRequest(r, sorted, payloadHash, false),
				readVectorFile(t, dir, "header-canonical-request.txt"); got != want {
				t.Errorf("canonical request mismatch:\ngot:\n%s\nwant:\n%s", got, want)
			}

			auth := r.Header.Get("Authorization")
			wantCred := authField(t, readVectorFile(t, dir, "header-signed-request.txt"), "Credential")
			if got := authField(t, auth, "Credential"); got != wantCred {
				t.Errorf("Credential = %q, want %q", got, wantCred)
			}
			if got := authField(t, auth, "SignedHeaders"); got != wantSigned {
				t.Errorf("SignedHeaders = %q, want %q", got, wantSigned)
			}

			sig, err := hex.DecodeString(authField(t, auth, "Signature"))
			if err != nil {
				t.Fatalf("signature is not hex: %v", err)
			}
			digest := sha256.Sum256([]byte(readVectorFile(t, dir, "header-string-to-sign.txt")))
			if !ecdsa.VerifyASN1(vectorPublicKey(t, dir), digest[:], sig) {
				t.Error("signature does not verify against the vector string-to-sign and public key")
			}
		})
	}
}

func testSigner(t *testing.T) *Signer {
	t.Helper()
	s, err := NewSigner("AKIDEXAMPLE", "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY", "us-east-1", "execute-api")
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	s.Now = func() time.Time { return time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC) }
	return s
}

func testVerifier() *Verifier {
	return &Verifier{
		Region:  "us-east-1",
		Service: "execute-api",
		Lookup: LookuperFunc(func(id string) (string, error) {
			if id == "AKIDEXAMPLE" {
				return "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY", nil
			}
			return "", ErrUnknownKey
		}),
		Now: func() time.Time { return time.Date(2026, 6, 29, 12, 0, 5, 0, time.UTC) },
	}
}

// TestRoundTrip_SignVerify closes the loop: our signer's output verifies
// with our verifier, bodies survive, and tampering is caught.
func TestRoundTrip_SignVerify(t *testing.T) {
	t.Run("GET with query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://api.example.com/v1/things?b=2&a=1&list=1&list-type=2", nil)
		if err := testSigner(t).Sign(req, nil); err != nil {
			t.Fatalf("Sign: %v", err)
		}
		got, err := testVerifier().Verify(req)
		if err != nil {
			t.Fatalf("verify: %v", err)
		}
		if got != "AKIDEXAMPLE" {
			t.Fatalf("key = %q", got)
		}
	})

	t.Run("POST body verifies and survives", func(t *testing.T) {
		body := []byte(`{"hello":"world"}`)
		req := httptest.NewRequest(http.MethodPost, "https://api.example.com/v1/submit", strings.NewReader(string(body)))
		if err := testSigner(t).Sign(req, body); err != nil {
			t.Fatalf("Sign: %v", err)
		}
		if _, err := testVerifier().Verify(req); err != nil {
			t.Fatalf("verify: %v", err)
		}
		rest, _ := io.ReadAll(req.Body)
		if string(rest) != string(body) {
			t.Fatalf("body not reset: got %q", rest)
		}
	})

	t.Run("tampered body rejected", func(t *testing.T) {
		body := []byte(`{"amount":1}`)
		req := httptest.NewRequest(http.MethodPost, "https://api.example.com/pay", strings.NewReader(string(body)))
		if err := testSigner(t).Sign(req, body); err != nil {
			t.Fatalf("Sign: %v", err)
		}
		req.Body = io.NopCloser(strings.NewReader(`{"amount":1000000}`))
		if _, err := testVerifier().Verify(req); err == nil {
			t.Fatal("expected rejection of tampered body")
		}
	})

	t.Run("path with space double-encodes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://api.example.com/a%20b/c", nil)
		if err := testSigner(t).Sign(req, nil); err != nil {
			t.Fatalf("Sign: %v", err)
		}
		if _, err := testVerifier().Verify(req); err != nil {
			t.Fatalf("verify: %v", err)
		}
	})
}

// TestMiddleware checks the full http.Handler path, including status mapping.
func TestMiddleware(t *testing.T) {
	var gotKey string
	h := testVerifier().Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey, _ = KeyID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	t.Run("valid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://api.example.com/ok", nil)
		if err := testSigner(t).Sign(req, nil); err != nil {
			t.Fatalf("Sign: %v", err)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", rec.Code)
		}
		if gotKey != "AKIDEXAMPLE" {
			t.Fatalf("context key = %q", gotKey)
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
