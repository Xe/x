package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"within.website/x/cmd/iamd/models"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
	"within.website/x/web/middleware/sigv4/iamsts"
	"within.website/x/web/middleware/sigv4/sigv4client"
	"within.website/x/web/middleware/sigv4a"
	"within.website/x/web/middleware/sigv4a/sigv4aclient"
)

const (
	intRegion  = "us-east-1"
	intService = "iam"
)

func quietLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func newDAO(t *testing.T) *models.DAO {
	t.Helper()
	dao, err := models.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return dao
}

// bootstrapCreds bootstraps an admin user on dao and returns its access key id
// and generated secret.
func bootstrapCreds(t *testing.T, dao *models.DAO) (akid, secret string) {
	t.Helper()
	if err := bootstrap(context.Background(), quietLogger(), dao, "root"); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	us, err := dao.ListUsers(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	ks, err := dao.ListKeys(context.Background(), 10, 0, us[0].UUID)
	if err != nil {
		t.Fatalf("ListKeys: %v", err)
	}
	return ks[0].AccessKeyID, ks[0].SecretAccessKey
}

// signedTransport builds a sigv4aclient round tripper that signs requests
// with the given credentials for the test region/service. It is used for
// every transport authenticating to iamd.
func signedTransport(t *testing.T, akid, secret string) http.RoundTripper {
	t.Helper()
	rt, err := sigv4aclient.NewSigV4ARoundTripper(&sigv4aclient.Config{
		Region:      intRegion,
		AccessKey:   akid,
		SecretKey:   secret,
		ServiceName: intService,
	}, nil)
	if err != nil {
		t.Fatalf("round tripper: %v", err)
	}
	return rt
}

// classicSignedTransport builds a sigv4client round tripper that signs
// requests with the given credentials for the test region/service. It is
// used only by the retained classic sigv4/iamsts downstream leg, which
// verifies classic SigV4 rather than SigV4A.
func classicSignedTransport(t *testing.T, akid, secret string) http.RoundTripper {
	t.Helper()
	rt, err := sigv4client.NewSigV4RoundTripper(&sigv4client.Config{
		Region:      intRegion,
		AccessKey:   akid,
		SecretKey:   secret,
		ServiceName: intService,
	}, nil)
	if err != nil {
		t.Fatalf("round tripper: %v", err)
	}
	return rt
}

// TestIntegration_SignedTwirpRoundTrip drives a full signed Twirp call through
// the real iamd mux and rejects unsigned / wrong-key calls at the middleware.
func TestIntegration_SignedTwirpRoundTrip(t *testing.T) {
	dao := newDAO(t)
	akid, secret := bootstrapCreds(t, dao)

	// Re-running bootstrap must be a no-op once users exist.
	if err := bootstrap(context.Background(), quietLogger(), dao, "root"); err != nil {
		t.Fatalf("bootstrap (2nd): %v", err)
	}
	if again, _ := dao.ListUsers(context.Background(), 10, 0); len(again) != 1 {
		t.Fatalf("after 2nd bootstrap, users = %d, want 1", len(again))
	}

	verifier := newVerifier(dao, intRegion, intService, 1<<20)
	mux := newMux(quietLogger(), dao, verifier, 5*time.Minute)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	listURL := srv.URL + iamv1.UserServicePathPrefix + "ListUsers"

	t.Run("signed call reaches handler", func(t *testing.T) {
		client := iamv1.NewUserServiceProtobufClient(srv.URL, &http.Client{Transport: signedTransport(t, akid, secret)})
		if _, err := client.ListUsers(context.Background(), &iamv1.ListUsersReq{Count: 10, Page: 1}); err != nil {
			t.Fatalf("ListUsers: %v", err)
		}
	})

	t.Run("unsigned call is rejected", func(t *testing.T) {
		resp, err := http.Post(listURL, "application/json", nil)
		if err != nil {
			t.Fatalf("POST: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	})

	t.Run("wrong key is rejected", func(t *testing.T) {
		resp, err := (&http.Client{Transport: signedTransport(t, "AKIDNOPE", secret)}).Post(listURL, "application/json", nil)
		if err != nil {
			t.Fatalf("POST: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", resp.StatusCode)
		}
	})
}

// TestUserMiddleware_resolvesCaller checks that UserMiddleware annotates the
// request context with the authenticated caller's DAO user, mirroring
// iamsts.Caller.
func TestUserMiddleware_resolvesCaller(t *testing.T) {
	dao := newDAO(t)
	akid, secret := bootstrapCreds(t, dao)
	verifier := newVerifier(dao, intRegion, intService, 1<<20)

	var gotUUID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, ok := sigv4a.User(r.Context()); ok {
			gotUUID = u.GetId()
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// Verify the signature, then resolve the caller to its user.
	stack := chain(verifier.Middleware, UserMiddleware(dao))
	srv := httptest.NewServer(stack(handler))
	defer srv.Close()

	resp, err := (&http.Client{Transport: signedTransport(t, akid, secret)}).Get(srv.URL + "/anything")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}

	us, _ := dao.ListUsers(context.Background(), 10, 0)
	if gotUUID != us[0].UUID {
		t.Errorf("context user = %q, want %q", gotUUID, us[0].UUID)
	}
}

// TestIntegration_SigningKeyVerifierChain drives the new deployment shape end
// to end against a real iamd: a downstream service verifies incoming SigV4
// requests locally with derived keys fetched from iamd's SigningKeyService,
// authenticating that fetch with its own IAM credential.
func TestIntegration_SigningKeyVerifierChain(t *testing.T) {
	dao := newDAO(t)
	verifierAKID, verifierSecret := bootstrapCreds(t, dao)

	// A second identity acts as the end user calling the downstream service.
	ctx := context.Background()
	endUser, err := dao.CreateUser(ctx, "end-user")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	endKey, err := dao.CreateKey(ctx, endUser, "end user key")
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}

	verifier := newVerifier(dao, intRegion, intService, 1<<20)
	iamd := httptest.NewServer(newMux(quietLogger(), dao, verifier, 5*time.Minute))
	defer iamd.Close()

	// The downstream service: fetches signing keys from iamd (signing those
	// fetches with its own credential) and verifies end-user requests locally.
	downstreamVerifier := iamsts.New(iamsts.Config{
		BaseURL:     iamd.URL,
		HTTPClient:  &http.Client{Transport: signedTransport(t, verifierAKID, verifierSecret)},
		Region:      intRegion,
		Service:     intService,
		MaxBodySize: 1 << 20,
	})
	var gotPrincipal string
	app := httptest.NewServer(downstreamVerifier.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, ok := iamsts.Caller(r.Context()); ok {
			gotPrincipal = c.PrincipalID
		}
		w.WriteHeader(http.StatusNoContent)
	})))
	defer app.Close()

	endUserClient := &http.Client{Transport: classicSignedTransport(t, endKey.AccessKeyID, endKey.SecretAccessKey)}

	t.Run("end user verified locally", func(t *testing.T) {
		resp, err := endUserClient.Get(app.URL + "/resource")
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", resp.StatusCode)
		}
		if gotPrincipal != endUser.UUID {
			t.Errorf("principal = %q, want %q", gotPrincipal, endUser.UUID)
		}
	})

	t.Run("disabled key stops verifying after cache expiry", func(t *testing.T) {
		if err := dao.DisableKey(ctx, endKey.AccessKeyID, "compromised", ""); err != nil {
			t.Fatalf("DisableKey: %v", err)
		}
		// The cached key may still verify (bounded staleness by design); a
		// fresh verifier simulates post-TTL behavior deterministically.
		fresh := iamsts.New(iamsts.Config{
			BaseURL:    iamd.URL,
			HTTPClient: &http.Client{Transport: signedTransport(t, verifierAKID, verifierSecret)},
			Region:     intRegion,
			Service:    intService,
		})
		freshApp := httptest.NewServer(fresh.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})))
		defer freshApp.Close()
		resp, err := endUserClient.Get(freshApp.URL + "/resource")
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("status = %d, want 403 (disabled key)", resp.StatusCode)
		}
	})
}
