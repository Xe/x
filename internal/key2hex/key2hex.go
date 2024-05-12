package key2hex

import (
	"encoding/base64"
	"encoding/hex"
)

func Convert(data string) (string, error) {
	buf := make([]byte, base64.StdEncoding.DecodedLen(len(data))-1)
	_, err := base64.StdEncoding.Decode(buf, []byte(data))
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}
