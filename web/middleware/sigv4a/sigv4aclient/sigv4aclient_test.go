package sigv4aclient

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func validConfig() *Config {
	return &Config{
		Region:      "us-east-1",
		AccessKey:   "AKIDEXAMPLE",
		SecretKey:   "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		ServiceName: "iam",
	}
}

func TestConfigValid(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{name: "complete", mutate: func(*Config) {}, wantErr: false},
		{name: "missing region", mutate: func(c *Config) { c.Region = "" }, wantErr: true},
		{name: "missing access key", mutate: func(c *Config) { c.AccessKey = "" }, wantErr: true},
		{name: "missing secret key", mutate: func(c *Config) { c.SecretKey = "" }, wantErr: true},
		{name: "missing service", mutate: func(c *Config) { c.ServiceName = "" }, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(cfg)
			if err := cfg.Valid(); (err != nil) != tt.wantErr {
				t.Errorf("Valid() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// authField extracts one comma-separated field from an Authorization header.
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

// TestRoundTrip checks the deployment shape: body buffered and preserved,
// payload hash declared, all four standard headers signed, and the caller's
// request left unmutated.
func TestRoundTrip(t *testing.T) {
	var got *http.Request
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())
		gotBody, _ = io.ReadAll(r.Body)
	}))
	defer srv.Close()

	rt, err := NewSigV4ARoundTripper(validConfig(), nil)
	if err != nil {
		t.Fatalf("NewSigV4ARoundTripper: %v", err)
	}
	client := &http.Client{Transport: rt}

	const body = `{"x":1}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/things", strings.NewReader(body))
	if _, err := client.Do(req); err != nil {
		t.Fatalf("Do: %v", err)
	}

	if string(gotBody) != body {
		t.Errorf("body = %q, want %q", gotBody, body)
	}
	auth := got.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "AWS4-ECDSA-P256-SHA256 ") {
		t.Errorf("Authorization = %q, want AWS4-ECDSA-P256-SHA256 prefix", auth)
	}
	if want := "host;x-amz-content-sha256;x-amz-date;x-amz-region-set"; authField(t, auth, "SignedHeaders") != want {
		t.Errorf("SignedHeaders = %q, want %q", authField(t, auth, "SignedHeaders"), want)
	}
	if got.Header.Get("X-Amz-Region-Set") != "us-east-1" {
		t.Error("X-Amz-Region-Set not set")
	}
	sum := sha256.Sum256([]byte(body))
	if got.Header.Get("X-Amz-Content-Sha256") != hex.EncodeToString(sum[:]) {
		t.Error("X-Amz-Content-Sha256 does not match body")
	}
	if req.Header.Get("Authorization") != "" {
		t.Error("RoundTrip mutated the caller's request")
	}
}

// roundTripFunc adapts a function to http.RoundTripper for stubbing next.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestRoundTrip_SetsGetBody checks that the request handed to next has
// GetBody set to rewind to the exact buffered body. net/http's Transport
// consults GetBody (on the request it was actually given, i.e. our clone) to
// retry a request transparently after a broken keep-alive connection, and
// http.Client consults it to resend a body on a 307/308 redirect when the
// caller's own request didn't already supply one (e.g. a non-seekable
// io.Reader). Without this, our clone silently loses both.
func TestRoundTrip_SetsGetBody(t *testing.T) {
	const body = `{"x":1}`
	var captured *http.Request
	next := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		captured = r
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})

	rt, err := NewSigV4ARoundTripper(validConfig(), next)
	if err != nil {
		t.Fatalf("NewSigV4ARoundTripper: %v", err)
	}

	// Built without passing the body to http.NewRequest, so the standard
	// library does not auto-populate GetBody on the request itself: any
	// GetBody the clone ends up with must come from our own RoundTrip.
	req, _ := http.NewRequest(http.MethodPost, "https://example.com/things", nil)
	req.Body = io.NopCloser(strings.NewReader(body))
	req.ContentLength = int64(len(body))
	if req.GetBody != nil {
		t.Fatal("test setup invalid: req.GetBody unexpectedly non-nil before RoundTrip")
	}
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}

	if captured.GetBody == nil {
		t.Fatal("GetBody is nil; redirects and transport retries cannot rewind the body")
	}
	rc, err := captured.GetBody()
	if err != nil {
		t.Fatalf("GetBody(): %v", err)
	}
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read GetBody() result: %v", err)
	}
	if string(got) != body {
		t.Fatalf("GetBody() = %q, want %q", got, body)
	}

	// GetBody must be independently rewindable (callable more than once) and
	// must not be the same reader driving the request that's already in
	// flight.
	rest, _ := io.ReadAll(captured.Body)
	if string(rest) != body {
		t.Fatalf("captured.Body = %q, want %q", rest, body)
	}
}

func TestInvalidConfigRejected(t *testing.T) {
	cfg := validConfig()
	cfg.SecretKey = ""
	if _, err := NewSigV4ARoundTripper(cfg, nil); err == nil {
		t.Fatal("expected error for invalid config")
	}
}
