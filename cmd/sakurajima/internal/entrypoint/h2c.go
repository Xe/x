package entrypoint

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"

	"golang.org/x/net/http2"
)

func newH2CReverseProxy(target *url.URL) (*httputil.ReverseProxy, error) {
	if target == nil {
		return nil, fmt.Errorf("h2c target cannot be nil")
	}
	if target.Host == "" {
		return nil, fmt.Errorf("h2c target must have a host")
	}
	if target.Scheme != "http" && target.Scheme != "h2c" {
		return nil, fmt.Errorf("h2c target must use http:// or h2c:// scheme, got: %s", target.Scheme)
	}

	target.Scheme = "http"

	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host
	}

	// Use h2c transport
	transport := &http2.Transport{
		AllowHTTP: true,
		DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			// Just do plain TCP (h2c)
			return net.Dial(network, addr)
		},
	}

	return &httputil.ReverseProxy{
		Director:  director,
		Transport: transport,
	}, nil
}
