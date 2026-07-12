package sigv4a

import (
	"encoding/hex"
	"testing"
)

// TestDeriveKeyPair_FixedVector pins the KDF against the known-answer vector
// in aws-c-auth tests/key_derivation_tests.c ("Values derived in
// synchronicity with Golang and IAM implementations").
func TestDeriveKeyPair_FixedVector(t *testing.T) {
	priv, err := DeriveKeyPair("AKISORANDOMAASORANDOM", "q+jcrXGc+0zWN6uzclKVhvMmUsIfRPa4rlRandom")
	if err != nil {
		t.Fatalf("DeriveKeyPair: %v", err)
	}
	if got, want := hex.EncodeToString(priv.D.FillBytes(make([]byte, 32))),
		"7fd3bd010c0d9c292141c2b77bfbde1042c92e6836fff749d1269ec890fca1bd"; got != want {
		t.Errorf("private key = %s, want %s", got, want)
	}
	if got, want := hex.EncodeToString(priv.PublicKey.X.FillBytes(make([]byte, 32))),
		"15d242ceebf8d8169fd6a8b5a746c41140414c3b07579038da06af89190fffcb"; got != want {
		t.Errorf("public X = %s, want %s", got, want)
	}
	if got, want := hex.EncodeToString(priv.PublicKey.Y.FillBytes(make([]byte, 32))),
		"0515242cedd82e94799482e4c0514b505afccf2c0c98d6a553bf539f424c5ec0"; got != want {
		t.Errorf("public Y = %s, want %s", got, want)
	}
}

// TestDeriveKeyPair_Vectors derives a keypair from each test vector's
// credentials and checks it matches the vector's published public key.
func TestDeriveKeyPair_Vectors(t *testing.T) {
	for _, dir := range vectorDirs(t) {
		t.Run(dir, func(t *testing.T) {
			vc := loadVectorContext(t, dir)
			priv, err := DeriveKeyPair(vc.Credentials.AccessKeyID, vc.Credentials.SecretAccessKey)
			if err != nil {
				t.Fatalf("DeriveKeyPair: %v", err)
			}
			want := vectorPublicKey(t, dir)
			if priv.PublicKey.X.Cmp(want.X) != 0 || priv.PublicKey.Y.Cmp(want.Y) != 0 {
				t.Error("derived public key does not match vector public-key.json")
			}
		})
	}
}
