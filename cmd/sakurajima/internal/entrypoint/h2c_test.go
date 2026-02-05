package entrypoint

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func newH2cServer(t *testing.T, h http.Handler) *httptest.Server {
	t.Helper()

	h2s := &http2.Server{}

	srv := httptest.NewServer(h2c.NewHandler(h, h2s))
	t.Cleanup(func() {
		srv.Close()
	})

	return srv
}

func TestH2CReverseProxy(t *testing.T) {
	h := &ackHandler{}

	srv := newH2cServer(t, h)

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	rp, err := newH2CReverseProxy(u)
	if err != nil {
		t.Fatal(err)
	}

	srv2 := httptest.NewServer(rp)
	defer srv2.Close()

	resp, err := srv2.Client().Get(srv2.URL)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("wrong status code from reverse proxy: %d", resp.StatusCode)
	}

	if !h.ack {
		t.Error("h2c handler was not executed")
	}
}

func TestNewH2CReverseProxyValidation(t *testing.T) {
	tests := []struct {
		name        string
		target      *url.URL
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid h2c scheme",
			target:  mustParseURL("h2c://example.com"),
			wantErr: false,
		},
		{
			name:    "valid http scheme",
			target:  mustParseURL("http://example.com"),
			wantErr: false,
		},
		{
			name:        "nil target",
			target:      nil,
			wantErr:     true,
			errContains: "h2c target cannot be nil",
		},
		{
			name:        "empty host",
			target:      mustParseURL("h2c://"),
			wantErr:     true,
			errContains: "h2c target must have a host",
		},
		{
			name:        "invalid scheme - https",
			target:      mustParseURL("https://example.com"),
			wantErr:     true,
			errContains: "h2c target must use http:// or h2c:// scheme, got: https",
		},
		{
			name:        "invalid scheme - ftp",
			target:      mustParseURL("ftp://example.com"),
			wantErr:     true,
			errContains: "h2c target must use http:// or h2c:// scheme, got: ftp",
		},
		{
			name:        "invalid scheme - ws",
			target:      mustParseURL("ws://example.com"),
			wantErr:     true,
			errContains: "h2c target must use http:// or h2c:// scheme, got: ws",
		},
		{
			name: "invalid scheme - unix",
			target: &url.URL{
				Scheme: "unix",
				Host:   "example.com",
			},
			wantErr:     true,
			errContains: "h2c target must use http:// or h2c:// scheme, got: unix",
		},
		{
			name:        "empty scheme",
			target:      mustParseURL("//example.com"),
			wantErr:     true,
			errContains: "h2c target must use http:// or h2c:// scheme, got: ",
		},
		{
			name:    "valid h2c with port",
			target:  mustParseURL("h2c://example.com:8080"),
			wantErr: false,
		},
		{
			name:    "valid http with port",
			target:  mustParseURL("http://example.com:8080"),
			wantErr: false,
		},
		{
			name:    "valid h2c with path",
			target:  mustParseURL("h2c://example.com/path"),
			wantErr: false,
		},
		{
			name:    "valid h2c with query",
			target:  mustParseURL("h2c://example.com?query=value"),
			wantErr: false,
		},
		{
			name:    "valid h2c with fragment",
			target:  mustParseURL("h2c://example.com#fragment"),
			wantErr: false,
		},
		{
			name:    "valid h2c with ipv4",
			target:  mustParseURL("h2c://192.168.1.1"),
			wantErr: false,
		},
		{
			name:    "valid h2c with ipv6",
			target:  mustParseURL("h2c://[2001:db8::1]"),
			wantErr: false,
		},
		{
			name:    "valid h2c with localhost",
			target:  mustParseURL("h2c://localhost"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy, err := newH2CReverseProxy(tt.target)

			if tt.wantErr {
				if err == nil {
					t.Errorf("newH2CReverseProxy() expected error containing %q, got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("newH2CReverseProxy() error = %q, want error containing %q", err.Error(), tt.errContains)
				}
				if proxy != nil {
					t.Errorf("newH2CReverseProxy() expected nil proxy on error, got non-nil")
				}
			} else {
				if err != nil {
					t.Errorf("newH2CReverseProxy() unexpected error: %v", err)
					return
				}
				if proxy == nil {
					t.Errorf("newH2CReverseProxy() expected non-nil proxy, got nil")
				}
			}
		})
	}
}

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}
