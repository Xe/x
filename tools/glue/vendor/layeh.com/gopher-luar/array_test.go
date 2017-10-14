package luar

import (
	"testing"

	"github.com/yuin/gopher-lua"
)

type TestArrayOneString [1]string

func (o TestArrayOneString) Get() string {
	return o[0]
}

func Test_array(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	type Elem struct {
		V [2]string
	}

	var elem Elem
	elem.V[0] = "Hello"
	elem.V[1] = "World"

	var arr TestArrayOneString
	arr[0] = "Test"

	L.SetGlobal("e", New(L, &elem))
	L.SetGlobal("arr", New(L, arr))

	testReturn(t, L, `return #e.V, e.V[1], e.V[2]`, "2", "Hello", "World")
	testReturn(t, L, `e.V[1] = "World"; e.V[2] = "Hello"`)
	testReturn(t, L, `return #e.V, e.V[1], e.V[2]`, "2", "World", "Hello")

	testReturn(t, L, `return #arr, arr[1]`, "1", "Test")
	testReturn(t, L, `return arr:Get()`, "Test")

	testError(t, L, `e.V[1] = nil`, "cannot use nil as type string")
}

func Test_array_iterator(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	a := [...]string{"x", "y"}

	L.SetGlobal("a", New(L, a))
	L.SetGlobal("ap", New(L, &a))

	testReturn(t, L, `local itr = a(); local a, b = itr(); local c, d = itr(); return a, b, c, d`, "1", "x", "2", "y")
	testReturn(t, L, `local itr = ap(); local a, b = itr(); local c, d = itr(); return a, b, c, d`, "1", "x", "2", "y")
}

func Test_array_eq(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	a := [...]string{"x", "y"}
	b := [...]string{"x", "y"}

	L.SetGlobal("a", New(L, a))
	L.SetGlobal("ap", New(L, &a))
	L.SetGlobal("b", New(L, b))
	L.SetGlobal("bp", New(L, &b))

	testReturn(t, L, `return a == b`, "true")
	testReturn(t, L, `return a ~= b`, "false")
	testReturn(t, L, `return ap == nil`, "false")
	testReturn(t, L, `return ap == bp`, "false")
}
