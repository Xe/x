package useragent_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"within.website/x/web/useragent"
)

func TestGenUserAgent(t *testing.T) {
	ua := useragent.GenUserAgent("test", "https://christine.website")
	if ua == "" {
		t.Fatal("no user agent generated")
	}

	t.Log(ua)
}

func TestGenUserAgentUsesBasename(t *testing.T) {
	ua := useragent.GenUserAgent("test", "https://christine.website")
	if ua == "" {
		t.Fatal("no user agent generated")
	}

	// Verify that the user agent contains the basename, not the full path.
	// The user agent should contain "+<basename>" but not path separators.
	expectedBasename := filepath.Base(os.Args[0])
	if !strings.Contains(ua, "+"+expectedBasename) {
		t.Errorf("user agent should contain %q, but got %q", "+"+expectedBasename, ua)
	}

	// On Unix-like systems, verify that path separators are not present.
	if strings.Contains(ua, "+/") {
		t.Errorf("user agent should not contain full path with separator /, got %q", ua)
	}
}

func TestTransport(t *testing.T) {
	ua := useragent.GenUserAgent("test", "https://example.com")

	http.DefaultTransport = useragent.Transport("test", "https://example.com",
		http.DefaultTransport)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ua != r.Header.Get("User-Agent") {
			t.Errorf("user agent must be %q, but returned %q", ua, r.Header.Get("User-Agent"))
		}
	})

	s := httptest.NewServer(h)
	defer s.Close()

	_, err := http.Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
}
