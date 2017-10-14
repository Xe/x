package luar

import (
	"testing"

	"github.com/yuin/gopher-lua"
	"strings"
)

func Test_ptr(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	str := "hello"

	L.SetGlobal("ptr", New(L, &str))

	testReturn(t, L, `return -ptr`, "hello")
}

func Test_ptr_comparison(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	var ptr1 *string
	str := "hello"

	L.SetGlobal("ptr1", New(L, ptr1))
	L.SetGlobal("ptr2", New(L, &str))

	testReturn(t, L, `return ptr1 == nil`, "true")
	testReturn(t, L, `return ptr2 == nil`, "false")
	testReturn(t, L, `return ptr1 == ptr2`, "false")
}

func Test_ptr_assignment(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	str := "hello"

	L.SetGlobal("str", New(L, &str))

	testReturn(t, L, `return tostring(-str)`, "hello")
	testReturn(t, L, `return tostring(str ^ "world")`, "world")
	testReturn(t, L, `return tostring(-str)`, "world")
}

type TestPtrNested struct {
	*TestPtrNestedChild
}

type TestPtrNestedChild struct {
	Value *string
	StructTestPerson
}

func Test_ptr_nested(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	a := TestPtrNested{
		TestPtrNestedChild: &TestPtrNestedChild{
			StructTestPerson: StructTestPerson{
				Name: "Tim",
			},
		},
	}

	L.SetGlobal("a", New(L, a))
	L.SetGlobal("str_ptr", NewType(L, ""))

	testReturn(t, L, `return a.Value == nil`, "true")
	testReturn(t, L, `a.Value = str_ptr(); _ = a.Value ^ "hello"`)
	testReturn(t, L, `return a.Value == nil`, "false")
	testReturn(t, L, `return -a.Value`, "hello")
	testReturn(t, L, `return a.Name`, "Tim")
}

func Test_ptr_assignstruct(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	a := &StructTestPerson{
		Name: "tim",
	}
	b := &StructTestPerson{
		Name: "bob",
	}

	L.SetGlobal("a", New(L, a))
	L.SetGlobal("b", New(L, b))

	testReturn(t, L, `return a.Name`, "tim")
	testReturn(t, L, `_ = a ^ -b`)
	testReturn(t, L, `return a.Name`, "bob")
}

type TestStringType string

func (s *TestStringType) ToUpper() {
	*s = TestStringType(strings.ToUpper(string(*s)))
}

func Test_ptr_nested2(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	a := [...]StructTestPerson{
		{Name: "Tim"},
	}
	s := []StructTestPerson{
		{Name: "Tim", Age: 32},
	}

	str := TestStringType("Hello World")

	L.SetGlobal("a", New(L, &a))
	L.SetGlobal("s", New(L, s))
	L.SetGlobal("p", New(L, s[0]))
	L.SetGlobal("str", New(L, &str))

	testReturn(t, L, `return a[1]:AddNumbers(1, 2, 3, 4, 5)`, "Tim counts: 15")
	testReturn(t, L, `return s[1]:AddNumbers(1, 2, 3, 4)`, "Tim counts: 10")
	testReturn(t, L, `return s[1].LastAddSum`, "10")
	testReturn(t, L, `return p:AddNumbers(1, 2, 3, 4, 5)`, "Tim counts: 15")
	testReturn(t, L, `return p.LastAddSum`, "15")

	testReturn(t, L, `return p.Age`, "32")
	testReturn(t, L, `p:IncreaseAge(); return p.Age`, "33")

	testReturn(t, L, `return -str`, "Hello World")
	testReturn(t, L, `str:ToUpper(); return -str`, "HELLO WORLD")
}
