package gluasimplebox

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"

	"github.com/brandur/simplebox"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

func Preload(L *lua.LState) {
	L.PreloadModule("simplebox", Loader)
}

// Loader is the module loader function.
func Loader(L *lua.LState) int {
	mod := L.SetFuncs(L.NewTable(), api)
	L.Push(mod)
	return 1
}

var api = map[string]lua.LGFunction{
	"new":    newSecretBox,
	"genkey": genKey,
}

func newSecretBox(L *lua.LState) int {
	key := L.CheckString(1)

	k, err := parseKey(key)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	sb := simplebox.NewFromSecretKey(k)

	L.Push(luar.New(L, &box{sb: sb}))
	return 1
}

func genKey(L *lua.LState) int {
	key, err := generateKey()
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LString(base64.URLEncoding.EncodeToString(key[:])))
	return 1
}

func generateKey() (*[32]byte, error) {
	var k [32]byte
	_, err := rand.Read(k[:])
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func parseKey(s string) (*[32]byte, error) {
	k := &[32]byte{}
	raw, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if n := copy(k[:], raw); n < len(k) {
		return nil, errors.New("not valid")
	}
	return k, nil
}

type box struct {
	sb *simplebox.SimpleBox
}

func (b *box) Encrypt(data string) string {
	result := b.sb.Encrypt([]byte(data))
	return hex.EncodeToString(result)
}

func (b *box) Decrypt(data string) (string, error) {
	d, err := hex.DecodeString(data)
	if err != nil {
		return "", err
	}

	plain, err := b.sb.Decrypt([]byte(d))
	if err != nil {
		return "", err
	}

	return string(plain), nil
}
