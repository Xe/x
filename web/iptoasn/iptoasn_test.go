package iptoasn

import (
	"net/netip"
	"testing"
)

func TestClient(t *testing.T) {
	cli := New("http://iptoasn.techaro.svc.alrest.xeserv.us")

	if _, err := cli.Lookup(t.Context(), netip.MustParseAddr("1.1.1.1")); err != nil {
		t.Fatal(err)
	}
}
