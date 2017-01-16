package gluaexpect

import (
	"github.com/ThomasRooney/gexpect"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

func Preload(L *lua.LState) {
	L.PreloadModule("expect", Loader)
}

// Loader is the module loader function.
func Loader(L *lua.LState) int {
	mod := L.SetFuncs(L.NewTable(), api)
	L.Push(mod)
	return 1
}

var api = map[string]lua.LGFunction{
	"spawn": spawn,
}

func spawn(L *lua.LState) int {
	cmd := L.CheckString(1)
	child, err := gexpect.Spawn(cmd)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(luar.New(L, child))
	return 1
}
