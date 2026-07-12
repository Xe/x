package sigv4keygen

import "testing"

func TestNext(t *testing.T) {
	ak1, sk1 := Next()
	ak2, sk2 := Next()

	t.Log(ak1)
	t.Log(sk1)

	// Keys are opaque random values; just sanity-check entropy and non-emptiness.
	if ak1 == "" || sk1 == "" {
		t.Fatal("keys should be non-empty")
	}
	if ak1 == ak2 {
		t.Fatal("access keys should differ between calls")
	}
	if sk1 == sk2 {
		t.Fatal("secret keys should differ between calls")
	}
	if ak1 == sk1 {
		t.Fatal("access and secret keys should differ")
	}
}
