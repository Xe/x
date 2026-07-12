// Package sigv4a signs and verifies HTTP requests with AWS Signature Version
// 4A (the AWS4-ECDSA-P256-SHA256 scheme): the same canonical-request
// construction as classic SigV4 (shared via web/middleware/internal/awssig),
// but signed with an ECDSA P-256 key derived deterministically from the
// credential instead of an HMAC key. Verifiers can therefore hold only the
// public key — material that verifies signatures but can never mint them.
package sigv4a

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"math/big"

	"within.website/x/web/middleware/internal/awssig"
)

// algorithm is the SigV4A signing algorithm name. It is also the label in
// the key-derivation input.
const algorithm = "AWS4-ECDSA-P256-SHA256"

// nMinusTwo is the P-256 curve order minus two, the rejection-sampling bound
// from the SigV4A key-derivation spec (aws-c-auth key_derivation.c).
var nMinusTwo = new(big.Int).Sub(elliptic.P256().Params().N, big.NewInt(2))

// DeriveKeyPair deterministically derives the SigV4A ECDSA P-256 keypair for
// a credential. It implements AWS's counter-mode KDF (NIST SP 800-108 style,
// PRF = HMAC-SHA256, key = "AWS4A"+secret) with rejection sampling: a
// candidate above N-2 retries with the next counter byte (1..254), an
// accepted candidate is incremented by one to land in [1, N-1]. The
// per-attempt rejection probability is ~2^-32, so the loop virtually always
// succeeds on the first pass.
//
// The keypair is a pure function of (accessKeyID, secretAccessKey): unlike
// the SigV4 HMAC ladder there is no date/region/service scoping, so the same
// key signs and verifies for the credential's whole lifetime.
//
// The big.Int comparison is not constant-time; that is acceptable because
// derivation only ever runs over our own stored secret, never comparing
// against attacker-controlled input.
func DeriveKeyPair(accessKeyID, secretAccessKey string) (*ecdsa.PrivateKey, error) {
	inputKey := []byte("AWS4A" + secretAccessKey)
	for counter := 1; counter <= 254; counter++ {
		// fixedInput layout (aws-c-auth key_derivation.c):
		//   BE32(1) || label || 0x00 || accessKeyID || counterByte || BE32(256)
		fixedInput := make([]byte, 0, 32+len(accessKeyID))
		fixedInput = append(fixedInput, 0x00, 0x00, 0x00, 0x01)
		fixedInput = append(fixedInput, algorithm...)
		fixedInput = append(fixedInput, 0x00)
		fixedInput = append(fixedInput, accessKeyID...)
		fixedInput = append(fixedInput, byte(counter))
		fixedInput = append(fixedInput, 0x00, 0x00, 0x01, 0x00)

		candidate := new(big.Int).SetBytes(awssig.HMACSHA256(inputKey, fixedInput))
		if candidate.Cmp(nMinusTwo) > 0 {
			continue
		}
		candidate.Add(candidate, big.NewInt(1))

		// Round-trip the scalar through crypto/ecdh: NewPrivateKey validates
		// it is in [1, N-1] and computes the public point without deprecated
		// elliptic API calls.
		ek, err := ecdh.P256().NewPrivateKey(candidate.FillBytes(make([]byte, 32)))
		if err != nil {
			return nil, err
		}
		pub := ek.PublicKey().Bytes() // uncompressed SEC1: 0x04 || X || Y
		return &ecdsa.PrivateKey{
			PublicKey: ecdsa.PublicKey{
				Curve: elliptic.P256(),
				X:     new(big.Int).SetBytes(pub[1:33]),
				Y:     new(big.Int).SetBytes(pub[33:65]),
			},
			D: candidate,
		}, nil
	}
	return nil, errors.New("sigv4a: key derivation exhausted its counter space")
}
