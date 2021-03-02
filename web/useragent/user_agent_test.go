package useragent_test

import (
	"net/http"
	"net/http/httptest"
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
