package awssig

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
)

// ResolvePayloadHash returns the hash string to place in the canonical
// request, buffering and resetting r.Body (capped at maxBodySize; zero means
// unlimited) so downstream handlers see a re-readable stream. When the client
// sent a concrete hash in x-amz-content-sha256 the buffered body is re-hashed
// and confirmed to match, otherwise the signed hash proves nothing about the
// bytes actually received. Any declared hash with a "STREAMING-" prefix is
// rejected — chunked-signing integrity lives in per-chunk signatures in the
// body framing, which neither middleware verifies, and rejecting the whole
// reserved sentinel family is the safe direction for both algorithms.
func ResolvePayloadHash(r *http.Request, maxBodySize int64) (string, error) {
	declared := r.Header.Get("X-Amz-Content-Sha256")

	if strings.HasPrefix(strings.ToUpper(declared), "STREAMING-") {
		return "", ErrStreamingUnsupported
	}

	body, err := readAndLimitBody(r, maxBodySize)
	if err != nil {
		return "", err
	}

	if declared == UnsignedPayload {
		return UnsignedPayload, nil
	}

	sum := sha256.Sum256(body)
	computed := hex.EncodeToString(sum[:])

	if declared == "" {
		// No content hash was signed; fall back to the body hash.
		return computed, nil
	}
	if subtle.ConstantTimeCompare([]byte(strings.ToLower(declared)), []byte(computed)) != 1 {
		return "", ErrBodyHash
	}
	// Use the verified lowercase hash in the canonical request; AWS requires
	// lowercase hex and some clients emit uppercase.
	return computed, nil
}

// readAndLimitBody reads the request body, enforcing maxBodySize when set,
// and resets r.Body to a reader over the bytes that were read on every return
// path so the body is always re-readable afterwards.
func readAndLimitBody(r *http.Request, maxBodySize int64) ([]byte, error) {
	var reader io.Reader = r.Body
	if maxBodySize > 0 {
		reader = io.LimitReader(r.Body, maxBodySize+1)
	}
	body, err := io.ReadAll(reader)
	// The original stream is consumed regardless; surface whatever was read.
	r.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if maxBodySize > 0 && int64(len(body)) > maxBodySize {
		return nil, ErrBodyTooLarge
	}
	return body, nil
}
