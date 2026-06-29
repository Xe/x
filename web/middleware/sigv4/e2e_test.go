package sigv4

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"within.website/x/web/middleware/sigv4/sigv4client"
)

// TestEndToEnd_SigV4Client drives a full round trip: sigv4client signs an
// outgoing request with the AWS SigV4 procedure, and this package's Verifier
// middleware validates it on the server side. Because the signer and verifier
// are independent implementations, a single byte of disagreement in
// canonicalization would make the recomputed signature mismatch and fail.
func TestEndToEnd_SigV4Client(t *testing.T) {
	v := &Verifier{
		Region:  testRegion,
		Service: testSvc,
		Lookup: LookuperFunc(func(id string) (string, error) {
			if id == testKey {
				return testSecret, nil
			}
			return "", ErrUnknownKey
		}),
	}

	var gotKey, gotBody string
	srv := httptest.NewServer(v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey, _ = KeyID(r.Context())
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusNoContent)
	})))
	defer srv.Close()

	rt, err := sigv4client.NewSigV4RoundTripper(&sigv4client.Config{
		Region:      testRegion,
		AccessKey:   testKey,
		SecretKey:   testSecret,
		ServiceName: testSvc,
	}, nil)
	if err != nil {
		t.Fatalf("new round tripper: %v", err)
	}
	client := &http.Client{Transport: rt}

	t.Run("GET", func(t *testing.T) {
		gotKey, gotBody = "", ""
		resp, err := client.Get(srv.URL + "/v1/things?b=2&a=1")
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", resp.StatusCode)
		}
		if gotKey != testKey {
			t.Fatalf("context key = %q, want %q", gotKey, testKey)
		}
	})

	t.Run("POST", func(t *testing.T) {
		gotKey, gotBody = "", ""
		body := `{"hello":"world"}`
		resp, err := client.Post(srv.URL+"/v1/submit", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("POST: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", resp.StatusCode)
		}
		if gotKey != testKey {
			t.Fatalf("context key = %q, want %q", gotKey, testKey)
		}
		if gotBody != body {
			t.Fatalf("body = %q, want %q", gotBody, body)
		}
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		badRT, err := sigv4client.NewSigV4RoundTripper(&sigv4client.Config{
			Region:      testRegion,
			AccessKey:   "AKIDWRONG",
			SecretKey:   testSecret,
			ServiceName: testSvc,
		}, nil)
		if err != nil {
			t.Fatalf("new round tripper: %v", err)
		}
		resp, err := (&http.Client{Transport: badRT}).Get(srv.URL + "/v1/things")
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", resp.StatusCode)
		}
	})
}
