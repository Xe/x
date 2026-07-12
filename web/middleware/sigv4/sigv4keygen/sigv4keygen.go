package sigv4keygen

import (
	"crypto/rand"
	"encoding/base64"
)

func Next() (string, string) {
	ak := "WTHNXA_" + randomKey()
	sk := "WTHNXS_" + randomKey()

	return ak, sk
}

func randomKey() string {
	secret := make([]byte, 32)
	rand.Read(secret)
	return base64.RawURLEncoding.EncodeToString(secret)
}
