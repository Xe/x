package web

import "testing"

func TestGenUserAgent(t *testing.T) {
	ua := genUserAgent()
	if ua == "" {
		t.Fatal("no user agent generated")
	}

	t.Log(ua)
}
