package sigv4a

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// A body larger than an explicitly-set MaxBodySize must be rejected with
// ErrBodyTooLarge rather than being buffered in full.
func TestMaxBodySizeExplicitExceeded(t *testing.T) {
	vc := loadVectorContext(t, "get-vanilla")
	req, _ := parseVectorRequest(t, readVectorFile(t, "get-vanilla", "header-signed-request.txt"))
	req.Body = io.NopCloser(bytes.NewReader(bytes.Repeat([]byte("a"), 1024)))
	req.ContentLength = 1024
	req.Header.Set("X-Amz-Content-Sha256", "")

	v := vectorVerifier(t, vc)
	v.MaxBodySize = 16
	_, err := v.Verify(req)
	if !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("err = %v, want ErrBodyTooLarge", err)
	}
}

// A body within an explicitly-set MaxBodySize proceeds past the size check
// (and fails later for signature reasons, which is enough to prove the cap
// was honored rather than treated as zero/unlimited).
func TestMaxBodySizeExplicitWithin(t *testing.T) {
	vc := loadVectorContext(t, "get-vanilla")
	req, _ := parseVectorRequest(t, readVectorFile(t, "get-vanilla", "header-signed-request.txt"))
	body := []byte("hello")
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	req.Header.Set("X-Amz-Content-Sha256", "")

	v := vectorVerifier(t, vc)
	v.MaxBodySize = int64(len(body))
	_, err := v.Verify(req)
	if errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("err = %v, did not expect ErrBodyTooLarge for in-limit body", err)
	}
}

// zeroReader yields an infinite stream of zero bytes.
type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

// A zero MaxBodySize must resolve to DefaultMaxBodySize inside Verify, not
// unlimited. This drives the real default-substitution path through Verify
// and ResolvePayloadHash rather than mirroring it in the test: a body one
// byte larger than the default must be rejected with ErrBodyTooLarge even
// when MaxBodySize is left zero. The Authorization header only needs to
// parse; the body-read step runs before scope or signature checks, so a
// bogus signature never gets in the way of hitting the size cap.
func TestMaxBodySizeDefaultApplied(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/",
		io.NopCloser(io.LimitReader(zeroReader{}, DefaultMaxBodySize+1)))
	req.Header.Set("Authorization", "AWS4-ECDSA-P256-SHA256 "+
		"Credential=AKIDEXAMPLE/20150830/service/aws4_request, "+
		"SignedHeaders=host, Signature=00")

	v := &Verifier{
		Lookup: LookuperFunc(func(string) (string, error) {
			t.Fatal("Lookup called before body size check")
			return "", ErrUnknownKey
		}),
		// MaxBodySize intentionally left zero; Verify must substitute
		// DefaultMaxBodySize before reading the body.
	}

	_, err := v.Verify(req)
	if !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("err = %v, want ErrBodyTooLarge", err)
	}
}
