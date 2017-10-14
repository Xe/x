package luar // import "layeh.com/gopher-luar"

import (
	"reflect"

	"github.com/yuin/gopher-lua"
)

func chanIndex(L *lua.LState) int {
	_, mt, isPtr := check(L, 1, reflect.Chan)
	key := L.CheckString(2)

	if !isPtr {
		if fn := mt.method(key); fn != nil {
			L.Push(fn)
			return 1
		}
	}

	if fn := mt.ptrMethod(key); fn != nil {
		L.Push(fn)
		return 1
	}

	return 0
}

func chanLen(L *lua.LState) int {
	ref, _, isPtr := check(L, 1, reflect.Chan)
	if isPtr {
		L.RaiseError("invalid operation on chan pointer")
	}
	L.Push(lua.LNumber(ref.Len()))
	return 1
}

func chanEq(L *lua.LState) int {
	ref1, _, isPtr1 := check(L, 1, reflect.Chan)
	ref2, _, isPtr2 := check(L, 2, reflect.Chan)

	if isPtr1 && isPtr2 {
		L.Push(lua.LBool(ref1.Pointer() == ref2.Pointer()))
		return 1
	}

	if isPtr1 || isPtr2 {
		L.RaiseError("invalid operation == on mixed chan value and pointer")
	}

	L.Push(lua.LBool(ref1.Interface() == ref2.Interface()))
	return 1
}

// chan methods

func chanSend(L *lua.LState) int {
	ref, _, _ := check(L, 1, reflect.Chan)
	value := L.CheckAny(2)
	convertedValue := lValueToReflect(L, value, ref.Type().Elem(), nil)
	if convertedValue.Type() != ref.Type().Elem() {
		L.ArgError(2, "incorrect type")
	}
	ref.Send(convertedValue)
	return 0
}

func chanReceive(L *lua.LState) int {
	ref, _, _ := check(L, 1, reflect.Chan)

	value, ok := ref.Recv()
	if !ok {
		L.Push(lua.LNil)
		L.Push(lua.LBool(false))
		return 2
	}
	L.Push(New(L, value.Interface()))
	L.Push(lua.LBool(true))
	return 2
}

func chanClose(L *lua.LState) int {
	ref, _, _ := check(L, 1, reflect.Chan)
	ref.Close()
	return 0
}
