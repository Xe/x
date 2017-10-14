package luar // import "layeh.com/gopher-luar"

import (
	"reflect"

	"github.com/yuin/gopher-lua"
)

func arrayIndex(L *lua.LState) int {
	ref, mt, isPtr := check(L, 1, reflect.Array)
	ref = reflect.Indirect(ref)
	key := L.CheckAny(2)

	switch converted := key.(type) {
	case lua.LNumber:
		index := int(converted)
		if index < 1 || index > ref.Len() {
			L.ArgError(2, "index out of range")
		}
		val := ref.Index(index - 1)
		if (val.Kind() == reflect.Struct || val.Kind() == reflect.Array) && val.CanAddr() {
			val = val.Addr()
		}
		L.Push(New(L, val.Interface()))
	case lua.LString:
		if !isPtr {
			if fn := mt.method(string(converted)); fn != nil {
				L.Push(fn)
				return 1
			}
		}
		if fn := mt.ptrMethod(string(converted)); fn != nil {
			L.Push(fn)
			return 1
		}
		return 0
	default:
		L.ArgError(2, "must be a number or string")
	}
	return 1
}

func arrayNewIndex(L *lua.LState) int {
	ref, _, isPtr := check(L, 1, reflect.Array)

	if !isPtr {
		L.RaiseError("invalid operation on array")
	}

	ref = ref.Elem()

	index := L.CheckInt(2)
	value := L.CheckAny(3)
	if index < 1 || index > ref.Len() {
		L.ArgError(2, "index out of range")
	}
	ref.Index(index - 1).Set(lValueToReflect(L, value, ref.Type().Elem(), nil))
	return 0
}

func arrayLen(L *lua.LState) int {
	ref, _, _ := check(L, 1, reflect.Array)
	ref = reflect.Indirect(ref)
	L.Push(lua.LNumber(ref.Len()))
	return 1
}

func arrayCall(L *lua.LState) int {
	ref, _, _ := check(L, 1, reflect.Array)
	ref = reflect.Indirect(ref)

	i := 0
	fn := func(L *lua.LState) int {
		if i >= ref.Len() {
			return 0
		}
		item := ref.Index(i).Interface()
		L.Push(lua.LNumber(i + 1))
		L.Push(New(L, item))
		i++
		return 2
	}

	L.Push(L.NewFunction(fn))
	return 1
}
