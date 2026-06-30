package iamsts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/twitchtv/twirp"

	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
)

// emptySHA is the SHA-256 of the empty string, the payload hash a GET with no
// body is signed with.
const emptySHA = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// fakeSTS is a stub STS service served over a real Twirp endpoint. It returns a
// fixed caller unless code is set, in which case it fails — so the test
// exercises the middleware's Twirp wiring and error mapping, not real crypto.
type fakeSTS struct {
	user      *iamv1.User
	accessKey string
	code      twirp.ErrorCode // non-empty => return this error
}

func (f fakeSTS) GetCallerIdentity(_ context.Context, _ *stsv1.GetCallerIdentityReq) (*stsv1.GetCallerIdentityResp, error) {
	if f.code != "" {
		return nil, twirp.NewError(f.code, "fake denial")
	}
	return &stsv1.GetCallerIdentityResp{
		User:        f.user,
		AccessKeyId: f.accessKey,
	}, nil
}

func newTestVerifier(t *testing.T, fake fakeSTS) (*Verifier, func()) {
	t.Helper()
	srv := httptest.NewServer(stsv1.NewSTSServiceServer(fake))
	return NewVerifier(srv.URL, http.DefaultClient, 1024), srv.Close
}

func TestMiddleware(t *testing.T) {
	user := &iamv1.User{Id: "u1", Name: "tester"}

	t.Run("valid", func(t *testing.T) {
		v, closeSrv := newTestVerifier(t, fakeSTS{user: user, accessKey: "AKID"})
		defer closeSrv()

		var got *Identity
		h := v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got, _ = Caller(r.Context())
			// The body must remain readable downstream.
			if r.URL.Path != "/things" {
				t.Errorf("path = %q, want /things", r.URL.Path)
			}
			w.WriteHeader(http.StatusNoContent)
		}))

		req := httptest.NewRequest(http.MethodGet, "https://svc.example.com/things?a=1", nil)
		req.Header.Set("X-Amz-Content-Sha256", emptySHA)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204; body=%s", rec.Code, rec.Body.String())
		}
		if got == nil || got.AccessKeyID != "AKID" || got.User.GetId() != "u1" {
			t.Fatalf("caller = %+v, want access key AKID / user u1", got)
		}
	})

	t.Run("body hash mismatch rejected", func(t *testing.T) {
		v, closeSrv := newTestVerifier(t, fakeSTS{user: user, accessKey: "AKID"})
		defer closeSrv()

		h := v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("downstream handler should not be reached")
		}))

		req := httptest.NewRequest(http.MethodPost, "https://svc.example.com/", strings.NewReader(`{"x":1}`))
		req.Header.Set("X-Amz-Content-Sha256", "0000000000000000000000000000000000000000000000000000000000000000")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403 (tampered body)", rec.Code)
		}
	})

	t.Run("sts unauthenticated maps to 401", func(t *testing.T) {
		v, closeSrv := newTestVerifier(t, fakeSTS{code: twirp.Unauthenticated})
		defer closeSrv()

		h := v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("downstream handler should not be reached")
		}))

		req := httptest.NewRequest(http.MethodGet, "https://svc.example.com/", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("sts internal error maps to 500", func(t *testing.T) {
		v, closeSrv := newTestVerifier(t, fakeSTS{code: twirp.Internal})
		defer closeSrv()

		h := v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("downstream handler should not be reached")
		}))

		req := httptest.NewRequest(http.MethodGet, "https://svc.example.com/", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
}
