# Encrypted store middleware (nacl/secretbox)

## Context

Add a middleware adaptor to `within.website/x/store` that transparently encrypts values with `golang.org/x/crypto/nacl/secretbox` before they reach the wrapped store, and decrypts them on read. The encryption key is either a fixed `*[32]byte` value or a function of the store key that returns `*[32]byte` (enables per-key keys and caller-managed rotation). Store keys stay plaintext; only values are encrypted.

User-confirmed choices: key type is secretbox-native `*[32]byte` (no KDF invented by this package); configuration follows the CAS driver's constructor convention — `NewX(...) (Interface, error)` middleware wrapping an `Interface`.

Verified facts this plan builds on:

- `store.Interface` = `Delete/Exists/Get/Set/List` (`store/store.go:38-53`); sentinels `ErrNotFound`, `ErrCantDecode`, `ErrBadConfig` (`store/store.go:11-25`); `iopsMetrics` GaugeVec (`store/store.go:27-32`); `Close(s Interface)` helper (`store/store.go:63-68`).
- `store/cas.go` is the layout model: `NewCAS` → type → `Close` → methods in Delete/Exists/Get/Set/List order, empty-key guards returning `fmt.Errorf("%w: key cannot be empty", ErrBadConfig)`, metrics incremented after success via `iopsMetrics.WithLabelValues("<driver>", "<op>")`.
- secretbox v0.57.0 API (read from module cache): `Seal(out, message []byte, nonce *[24]byte, key *[32]byte) []byte`, `Open(out, box []byte, nonce *[24]byte, key *[32]byte) ([]byte, bool)`, `const Overhead = 16`.
- `golang.org/x/crypto` v0.57.0 is already a direct dependency — **no go.mod change**.
- No symbol collisions exist for `Encrypted`, `NewEncrypted`, `NewEncryptedFunc`, `nonceSize`, `sealSecretbox`, `openSecretbox`, `testSecretboxKey`, `bigValue` (grepped whole repo + package).
- `DirectFile.List` returns full slash-relative keys filtered by prefix, lexical order (`store/directfile.go:109-145`).

## Approach

### Step 1 — create `store/encrypted.go` (new file, exact contents)

```go
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
```

Design notes (already baked into the code above — do not revisit):

