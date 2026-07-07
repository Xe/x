// Package awssig holds the algorithm-independent internals shared by the
// SigV4 (web/middleware/sigv4) and SigV4A (web/middleware/sigv4a)
// middlewares: canonical-request construction, payload-hash resolution, and
// the constants both schemes define identically. Keeping one copy means the
// two packages' signers and verifiers agree by construction.
package awssig

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
)

const (
	// AmzTimeFormat is the X-Amz-Date timestamp layout.
	AmzTimeFormat = "20060102T150405Z"
	// ShortDateFormat is the credential-scope date layout.
	ShortDateFormat = "20060102"
	// Terminator ends every credential scope.
	Terminator = "aws4_request"
	// UnsignedPayload is the payload-hash sentinel for unsigned bodies. AWS
	// defines it case-sensitively.
	UnsignedPayload = "UNSIGNED-PAYLOAD"
)

// Error sentinels shared by both middlewares (each package re-exports them
// under its own name, so errors.Is works with either export).
var (
	ErrStreamingUnsupported = errors.New("sigv4: streaming payloads are not supported")
	ErrBodyHash             = errors.New("sigv4: body does not match x-amz-content-sha256")
	ErrBodyTooLarge         = errors.New("sigv4: request body exceeds limit")
)

// HMACSHA256 computes HMAC-SHA256(key, data).
func HMACSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
