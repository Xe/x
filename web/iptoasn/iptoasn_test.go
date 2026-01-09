package iptoasn

import (
	"net"
	"net/netip"
	"net/url"
	"testing"
)

func TestClient(t *testing.T) {
	baseURL := "http://iptoasn.techaro.svc.alrest.xeserv.us"

	u, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("failed to parse base URL: %v", err)
	}

	if _, err := net.LookupHost(u.Hostname()); err != nil {
		t.Skipf("hostname %s does not resolve, skipping test: %v", u.Hostname(), err)
	}

	cli := New(baseURL)

	if _, err := cli.Lookup(t.Context(), netip.MustParseAddr("1.1.1.1")); err != nil {
		t.Fatal(err)
	}
}
