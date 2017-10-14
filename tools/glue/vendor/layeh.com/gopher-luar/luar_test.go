package luar

import (
	"testing"

	"github.com/yuin/gopher-lua"
)

func Test_luar_complex128(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	a := complex(float64(1), float64(2))

	L.SetGlobal("a", New(L, a))

	testReturn(t, L, `b = a`)

	b := L.GetGlobal("b").(*lua.LUserData).Value.(complex128)
	if a != b {
		t.Fatalf("expected a = b, got %v", b)
	}
}

type ChanAlias chan string

func (ChanAlias) Test() string {
	return `I'm a "chan string" alias`
}

func (ChanAlias) hidden() {
}

type SliceAlias []string

func (s SliceAlias) Len() int {
	return len(s)
}

func (s *SliceAlias) Append(v string) {
	*s = append(*s, v)
}

type MapAlias map[string]int

func (m MapAlias) Y() int {
	return len(m)
}

func Test_type_methods(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	a := make(ChanAlias)
	var b SliceAlias = []string{"Hello", "world"}
	c := MapAlias{
		"x": 15,
	}

	L.SetGlobal("a", New(L, a))
	L.SetGlobal("b", New(L, &b))
	L.SetGlobal("c", New(L, c))

	testReturn(t, L, `return a:Test()`, `I'm a "chan string" alias`)
	testReturn(t, L, `len1 = b:Len(); b:Append("!")`)
	testReturn(t, L, `return len1, b:len()`, "2", "3")
	testReturn(t, L, `return c.x, c:y()`, "15", "1")
}

func Test_comparisons(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	{
		s := make([]int, 10)
		L.SetGlobal("s1", New(L, s))
		L.SetGlobal("sp1", New(L, &s))
		L.SetGlobal("s2", New(L, s))
		L.SetGlobal("sp2", New(L, &s))
	}
	{
		m := make(map[string]int, 10)
		L.SetGlobal("m1", New(L, m))
		L.SetGlobal("mp1", New(L, &m))
		L.SetGlobal("m2", New(L, m))
		L.SetGlobal("mp2", New(L, &m))
	}
	{
		c := make(chan string)
		L.SetGlobal("c1", New(L, c))
		L.SetGlobal("cp1", New(L, &c))
		L.SetGlobal("c2", New(L, c))

		c3 := make(chan string)
		L.SetGlobal("c3", New(L, c3))
	}
	{
		s := ""
		L.SetGlobal("sp1", New(L, &s))
		L.SetGlobal("sp2", New(L, &s))
	}

	testReturn(t, L, `return s1 == s1`, "true")
	testError(t, L, `return s1 == s2`, "invalid operation == on slice")
	testReturn(t, L, `return sp1 == sp2`, "true")

	testReturn(t, L, `return m1 == m1`, "true")
	testError(t, L, `return m1 == m2`, "invalid operation == on map")
	testReturn(t, L, `return mp1 == mp2`, "true")

	testReturn(t, L, `return c1 == c1`, "true")
	testError(t, L, `return c1 == cp1`, "invalid operation == on mixed chan value and pointer")
	testReturn(t, L, `return c1 == c3`, "false")

	testReturn(t, L, `return sp1 == sp2`, "true")
}

type TestSliceConversion struct {
	S []string
}

func Test_sliceconversion(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	e := &TestSliceConversion{}
	L.SetGlobal("e", New(L, e))

	testReturn(t, L, `e.S = {"a", "b", "", 3, true, "c"}`)

	valid := true
	expecting := []string{"a", "b", "", "3", "true", "c"}
	if len(e.S) != 6 {
		valid = false
	} else {
		for i, item := range e.S {
			if item != expecting[i] {
				valid = false
				break
			}
		}
	}

	if !valid {
		t.Fatalf("expecting %#v, got %#v", expecting, e.S)
	}
}

type TestMapConversion struct {
	S map[string]string
}

func Test_mapconversion(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	e := &TestMapConversion{}
	L.SetGlobal("e", New(L, e))

	testReturn(t, L, `e.S = {33, a = 123, b = nil, c = "hello", d = false}`)

	valid := true
	expecting := map[string]string{
		"a": "123",
		"c": "hello",
		"d": "false",
	}

	if len(e.S) != len(expecting) {
		valid = false
	} else {
		for key, value := range e.S {
			expected, ok := expecting[key]
			if !ok || value != expected {
				valid = false
				break
			}
		}
	}

	if !valid {
		t.Fatalf("expecting %#v, got %#v", expecting, e.S)
	}

	if _, ok := e.S["b"]; ok {
		t.Fatal(`e.S["b"] should not be set`)
	}
}
