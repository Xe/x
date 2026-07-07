package sigv4a

import (
	"context"
	"crypto/ecdsa"
)

// Lookuper encapsulates the secret access key lookup so that the underlying
// logic can have arbitrary implementations.
type Lookuper interface {
	// Lookup resolves an access key id to its secret access key. Return
	// ErrUnknownKey for unknown keys. This is the one piece you must supply.
	Lookup(accessKeyID string) (secretAccessKey string, err error)
}

// LookuperFunc adapts an ordinary function to the Lookuper interface, the
// same way http.HandlerFunc adapts a function to http.Handler.
type LookuperFunc func(accessKeyID string) (secretAccessKey string, err error)

// Lookup calls f(accessKeyID).
func (f LookuperFunc) Lookup(accessKeyID string) (secretAccessKey string, err error) {
	return f(accessKeyID)
}

// PublicKeyLookuper resolves an access key id to the credential's SigV4A
// ECDSA P-256 public key. Return ErrUnknownKey when the key does not exist
// or may not sign (disabled key or user); any other error is treated as a
// server fault. Public keys are verification-only material: an
// implementation can hold and cache them without ever being able to mint a
// signature.
type PublicKeyLookuper interface {
	LookupPublicKey(ctx context.Context, accessKeyID string) (*ecdsa.PublicKey, error)
}

// PublicKeyLookuperFunc adapts an ordinary function to PublicKeyLookuper.
type PublicKeyLookuperFunc func(ctx context.Context, accessKeyID string) (*ecdsa.PublicKey, error)

// LookupPublicKey calls f.
func (f PublicKeyLookuperFunc) LookupPublicKey(ctx context.Context, accessKeyID string) (*ecdsa.PublicKey, error) {
	return f(ctx, accessKeyID)
}
