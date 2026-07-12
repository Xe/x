package sigv4client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	testKey    = "AKIDEXAMPLE"
	testSecret = "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	testSvc    = "execute-api"
)

// When Config.Region is empty, the region resolved from the default credentials
// chain (here, the env) must be the one used to sign — not the empty string.
func TestRegionFromDefaultChain(t *testing.T) {
	t.Setenv("AWS_DEFAULT_REGION", "us-west-2")
	t.Setenv("AWS_REGION", "us-west-2")

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rt, err := NewSigV4RoundTripper(&Config{
		Region:      "", // rely on the env-resolved region
		AccessKey:   testKey,
		SecretKey:   testSecret,
		ServiceName: testSvc,
	}, nil)
	if err != nil {
		t.Fatalf("new round tripper: %v", err)
	}

	resp, err := (&http.Client{Transport: rt}).Get(srv.URL + "/foo")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()

	if !strings.Contains(gotAuth, "/us-west-2/") {
		t.Fatalf("Authorization %q does not contain the env-resolved region us-west-2", gotAuth)
	}
}

// RoundTrip must not mutate the caller's request (beyond consuming its body);
// in particular the URL path cleaning must happen on the clone, not on req.
func TestRoundTripDoesNotMutateRequest(t *testing.T) {
	rt, err := NewSigV4RoundTripper(&Config{
		Region:      "us-east-1",
		AccessKey:   testKey,
		SecretKey:   testSecret,
		ServiceName: testSvc,
	}, nil)
	if err != nil {
		t.Fatalf("new round tripper: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/a/../b", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	origPath := req.URL.Path

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("round trip: %v", err)
	}
	resp.Body.Close()

	if req.URL.Path != origPath {
		t.Fatalf("RoundTrip mutated req.URL.Path: got %q, want %q", req.URL.Path, origPath)
	}
}
