// Package sigv4aclient provides an http.RoundTripper that signs outgoing
// requests with AWS Signature Version 4A. It is the SigV4A counterpart to
// web/middleware/sigv4/sigv4client (which signs classic SigV4 for real AWS
// services): use this package for within.website services verified by
// web/middleware/sigv4a. The API deliberately mirrors sigv4client so
// migrating a call site is a rename.
package sigv4aclient

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"within.website/x/web/middleware/sigv4a"
)

// Config holds the static credential and scope a round tripper signs with.
// All fields are required: SigV4A for within.website services has no
// AWS-style credential chain or default region to fall back on.
type Config struct {
	// Region becomes the X-Amz-Region-Set header value.
	Region string
	// AccessKey and SecretKey are the IAM credential to sign with.
	AccessKey string
	SecretKey string
	// ServiceName is the credential-scope service, e.g. "iam".
	ServiceName string
}

// Valid reports whether every required field is set.
func (c *Config) Valid() error {
	switch {
	case c == nil:
		return fmt.Errorf("sigv4aclient: nil config")
	case c.Region == "":
		return fmt.Errorf("sigv4aclient: Region is required")
	case c.AccessKey == "":
		return fmt.Errorf("sigv4aclient: AccessKey is required")
	case c.SecretKey == "":
		return fmt.Errorf("sigv4aclient: SecretKey is required")
	case c.ServiceName == "":
		return fmt.Errorf("sigv4aclient: ServiceName is required")
	}
	return nil
}

// NewSigV4ARoundTripper returns an http.RoundTripper that signs every
// request per cfg and forwards it to next (http.DefaultTransport when nil).
// The caller's request is cloned and its body buffered for hashing; the
// original is never mutated.
func NewSigV4ARoundTripper(cfg *Config, next http.RoundTripper) (http.RoundTripper, error) {
	if err := cfg.Valid(); err != nil {
		return nil, err
	}
	signer, err := sigv4a.NewSigner(cfg.AccessKey, cfg.SecretKey, cfg.Region, cfg.ServiceName)
	if err != nil {
		return nil, err
	}
	if next == nil {
		next = http.DefaultTransport
	}
	return &roundTripper{signer: signer, next: next}, nil
}

type roundTripper struct {
	signer *sigv4a.Signer
	next   http.RoundTripper
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body.Close()
		r.Body = io.NopCloser(bytes.NewReader(body))
		r.ContentLength = int64(len(body))
		r.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
	}
	if err := rt.signer.Sign(r, body); err != nil {
		return nil, err
	}
	return rt.next.RoundTrip(r)
}