- Wire format: `nonce(24) || secretbox ciphertext`; minimum stored length for any value (incl. empty) is `24 + secretbox.Overhead` = 40 bytes.
- Key resolution happens before touching storage (fail fast); a `nil` result from `keyFn` is `ErrBadConfig`, not passthrough.
- Underlying errors (e.g. `ErrNotFound`) pass through unwrapped; only crypto failures map to `ErrCantDecode`.
- `Close()` forwards via the package-level `Close` helper so wrapped backends (CAS/bbolt) release resources; no recursion is possible.
- Metrics driver label is `"encrypted"` (CAS-driver convention per the user's phrasing); the underlying backend still counts its own ops — intentional per-layer visibility.
- Empty-key guards on Delete/Exists/Get/Set mirror CAS exactly; `List` has no guard (empty prefix = list all), also mirroring CAS.

### Step 2 — create `store/encrypted_test.go` (new file, exact contents)

Internal tests (`package store`), following `cas_test.go` conventions: `t.Parallel()`, table-driven with `name`/`want`/`err` fields, `errors.Is`, `t.TempDir()`. `NewDirectFile` is the test backend (simple, raw values directly inspectable); `NewCAS` only for the Close-forwarding test.

```go
package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"slices"
	"testing"

	"golang.org/x/crypto/nacl/secretbox"
)

// testSecretboxKey derives a deterministic 32-byte secretbox key from seed.
func testSecretboxKey(seed string) *[32]byte {
	k := sha256.Sum256([]byte(seed))
	return &k
}

// bigValue returns a deterministic n-byte value with varied contents.
func bigValue(n int) []byte {
	v := make([]byte, n)
	for i := range v {
		v[i] = byte(i)
	}
	return v
}

func TestEncryptedRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cases := []struct {
		name  string
		value []byte
	}{
		{name: "text", value: []byte("hello world")},
		{name: "empty", value: []byte{}},
		{name: "binary with NULs", value: []byte{0x00, 0x01, 0xff, 0x00, 0xfe}},
		{name: "1MiB", value: bigValue(1 << 20)},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			df, err := NewDirectFile(t.TempDir())
			if err != nil {
				t.Fatalf("NewDirectFile() error = %v", err)
			}

			enc, err := NewEncrypted(df, testSecretboxKey("roundtrip"))
			if err != nil {
				t.Fatalf("NewEncrypted() error = %v", err)
			}

			if err := enc.Set(ctx, "foo", tt.value); err != nil {
				t.Fatalf("Set() error = %v", err)
			}

			got, err := enc.Get(ctx, "foo")
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			if !bytes.Equal(got, tt.value) {
				t.Errorf("Get() = %q, want %q", got, tt.value)
			}
		})
	}
}

func TestEncryptedCiphertextIsOpaque(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	df, err := NewDirectFile(t.TempDir())
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
	}

	enc, err := NewEncrypted(df, testSecretboxKey("opaque"))
	if err != nil {
		t.Fatalf("NewEncrypted() error = %v", err)
	}

	plaintext := []byte("top secret payload")
	if err := enc.Set(ctx, "secret", plaintext); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	raw, err := df.Get(ctx, "secret")
	if err != nil {
		t.Fatalf("underlying Get() error = %v", err)
	}

	if bytes.Equal(raw, plaintext) {
		t.Error("stored value equals plaintext")
	}

	if want := len(plaintext) + nonceSize + secretbox.Overhead; len(raw) != want {
		t.Errorf("len(stored) = %d, want %d", len(raw), want)
	}

	if bytes.Contains(raw, plaintext) {
		t.Error("stored value contains plaintext")
	}
}

func TestEncryptedNonceRandomization(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	df, err := NewDirectFile(t.TempDir())
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
	}

	enc, err := NewEncrypted(df, testSecretboxKey("nonce"))
	if err != nil {
		t.Fatalf("NewEncrypted() error = %v", err)
	}

	plaintext := []byte("same message")
	if err := enc.Set(ctx, "k", plaintext); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	raw1, err := df.Get(ctx, "k")
	if err != nil {
		t.Fatalf("underlying Get() error = %v", err)
	}

	if err := enc.Set(ctx, "k", plaintext); err != nil {
		t.Fatalf("Set() again error = %v", err)
	}

	raw2, err := df.Get(ctx, "k")
	if err != nil {
		t.Fatalf("underlying Get() again error = %v", err)
	}

	if bytes.Equal(raw1, raw2) {
		t.Error("re-setting the same value produced identical ciphertext; nonce is not random")
	}

	got, err := enc.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if !bytes.Equal(got, plaintext) {
		t.Errorf("Get() = %q, want %q", got, plaintext)
	}
}

func TestEncryptedKeyFunc(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	df, err := NewDirectFile(t.TempDir())
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
	}

	enc, err := NewEncryptedFunc(df, func(storeKey string) *[32]byte {
		return testSecretboxKey("key:" + storeKey)
	})
	if err != nil {
		t.Fatalf("NewEncryptedFunc() error = %v", err)
	}

	for _, k := range []string{"alpha", "beta"} {
		if err := enc.Set(ctx, k, []byte("value of "+k)); err != nil {
			t.Fatalf("Set(%q) error = %v", k, err)
		}
	}

	for _, k := range []string{"alpha", "beta"} {
		got, err := enc.Get(ctx, k)
		if err != nil {
			t.Fatalf("Get(%q) error = %v", k, err)
		}

		if want := "value of " + k; string(got) != want {
			t.Errorf("Get(%q) = %q, want %q", k, got, want)
		}
	}

	rawAlpha, err := df.Get(ctx, "alpha")
	if err != nil {
		t.Fatalf("underlying Get(alpha) error = %v", err)
	}

	rawBeta, err := df.Get(ctx, "beta")
	if err != nil {
		t.Fatalf("underlying Get(beta) error = %v", err)
	}

	if bytes.Equal(rawAlpha, rawBeta) {
		t.Error("different keys produced identical ciphertext; per-key keys not in use")
	}
}

func TestEncryptedKeyFuncNilKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	df, err := NewDirectFile(t.TempDir())
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
	}

	enc, err := NewEncryptedFunc(df, func(storeKey string) *[32]byte {
		if storeKey == "locked" {
			return nil
		}
		return testSecretboxKey("nilkey")
	})
	if err != nil {
		t.Fatalf("NewEncryptedFunc() error = %v", err)
	}

	if err := enc.Set(ctx, "locked", []byte("x")); !errors.Is(err, ErrBadConfig) {
		t.Errorf("Set() error = %v, want ErrBadConfig", err)
	}

	if _, err := enc.Get(ctx, "locked"); !errors.Is(err, ErrBadConfig) {
		t.Errorf("Get() error = %v, want ErrBadConfig", err)
	}
}

func TestEncryptedWrongKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	df, err := NewDirectFile(t.TempDir())
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
	}

	encA, err := NewEncrypted(df, testSecretboxKey("key A"))
	if err != nil {
		t.Fatalf("NewEncrypted() error = %v", err)
	}

	if err := encA.Set(ctx, "k", []byte("s3cr3t")); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	encB, err := NewEncrypted(df, testSecretboxKey("key B"))
	if err != nil {
		t.Fatalf("NewEncrypted() error = %v", err)
	}

	if _, err := encB.Get(ctx, "k"); !errors.Is(err, ErrCantDecode) {
		t.Errorf("Get() with wrong key error = %v, want ErrCantDecode", err)
	}

	encF, err := NewEncryptedFunc(df, func(string) *[32]byte {
		return testSecretboxKey("key B")
	})
	if err != nil {
		t.Fatalf("NewEncryptedFunc() error = %v", err)
	}

	if _, err := encF.Get(ctx, "k"); !errors.Is(err, ErrCantDecode) {
		t.Errorf("Get() with wrong key function error = %v, want ErrCantDecode", err)
	}
}

func TestEncryptedTamperedCiphertext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	df, err := NewDirectFile(t.TempDir())
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
	}

	enc, err := NewEncrypted(df, testSecretboxKey("tamper"))
	if err != nil {
		t.Fatalf("NewEncrypted() error = %v", err)
	}

	if err := enc.Set(ctx, "k", []byte("authenticated payload")); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	raw, err := df.Get(ctx, "k")
	if err != nil {
		t.Fatalf("underlying Get() error = %v", err)
	}

	raw[len(raw)-1] ^= 0xff
	if err := df.Set(ctx, "k", raw); err != nil {
		t.Fatalf("underlying Set() error = %v", err)
	}

	if _, err := enc.Get(ctx, "k"); !errors.Is(err, ErrCantDecode) {
		t.Errorf("Get() after tamper error = %v, want ErrCantDecode", err)
	}
}

func TestEncryptedTruncatedValue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	df, err := NewDirectFile(t.TempDir())
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
	}

	enc, err := NewEncrypted(df, testSecretboxKey("truncate"))
	if err != nil {
		t.Fatalf("NewEncrypted() error = %v", err)
	}

	if err := df.Set(ctx, "k", []byte("too short")); err != nil {
		t.Fatalf("underlying Set() error = %v", err)
	}

	if _, err := enc.Get(ctx, "k"); !errors.Is(err, ErrCantDecode) {
		t.Errorf("Get() of truncated value error = %v, want ErrCantDecode", err)
	}
}

func TestEncryptedPassthrough(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	df, err := NewDirectFile(t.TempDir())
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
	}

	enc, err := NewEncrypted(df, testSecretboxKey("passthrough"))
	if err != nil {
		t.Fatalf("NewEncrypted() error = %v", err)
	}

	for _, k := range []string{"prefix/bar", "prefix/foo", "other"} {
		if err := enc.Set(ctx, k, []byte("v:"+k)); err != nil {
			t.Fatalf("Set(%q) error = %v", k, err)
		}
	}

	if err := enc.Exists(ctx, "prefix/foo"); err != nil {
		t.Errorf("Exists(prefix/foo) error = %v, want nil", err)
	}

	if err := enc.Exists(ctx, "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Exists(nope) error = %v, want ErrNotFound", err)
	}

	if _, err := enc.Get(ctx, "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get(nope) error = %v, want ErrNotFound", err)
	}

	keys, err := enc.List(ctx, "prefix/")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	slices.Sort(keys)
	if want := []string{"prefix/bar", "prefix/foo"}; !slices.Equal(want, keys) {
		t.Errorf("List() = %v, want %v", keys, want)
	}

	if err := enc.Delete(ctx, "prefix/foo"); err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	if err := enc.Exists(ctx, "prefix/foo"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Exists() after delete error = %v, want ErrNotFound", err)
	}
}

func TestEncryptedEmptyKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	df, err := NewDirectFile(t.TempDir())
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
	}

	enc, err := NewEncrypted(df, testSecretboxKey("emptykey"))
	if err != nil {
		t.Fatalf("NewEncrypted() error = %v", err)
	}

	cases := []struct {
		name string
		op   func() error
	}{
		{name: "get", op: func() error { _, err := enc.Get(ctx, ""); return err }},
		{name: "set", op: func() error { return enc.Set(ctx, "", []byte("x")) }},
		{name: "delete", op: func() error { return enc.Delete(ctx, "") }},
		{name: "exists", op: func() error { return enc.Exists(ctx, "") }},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.op(); !errors.Is(err, ErrBadConfig) {
				t.Errorf("error = %v, want ErrBadConfig", err)
			}
		})
	}
}

func TestNewEncryptedConfig(t *testing.T) {
	t.Parallel()

	df, err := NewDirectFile(t.TempDir())
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
	}

	cases := []struct {
		name       string
		underlying Interface
		key        *[32]byte
		wantErr    error
	}{
		{name: "nil underlying", underlying: nil, key: testSecretboxKey("a"), wantErr: ErrBadConfig},
		{name: "nil key", underlying: df, key: nil, wantErr: ErrBadConfig},
		{name: "valid", underlying: df, key: testSecretboxKey("a")},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			st, err := NewEncrypted(tt.underlying, tt.key)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewEncrypted() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && st == nil {
				t.Fatal("NewEncrypted() = nil store, want non-nil")
			}
		})
	}
}

func TestNewEncryptedFuncConfig(t *testing.T) {
	t.Parallel()

	df, err := NewDirectFile(t.TempDir())
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
	}

	fn := func(string) *[32]byte { return testSecretboxKey("a") }

	cases := []struct {
		name       string
		underlying Interface
		keyFn      func(string) *[32]byte
		wantErr    error
	}{
		{name: "nil underlying", underlying: nil, keyFn: fn, wantErr: ErrBadConfig},
		{name: "nil key function", underlying: df, keyFn: nil, wantErr: ErrBadConfig},
		{name: "valid", underlying: df, keyFn: fn},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			st, err := NewEncryptedFunc(tt.underlying, tt.keyFn)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewEncryptedFunc() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && st == nil {
				t.Fatal("NewEncryptedFunc() = nil store, want non-nil")
			}
		})
	}
}

func TestEncryptedCloseForwards(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cas, err := NewCAS(t.TempDir())
	if err != nil {
		t.Fatalf("NewCAS() error = %v", err)
	}

	enc, err := NewEncrypted(cas, testSecretboxKey("close"))
	if err != nil {
		t.Fatalf("NewEncrypted() error = %v", err)
	}

	if err := enc.Set(ctx, "k", []byte("v")); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := Close(enc); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// The underlying CAS must now be closed: further reads fail.
	if _, err := enc.Get(ctx, "k"); err == nil {
		t.Error("Get() after Close() = nil error, want failure (underlying store closed)")
	}
}

func TestEncryptedMiddlewareComposition(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	df, err := NewDirectFile(t.TempDir())
	if err != nil {
		t.Fatalf("NewDirectFile() error = %v", err)
	}

	enc, err := NewEncrypted(df, testSecretboxKey("compose"))
	if err != nil {
		t.Fatalf("NewEncrypted() error = %v", err)
	}

	js := &JSON[map[string]string]{
		Underlying: enc,
	}

	if err := js.Set(ctx, "cfg", map[string]string{"a": "b"}); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := js.Get(ctx, "cfg")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(got) != 1 || got["a"] != "b" {
		t.Errorf("Get() = %v, want map[a:b]", got)
	}

	raw, err := df.Get(ctx, "cfg")
	if err != nil {
		t.Fatalf("underlying Get() error = %v", err)
	}

	if bytes.Contains(raw, []byte(`"a":"b"`)) {
		t.Error("stored value contains plaintext JSON; value was not encrypted")
	}
}
```

No other files change. The adaptor is purely additive — the only current `store` consumer (`cmd/sapientwindex`) is untouched.

## Critical files & anchors

- `store/store.go:11-68` — sentinels (`ErrBadConfig`, `ErrCantDecode`, `ErrNotFound`), `iopsMetrics`, `Interface`, `Close` helper the middleware reuses.
- `store/cas.go` — layout/constructor/guard/metrics model being mirrored (read before editing).
- `store/directfile.go:109-159` — `List` semantics and empty-key guard wording; DirectFile is the test backend.
- `store/json.go:12-15` — `JSON[T].Underlying` composition used by `TestEncryptedMiddlewareComposition`.
- New: `store/encrypted.go`, `store/encrypted_test.go`.

## Verification

Working directory: `/Users/cadey/Code/Xe/x`. No env vars or fixtures needed.

```bash
gofmt -l store/                 # must print nothing
go vet ./store/                 # must pass
go test ./store/... -count=1 -v # all package tests pass, including the 14 new TestEncrypted*/TestNewEncrypted* functions
go build ./...                  # whole repo still compiles (no consumer touched)
```

New-behavior proof mapped to the ask (all inside the `go test` run above):

- Transparent encryption at rest: `TestEncryptedCiphertextIsOpaque` — Set `"top secret payload"`, raw underlying read is ≠ plaintext, is 40 bytes longer, and does not contain the plaintext.
- Fixed-key round trip incl. empty/binary/1 MiB values: `TestEncryptedRoundTrip`.
- Key as a function of the store key: `TestEncryptedKeyFunc` (per-key keys, distinct ciphertexts) and `TestEncryptedKeyFuncNilKey` (`ErrBadConfig`).
- Fresh random nonce per write: `TestEncryptedNonceRandomization`.
- Wrong key / tampering / truncation → `ErrCantDecode`: `TestEncryptedWrongKey`, `TestEncryptedTamperedCiphertext`, `TestEncryptedTruncatedValue`.
- Drop-in middleware: `TestEncryptedPassthrough` (Exists/List/Delete/ErrNotFound), `TestEncryptedMiddlewareComposition` (`JSON[T]` over `Encrypted` over DirectFile), `TestEncryptedCloseForwards`.

## Assumptions & contingencies

- "Middleware like the CAS driver" is implemented as: constructor `NewEncrypted(underlying Interface, key *[32]byte) (Interface, error)` (plus `NewEncryptedFunc`) wrapping another `Interface`, following CAS conventions — `ErrBadConfig` validation, empty-key guards, `iopsMetrics` with driver label `"encrypted"`, unexported fields, cas.go method order. If the intent was instead `JSON[T]`-style struct-field config, replace the two constructors with exported `Underlying`/`Key`/`KeyFunc` fields (KeyFunc takes precedence); method bodies, wire format, and tests stay the same.
- Key type is `*[32]byte` (user-confirmed). No KDF or passphrase-derivation helper is provided; callers derive/decode 32-byte keys themselves.
- Only values are encrypted; keys and prefixes remain plaintext (per the ask). `List` needs no decryption.
- No format sniffing or migration: values not written through this middleware (including pre-existing plaintext) read back as `ErrCantDecode`. Use a fresh keyspace or re-`Set` through the middleware.
- A `keyFn` returning `nil` means "no key for this store key" → `ErrBadConfig` at operation time (not passthrough-unencrypted).
- `Close()` forwards to the underlying store so wrapped backends release resources.
- Values are sealed as single secretbox messages (no chunking); secretbox guidance favors small messages, and store values here are whole blobs.
- Concurrency safety is the underlying store's and `keyFn`'s; the middleware itself is stateless.
- No commit is part of this plan. If a commit is requested afterward: `feat(store): add encrypted middleware adaptor using nacl/secretbox` with `--signoff`; after committing, `git restore` the ~16 toolchain-noise files the pre-commit hook regenerates (protoc-gen-go version comments, prettier whitespace in vendored JS/CSS) and remove any stray `x` binary.
