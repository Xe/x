package luar // import "layeh.com/gopher-luar"

import (
	"reflect"

	"github.com/yuin/gopher-lua"
)

// LState is an wrapper for gopher-lua's LState. It should be used when you
// wish to have a function/method with the standard "func(*lua.LState) int"
// signature.
type LState struct {
	*lua.LState
}

var (
	refTypeLStatePtr      reflect.Type
	refTypeLuaLValueSlice reflect.Type
	refTypeLuaLValue      reflect.Type
	refTypeInt            reflect.Type
)

func init() {
	refTypeLStatePtr = reflect.TypeOf(&LState{})
	refTypeLuaLValueSlice = reflect.TypeOf([]lua.LValue{})
	refTypeLuaLValue = reflect.TypeOf((*lua.LValue)(nil)).Elem()
	refTypeInt = reflect.TypeOf(int(0))
}

func getFunc(L *lua.LState) (ref reflect.Value, refType reflect.Type) {
	ref = L.Get(lua.UpvalueIndex(1)).(*lua.LUserData).Value.(reflect.Value)
	refType = ref.Type()
	return
}

func isPtrReceiverMethod(L *lua.LState) bool {
	return bool(L.Get(lua.UpvalueIndex(2)).(lua.LBool))
}

func funcIsBypass(t reflect.Type) bool {
	if t.NumIn() == 1 && t.NumOut() == 1 && t.In(0) == refTypeLStatePtr && t.Out(0) == refTypeInt {
		return true
	}
	if t.NumIn() == 2 && t.NumOut() == 1 && t.In(1) == refTypeLStatePtr && t.Out(0) == refTypeInt {
		return true
	}
	return false
}

func funcBypass(L *lua.LState) int {
	ref, refType := getFunc(L)

	convertedPtr := false
	var receiver reflect.Value
	var ud lua.LValue

	luarState := LState{L}
	args := make([]reflect.Value, 0, 2)
	if refType.NumIn() == 2 {
		receiverHint := refType.In(0)
		ud = L.Get(1)
		if isPtrReceiverMethod(L) {
			receiver = lValueToReflect(L, ud, receiverHint, &convertedPtr)
		} else {
			receiver = lValueToReflect(L, ud, receiverHint, nil)
		}
		if receiver.Type() != receiverHint {
			L.RaiseError("incorrect receiver type")
		}
		args = append(args, receiver)
		L.Remove(1)
	}
	args = append(args, reflect.ValueOf(&luarState))
	ret := ref.Call(args)[0].Interface().(int)
	if convertedPtr {
		ud.(*lua.LUserData).Value = receiver.Elem().Interface()
	}
	return ret
}

func funcRegular(L *lua.LState) int {
	ref, refType := getFunc(L)

	top := L.GetTop()
	expected := refType.NumIn()
	variadic := refType.IsVariadic()
	if !variadic && top != expected {
		L.RaiseError("invalid number of function arguments (%d expected, got %d)", expected, top)
	}
	if variadic && top < expected-1 {
		L.RaiseError("invalid number of function arguments (%d or more expected, got %d)", expected-1, top)
	}

	convertedPtr := false
	var receiver reflect.Value
	var ud lua.LValue

	args := make([]reflect.Value, top)
	for i := 0; i < L.GetTop(); i++ {
		var hint reflect.Type
		if variadic && i >= expected-1 {
			hint = refType.In(expected - 1).Elem()
		} else {
			hint = refType.In(i)
		}
		var arg reflect.Value
		if i == 0 && isPtrReceiverMethod(L) {
			ud = L.Get(1)
			arg = lValueToReflect(L, ud, hint, &convertedPtr)
			receiver = arg
		} else {
			arg = lValueToReflect(L, L.Get(i+1), hint, nil)
		}
		args[i] = arg
	}
	ret := ref.Call(args)

	if convertedPtr {
		ud.(*lua.LUserData).Value = receiver.Elem().Interface()
	}

	if len(ret) == 1 && ret[0].Type() == refTypeLuaLValueSlice {
		values := ret[0].Interface().([]lua.LValue)
		for _, value := range values {
			L.Push(value)
		}
		return len(values)
	}
	for _, val := range ret {
		L.Push(New(L, val.Interface()))
	}
	return len(ret)
}

func funcWrapper(L *lua.LState, fn reflect.Value, isPtrReceiverMethod bool) *lua.LFunction {
	up := L.NewUserData()
	up.Value = fn

	if funcIsBypass(fn.Type()) {
		return L.NewClosure(funcBypass, up, lua.LBool(isPtrReceiverMethod))
	}
	return L.NewClosure(funcRegular, up, lua.LBool(isPtrReceiverMethod))
}
