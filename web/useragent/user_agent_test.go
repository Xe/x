package useragent

import "testing"

func TestGenUserAgent(t *testing.T) {
	ua := GenUserAgent("test", "https://christine.website")
	if ua == "" {
		t.Fatal("no user agent generated")
	}

	t.Log(ua)
}
