package luar

import (
	"testing"

	"github.com/yuin/gopher-lua"
)

func Test_func_variableargs(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	fn := func(str string, extra ...int) {
		switch str {
		case "a":
			if len(extra) != 3 || extra[0] != 1 || extra[1] != 2 || extra[2] != 3 {
				t.Fatalf("unexpected variable arguments: %v", extra)
			}
		case "b":
			if len(extra) != 0 {
				t.Fatalf("unexpected variable arguments: %v", extra)
			}
		case "c":
			if len(extra) != 1 || extra[0] != 4 {
				t.Fatalf("unexpected variable arguments: %v", extra)
			}
		}
	}

	L.SetGlobal("fn", New(L, fn))

	testReturn(t, L, `return fn("a", 1, 2, 3)`)
	testReturn(t, L, `return fn("b")`)
	testReturn(t, L, `return fn("c", 4)`)
}

func Test_func_structarg(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	tim := &StructTestPerson{
		Name: "Tim",
	}

	fn := func(p *StructTestPerson) string {
		return "Hello, " + p.Name
	}

	L.SetGlobal("person", New(L, tim))
	L.SetGlobal("getHello", New(L, fn))

	testReturn(t, L, `return getHello(person)`, "Hello, Tim")
}

func Test_func_unpacking(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	fn := func(name string, count int) []lua.LValue {
		s := make([]lua.LValue, count+1)
		s[0] = lua.LNumber(count)
		for i := 1; i < count+1; i++ {
			s[i] = lua.LString(name)
		}
		return s
	}

	L.SetGlobal("fn", New(L, fn))

	testReturn(t, L, `return fn("tim", 5)`, "5", "tim", "tim", "tim", "tim", "tim")
}

func Test_func_nilreference(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	var fn func()

	L.SetGlobal("fn", New(L, fn))

	testReturn(t, L, `return fn`, "nil")
}

func Test_func_arrayarg(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	arr := [3]int{1, 2, 3}
	fn := func(val [3]int) {
		if val != arr {
			t.Fatalf("expecting %v, got %v", arr, val)
		}
	}

	L.SetGlobal("fn", New(L, fn))
	L.SetGlobal("arr", New(L, arr))

	testReturn(t, L, `return fn(arr)`)
}

func Test_func_luareturntype(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	fn := func(x ...float64) *lua.LTable {
		tbl := L.NewTable()
		for i := len(x) - 1; i >= 0; i-- {
			tbl.Insert(len(x)-i, lua.LNumber(x[i]))
		}
		return tbl
	}

	L.SetGlobal("fn", New(L, fn))

	testReturn(
		t,
		L,
		`
		local t = {}
		for _, x in ipairs(fn(1, 2, 3)) do
			table.insert(t, x)
		end
		return t[1], t[2], t[3]`,
		"3", "2", "1",
	)

	testReturn(
		t,
		L,
		`
		local t = {}
		for _, x in ipairs(fn()) do
			table.insert(t, x)
		end
		return t[1]`,
		"nil",
	)
}

type TestLuaFuncRef struct {
	F1 *lua.LFunction
}

func Test_func_luafuncref(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	e := &TestLuaFuncRef{}
	L.SetGlobal("e", New(L, e))

	testReturn(
		t,
		L,
		`
		e.F1 = function(str)
			return "Hello World", 123
		end
		`,
	)

	L.Push(e.F1)
	L.Call(0, 2)

	if L.GetTop() != 2 || L.Get(1).String() != "Hello World" || L.Get(2).String() != "123" {
		t.Fatal("incorrect return values")
	}
}

type TestFuncCall struct {
	Fn  func(a string) (string, int)
	Fn2 func(a string, b ...int) string
}

func Test_func_luafunccall(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	e := &TestFuncCall{}
	L.SetGlobal("x", New(L, e))

	testReturn(
		t,
		L,
		`
		i = 0
		x.Fn = function(str)
			i = i + 1
			return ">" .. str .. "<", i
		end

		x.Fn2 = function(str, a, b, c)
			if type(a) == "number" and type(b) == "number" and type(c) == "number" then
				return str
			end
			return ""
		end
		`,
	)

	if str, i := e.Fn("A"); str != ">A<" || i != 1 {
		t.Fatal("unexpected return values")
	}

	if str, i := e.Fn("B"); str != ">B<" || i != 2 {
		t.Fatal("unexpected return values")
	}

	if val := e.Fn2("hello", 1, 2); val != "" {
		t.Fatal("unexpected return value")
	}

	if val := e.Fn2("hello", 1, 2, 3); val != "hello" {
		t.Fatal("unexpected return value")
	}

	if L.GetTop() != 0 {
		t.Fatalf("expecting GetTop to return 0, got %d", L.GetTop())
	}
}
