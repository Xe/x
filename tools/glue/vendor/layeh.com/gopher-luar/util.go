package luar // import "layeh.com/gopher-luar"

import (
	"fmt"
	"reflect"
	"unicode"
	"unicode/utf8"

	"github.com/yuin/gopher-lua"
)

func check(L *lua.LState, idx int, kind reflect.Kind) (ref reflect.Value, mt *Metatable, isPtr bool) {
	ud := L.CheckUserData(idx)
	ref = reflect.ValueOf(ud.Value)
	if ref.Kind() != kind {
		if ref.Kind() != reflect.Ptr || ref.Elem().Kind() != kind {
			s := kind.String()
			L.ArgError(idx, "expecting "+s+" or "+s+" pointer")
		}
		isPtr = true
	}
	mt = &Metatable{LTable: ud.Metatable.(*lua.LTable)}
	return
}

func tostring(L *lua.LState) int {
	ud := L.CheckUserData(1)
	value := ud.Value
	if stringer, ok := value.(fmt.Stringer); ok {
		L.Push(lua.LString(stringer.String()))
	} else {
		L.Push(lua.LString(fmt.Sprintf("userdata (luar): %p", ud)))
	}
	return 1
}

func eq(L *lua.LState) int {
	ud1 := L.CheckUserData(1).Value
	ud2 := L.CheckUserData(2).Value
	L.Push(lua.LBool(reflect.DeepEqual(ud1, ud2)))
	return 1
}

func getUnexportedName(name string) string {
	first, n := utf8.DecodeRuneInString(name)
	if n == 0 {
		return name
	}
	return string(unicode.ToLower(first)) + name[n:]
}
