package sigv4keygen

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
)

func Next() (string, string) {
	ak := "WTHNXA_" + randomKey()
	sk := "WTHNXS_" + randomKey()

	return ak, sk
}

func randomKey() string {
	secret := make([]byte, 32)
	rand.Read(secret)
	return strings.ReplaceAll(base64.URLEncoding.EncodeToString(secret), "=", "")
}
