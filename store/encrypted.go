package store

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"

	"golang.org/x/crypto/nacl/secretbox"
)

// nonceSize is the length of the random nonce prefixed to every stored value.
const nonceSize = 24

// NewEncrypted returns a store middleware that transparently encrypts values
// with nacl/secretbox before they reach underlying, and decrypts them on
// retrieval. Store keys are passed through unmodified; only values are
// encrypted.
//
// Stored values have the format: 24-byte random nonce || secretbox ciphertext.
func NewEncrypted(underlying Interface, key *[32]byte) (Interface, error) {
	if underlying == nil {
		return nil, fmt.Errorf("%w: underlying store cannot be nil", ErrBadConfig)
	}
	if key == nil {
		return nil, fmt.Errorf("%w: key cannot be nil", ErrBadConfig)
	}

	return &Encrypted{
		underlying: underlying,
		keyFn:      func(string) *[32]byte { return key },
	}, nil
}

// NewEncryptedFunc returns a store middleware like NewEncrypted, but the
// secretbox key for each value is computed by keyFn from the store key. This
// enables per-key keys and caller-managed key rotation. If keyFn returns nil
// for a key, operations on that key fail with ErrBadConfig.
func NewEncryptedFunc(underlying Interface, keyFn func(storeKey string) *[32]byte) (Interface, error) {
	if underlying == nil {
		return nil, fmt.Errorf("%w: underlying store cannot be nil", ErrBadConfig)
	}
	if keyFn == nil {
		return nil, fmt.Errorf("%w: key function cannot be nil", ErrBadConfig)
	}

	return &Encrypted{
		underlying: underlying,
		keyFn:      keyFn,
	}, nil
}

// Encrypted is a store middleware that encrypts values with nacl/secretbox.
// Create it with NewEncrypted or NewEncryptedFunc.
//
// Encrypted is safe for concurrent use if the underlying store and key
// function are.
type Encrypted struct {
	underlying Interface
	keyFn      func(key string) *[32]byte
}

// Close closes the underlying store if it implements Closer.
func (e *Encrypted) Close() error {
	return Close(e.underlying)
}

// Delete removes a value from the underlying store by key.
func (e *Encrypted) Delete(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("%w: key cannot be empty", ErrBadConfig)
	}

	if err := e.underlying.Delete(ctx, key); err != nil {
		return err
	}

	iopsMetrics.WithLabelValues("encrypted", "Delete")
	return nil
}

// Exists returns nil if the key exists in the underlying store, ErrNotFound
// if it does not.
func (e *Encrypted) Exists(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("%w: key cannot be empty", ErrBadConfig)
	}

	if err := e.underlying.Exists(ctx, key); err != nil {
		return err
	}

	iopsMetrics.WithLabelValues("encrypted", "Exists")
	return nil
}

// Get returns the decrypted value of a key.
func (e *Encrypted) Get(ctx context.Context, key string) ([]byte, error) {
	if key == "" {
		return nil, fmt.Errorf("%w: key cannot be empty", ErrBadConfig)
	}

	key32 := e.keyFn(key)
	if key32 == nil {
		return nil, fmt.Errorf("%w: no encryption key for key %q", ErrBadConfig, key)
	}

	data, err := e.underlying.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	plaintext, ok := openSecretbox(data, key32)
	if !ok {
		return nil, fmt.Errorf("%w: can't decrypt value for key %q", ErrCantDecode, key)
	}

	iopsMetrics.WithLabelValues("encrypted", "Get")
	return plaintext, nil
}

// Set encrypts value and puts it into the underlying store.
func (e *Encrypted) Set(ctx context.Context, key string, value []byte) error {
	if key == "" {
		return fmt.Errorf("%w: key cannot be empty", ErrBadConfig)
	}

	key32 := e.keyFn(key)
	if key32 == nil {
		return fmt.Errorf("%w: no encryption key for key %q", ErrBadConfig, key)
	}

	sealed, err := sealSecretbox(value, key32)
	if err != nil {
		return fmt.Errorf("can't encrypt value for key %q: %w", key, err)
	}

	if err := e.underlying.Set(ctx, key, sealed); err != nil {
		return err
	}

	iopsMetrics.WithLabelValues("encrypted", "Set")
	return nil
}

// List lists the keys in the underlying store optionally matching by prefix.
// Keys are stored in plaintext, so no decryption is needed.
func (e *Encrypted) List(ctx context.Context, prefix string) ([]string, error) {
	keys, err := e.underlying.List(ctx, prefix)
	if err != nil {
		return nil, err
	}

	iopsMetrics.WithLabelValues("encrypted", "List")
	return keys, nil
}

// sealSecretbox encrypts plaintext with a fresh random nonce, returning
// nonce || ciphertext.
func sealSecretbox(plaintext []byte, key *[32]byte) ([]byte, error) {
	var nonce [nonceSize]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, fmt.Errorf("can't generate nonce: %w", err)
	}

	return secretbox.Seal(nonce[:], plaintext, &nonce, key), nil
}

// openSecretbox decrypts nonce || ciphertext produced by sealSecretbox.
func openSecretbox(data []byte, key *[32]byte) ([]byte, bool) {
	if len(data) < nonceSize+secretbox.Overhead {
		return nil, false
	}

	var nonce [nonceSize]byte
	copy(nonce[:], data[:nonceSize])

	return secretbox.Open(nil, data[nonceSize:], &nonce, key)
}
