package luar

import (
	"testing"

	"github.com/yuin/gopher-lua"
)

type TestMetatableChild struct {
}

func (*TestMetatableChild) Public() string {
	return "You can call me"
}

func (TestMetatableChild) Private() string {
	return "Should not be able to call me"
}

type TestMetatable struct {
	*TestMetatableChild
}

func Test_metatable(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	b := &TestMetatable{
		TestMetatableChild: &TestMetatableChild{},
	}

	mt := MT(L, TestMetatable{})
	mt.Blacklist("private", "Private")

	mt = MT(L, TestMetatableChild{})
	mt.Whitelist("public", "Public")

	L.SetGlobal("b", New(L, b))

	testReturn(t, L, `return b:public()`, "You can call me")
	testReturn(t, L, `return b.TestMetatableChild:public()`, "You can call me")
	testError(t, L, `return b:private()`, "attempt to call")
	testError(t, L, `return b.TestMetatableChild:private()`, "attempt to call")
	testError(t, L, `return b:Private()`, "attempt to call")
	testError(t, L, `return b.TestMetatableChild:Private()`, "attempt to call")
	testError(t, L, `local a = -b.TestMetatableChild; return a:Private()`, "attempt to call")
}
