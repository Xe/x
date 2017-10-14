package luar

import (
	"reflect"
	"strings"
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
	testReturn(t, L, `return sp1 == sp2`, "true")

	testReturn(t, L, `return m1 == m1`, "true")
	testReturn(t, L, `return mp1 == mp2`, "true")

	testReturn(t, L, `return c1 == c1`, "true")
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

	testReturn(t, L, `e.S = {"a", "b", "", "c"}`)

	valid := true
	expecting := []string{"a", "b", "", "c"}
	if len(e.S) != len(expecting) {
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

	testReturn(t, L, `e.S = {b = nil, c = "hello"}`)

	valid := true
	expecting := map[string]string{
		"c": "hello",
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

func Test_udconversion(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	ud := L.NewUserData()
	ud.Value = "hello world"
	L.SetGlobal("ud", ud)

	var out int
	L.SetGlobal("out", New(L, &out))

	testError(t, L, `_ = out ^ ud`, "cannot use hello world (type string) as type int")
}

func Test_arrayconversion(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	var arr [3]int
	L.SetGlobal("arr", New(L, &arr))
	testReturn(t, L, `arr = arr ^ {10, 20, 11}; return arr[1], arr[2], arr[3]`, "10", "20", "11")
}

type Test_interface_struct struct{}

func Test_interface(t *testing.T) {
	tbl := []struct {
		Code         string
		Var          func(L *lua.LState) lua.LValue
		Expected     interface{}
		ExpectedType reflect.Type
	}{
		{
			Code:     `nil`,
			Expected: interface{}(nil),
		},
		{
			Code:     `"Hello"`,
			Expected: string("Hello"),
		},
		{
			Code:     `true`,
			Expected: bool(true),
		},
		{
			Code:     `1`,
			Expected: float64(1),
		},
		{
			Code: `function(a, b) end`,
			ExpectedType: reflect.TypeOf(func(...interface{}) []interface{} {
				return nil
			}),
		},
		{
			Code: `{hello = "world", [123] = 321}`,
			Expected: map[interface{}]interface{}{
				string("hello"): string("world"),
				float64(123):    float64(321),
			},
		},
		{
			Code: `var`,
			Var: func(L *lua.LState) lua.LValue {
				ud := L.NewUserData()
				ud.Value = "Hello World"
				return ud
			},
			Expected: string("Hello World"),
		},
		// TODO: LChannel
		// TODO: *LState
	}

	for _, cur := range tbl {
		func() {
			L := lua.NewState()
			defer L.Close()

			var out interface{} = Test_interface_struct{}
			L.SetGlobal("out", New(L, &out))

			if cur.Var != nil {
				L.SetGlobal("var", cur.Var(L))
			}

			if err := L.DoString(`_ = out ^ ` + cur.Code); err != nil {
				t.Fatal(err)
			}

			if cur.ExpectedType != nil {
				if reflect.TypeOf(out) != cur.ExpectedType {
					t.Fatalf("expected conversion of %#v = type %s, got type %s\n", cur.Code, cur.ExpectedType, reflect.TypeOf(out))
				}
			} else if !reflect.DeepEqual(out, cur.Expected) {
				t.Fatalf("expected conversion of %#v = %#v (%T), got %#v (%T)\n", cur.Code, cur.Expected, cur.Expected, out, out)
			}
		}()

	}
}

func Test_recursivetable(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	var x interface{}
	L.SetGlobal("x", New(L, &x))

	if err := L.DoString(`local tbl = {}; tbl.inner = tbl; _ = x ^ tbl`); err != nil {
		t.Fatal(err)
	}
}

func Test_tostringfallback(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	type Struct struct {
	}
	var out string

	L.SetGlobal("struct", New(L, &Struct{}))
	L.SetGlobal("out", New(L, &out))
	if err := L.DoString(`_ = out ^ tostring(struct)`); err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(out, "userdata: ") {
		t.Fatalf("invalid tostring %#v\n", out)
	}
}
