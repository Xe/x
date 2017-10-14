package gluaflag

import (
	"fmt"

	"github.com/yuin/gopher-lua"
)

const (
	luaFlagSetTypeName = "flagset"
	luaFlagTypeName    = "flag"
)

// UserDataTypeError is returned when it is not a flag userdata received
var ErrUserDataType = fmt.Errorf("Expected gluaflag userdata")

var exports = map[string]lua.LGFunction{
	"new": new,
}

// Loader is used for preloading the module
func Loader(L *lua.LState) int {

	// register functions to the table
	mod := L.SetFuncs(L.NewTable(), exports)

	flagSetMetaTable := L.NewTypeMetatable(luaFlagSetTypeName)
	L.SetField(flagSetMetaTable, "__index", L.SetFuncs(L.NewTable(), flagSetFuncs))

	flagMetaTable := L.NewTypeMetatable(luaFlagTypeName)
	L.SetField(flagMetaTable, "__index", L.SetFuncs(L.NewTable(), flagFuncs))

	// returns the module
	L.Push(mod)
	return 1
}
