package sigv4keygen

import (
	"crypto/rand"
	"encoding/base64"
)

func Next() (string, string) {
	ak := "WTHNXA_" + rand.Text()
	sk := "WTHNXS_" + randomSecretAccessKey()

	return ak, sk
}

func randomSecretAccessKey() string {
	src := make([]byte, 42)
	rand.Read(src)
	return base64.RawURLEncoding.EncodeToString(src)
}
