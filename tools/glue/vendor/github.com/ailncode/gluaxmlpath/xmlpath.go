package gluaxmlpath

import (
	"github.com/yuin/gopher-lua"
)

// Preload adds xmlpath to the given Lua state's package.preload table. After it
// has been preloaded, it can be loaded using require:
//
//  local xmlpath = require("xmlpath")
func Preload(L *lua.LState) {
	L.PreloadModule("xmlpath", Loader)
}

// Loader is the module loader function.
func Loader(L *lua.LState) int {
	mod := L.SetFuncs(L.NewTable(), api)
	registerType(L, mod)
	L.Push(mod)
	return 1
}
