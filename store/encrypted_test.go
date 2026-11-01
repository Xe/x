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
