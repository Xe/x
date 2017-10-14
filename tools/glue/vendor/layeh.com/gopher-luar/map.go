package luar // import "layeh.com/gopher-luar"

import (
	"reflect"

	"github.com/yuin/gopher-lua"
)

func mapIndex(L *lua.LState) int {
	ref, mt, isPtr := check(L, 1, reflect.Map)
	key := L.CheckAny(2)

	if isPtr {
		if lstring, ok := key.(lua.LString); ok {
			if fn := mt.ptrMethod(string(lstring)); fn != nil {
				L.Push(fn)
				return 1
			}
		}
		return 0
	}

	convertedKey := lValueToReflect(L, key, ref.Type().Key(), nil)
	item := ref.MapIndex(convertedKey)
	if !item.IsValid() {

		if !isPtr {
			if lstring, ok := key.(lua.LString); ok {
				if fn := mt.method(string(lstring)); fn != nil {
					L.Push(fn)
					return 1
				}
			}
		}

		if lstring, ok := key.(lua.LString); ok {
			if fn := mt.ptrMethod(string(lstring)); fn != nil {
				L.Push(fn)
				return 1
			}
		}

		return 0
	}
	L.Push(New(L, item.Interface()))
	return 1
}

func mapNewIndex(L *lua.LState) int {
	ref, _, isPtr := check(L, 1, reflect.Map)
	if isPtr {
		L.RaiseError("invalid operation on map pointer")
	}
	key := L.CheckAny(2)
	value := L.CheckAny(3)

	convertedKey := lValueToReflect(L, key, ref.Type().Key(), nil)
	if convertedKey.Type() != ref.Type().Key() {
		L.ArgError(2, "invalid map key type")
	}
	var convertedValue reflect.Value
	if value != lua.LNil {
		convertedValue = lValueToReflect(L, value, ref.Type().Elem(), nil)
		if convertedValue.Type() != ref.Type().Elem() {
			L.ArgError(3, "invalid map value type")
		}
	}
	ref.SetMapIndex(convertedKey, convertedValue)
	return 0
}

func mapLen(L *lua.LState) int {
	ref, _, isPtr := check(L, 1, reflect.Map)
	if isPtr {
		L.RaiseError("invalid operation on map pointer")
	}
	L.Push(lua.LNumber(ref.Len()))
	return 1
}

func mapCall(L *lua.LState) int {
	ref, _, isPtr := check(L, 1, reflect.Map)
	if isPtr {
		L.RaiseError("invalid operation on map pointer")
	}
	keys := ref.MapKeys()
	i := 0
	fn := func(L *lua.LState) int {
		if i >= len(keys) {
			return 0
		}
		L.Push(New(L, keys[i].Interface()))
		L.Push(New(L, ref.MapIndex(keys[i]).Interface()))
		i++
		return 2
	}
	L.Push(L.NewFunction(fn))
	return 1
}

func mapEq(L *lua.LState) int {
	ref1, _, isPtr1 := check(L, 1, reflect.Map)
	ref2, _, isPtr2 := check(L, 2, reflect.Map)

	if isPtr1 && isPtr2 {
		L.Push(lua.LBool(ref1.Pointer() == ref2.Pointer()))
		return 1
	}

	L.RaiseError("invalid operation == on map")
	return 0 // never reaches
}
