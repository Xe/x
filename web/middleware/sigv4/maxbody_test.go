package sigv4

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
	body := []byte("hello world")
	req := httptest.NewRequest(http.MethodPost, "https://api.example.com/",
		bytes.NewReader(body))
	signWithSDK(t, req, body)
	req.Header.Set("X-Amz-Content-Sha256", UnsignedPayload)

	v := newVerifier()
	v.MaxBodySize = 4
	_, err := v.Verify(req)
	if !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("err = %v, want ErrBodyTooLarge", err)
	}
}

// A body within an explicitly-set MaxBodySize proceeds past the size check
// (and either verifies or fails later for signature reasons, which is enough
// to prove the cap was honored rather than treated as zero/unlimited).
func TestMaxBodySizeExplicitWithin(t *testing.T) {
	body := []byte("hello")
	req := httptest.NewRequest(http.MethodPost, "https://api.example.com/",
		bytes.NewReader(body))
	signWithSDK(t, req, body)

	v := newVerifier()
	v.MaxBodySize = int64(len(body))
	_, err := v.Verify(req)
	if errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("err = %v, did not expect ErrBodyTooLarge for in-limit body", err)
	}
}

// zeroReader is an endless source of zero bytes used so the default-cap test
// can feed DefaultMaxBodySize+1 bytes without pre-allocating 10 MiB.
type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

// A zero MaxBodySize must resolve to DefaultMaxBodySize, not unlimited: a
// body one byte over the default is rejected. Serves as an end-to-end check
// that Verify actually performs the substitution instead of re-opening the
// unbounded-allocation hole.
func TestMaxBodySizeDefaultApplied(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "https://api.example.com/",
		io.LimitReader(zeroReader{}, DefaultMaxBodySize+1))
	signWithSDKPayloadHash(t, req, UnsignedPayload)

	v := newVerifier()
	if v.MaxBodySize != 0 {
		t.Fatalf("test precondition: newVerifier returned MaxBodySize = %d, want 0", v.MaxBodySize)
	}
	_, err := v.Verify(req)
	if !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("err = %v, want ErrBodyTooLarge", err)
	}
}

// The default cap must stay pinned at 10 MiB; if it drifts the test above
// would silently weaken.
func TestDefaultMaxBodySizeValue(t *testing.T) {
	if DefaultMaxBodySize != 10<<20 {
		t.Fatalf("DefaultMaxBodySize = %d, want %d", DefaultMaxBodySize, 10<<20)
	}
}
