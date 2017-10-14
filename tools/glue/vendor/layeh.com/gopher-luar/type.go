package luar // import "layeh.com/gopher-luar"

import (
	"reflect"

	"github.com/yuin/gopher-lua"
)

func checkType(L *lua.LState, idx int) reflect.Type {
	ud := L.CheckUserData(idx)
	ref, ok := ud.Value.(reflect.Type)
	if !ok {
		L.ArgError(idx, "expecting type")
	}
	return ref
}

func typeCall(L *lua.LState) int {
	ref := checkType(L, 1)

	var value reflect.Value
	switch ref.Kind() {
	case reflect.Chan:
		buffer := L.OptInt(2, 0)
		value = reflect.MakeChan(ref, buffer)
	case reflect.Map:
		value = reflect.MakeMap(ref)
	case reflect.Slice:
		length := L.OptInt(2, 0)
		capacity := L.OptInt(3, length)
		value = reflect.MakeSlice(ref, length, capacity)
	default:
		value = reflect.New(ref)
	}
	L.Push(New(L, value.Interface()))
	return 1
}

func typeEq(L *lua.LState) int {
	type1 := checkType(L, 1)
	type2 := checkType(L, 2)
	L.Push(lua.LBool(type1 == type2))
	return 1
}
