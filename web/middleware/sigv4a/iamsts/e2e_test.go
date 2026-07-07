package iamsts

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	"within.website/x/web/middleware/sigv4a/sigv4aclient"
)

// TestEndToEnd_SigV4AClientToLocalVerify drives the full chain in its
// deployment shape: sigv4aclient (our own signer) signs an outgoing request,
// the iamsts middleware verifies it locally with a public key fetched over a
// real Twirp round trip from a SigningKeyService stub that derives real keys.
// Any canonicalization disagreement between signer and verifier fails here.
func TestEndToEnd_SigV4AClientToLocalVerify(t *testing.T) {
	fake := &fakeKeys{cacheTTL: 5 * time.Minute, now: time.Now}
	keySrv := httptest.NewServer(stsv1.NewSigningKeyServiceServer(fake))
	defer keySrv.Close()

	v := New(Config{
		BaseURL:     keySrv.URL,
		HTTPClient:  http.DefaultClient,
		Region:      testRegion,
		Service:     testSvc,
		MaxBodySize: 1 << 20,
	})

	var gotBody string
	var gotCaller *Identity
	app := httptest.NewServer(v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotCaller, _ = Caller(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})))
	defer app.Close()

	rt, err := sigv4aclient.NewSigV4ARoundTripper(&sigv4aclient.Config{
		Region:      testRegion,
		AccessKey:   testKey,
		SecretKey:   testSecret,
		ServiceName: testSvc,
	}, nil)
	if err != nil {
		t.Fatalf("round tripper: %v", err)
	}
	client := &http.Client{Transport: rt}

	t.Run("signed POST with body verifies and body survives", func(t *testing.T) {
		resp, err := client.Post(app.URL+"/things?a=1&b=2", "application/json", strings.NewReader(`{"x":1}`))
		if err != nil {
			t.Fatalf("POST: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d, body=%s", resp.StatusCode, body)
		}
		if gotBody != `{"x":1}` {
			t.Errorf("downstream body = %q", gotBody)
		}
		if gotCaller == nil || gotCaller.PrincipalID != "u1" {
			t.Errorf("caller = %+v, want principal u1", gotCaller)
		}
	})

	t.Run("unsigned request rejected", func(t *testing.T) {
		resp, err := http.Get(app.URL + "/things")
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	})

	t.Run("tampered body rejected", func(t *testing.T) {
		// Sign one request, then replay its headers over a different body.
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, app.URL+"/things", strings.NewReader(`{"x":1}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("signed POST: %v", err)
		}
		resp.Body.Close()

		forged, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, app.URL+"/things", strings.NewReader(`{"x":2}`))
		// sigv4aclient clones and signs per attempt, so forge manually: reuse
		// a stale signature against new bytes via a plain client.
		forged.Header = resp.Request.Header.Clone()
		plainResp, err := http.DefaultClient.Do(forged)
		if err != nil {
			t.Fatalf("forged POST: %v", err)
		}
		defer plainResp.Body.Close()
		if plainResp.StatusCode != http.StatusForbidden {
			t.Fatalf("status = %d, want 403 (body swap must not verify)", plainResp.StatusCode)
		}
	})
}
