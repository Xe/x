package luar

import (
	"runtime/debug"
	"strconv"
	"strings"
	"testing"

	"github.com/yuin/gopher-lua"
)

type StructTestPerson struct {
	Name       string
	Age        int
	Friend     *StructTestPerson
	LastAddSum int
}

func (p StructTestPerson) Hello() string {
	return "Hello, " + p.Name
}

func (p StructTestPerson) String() string {
	return p.Name + " (" + strconv.Itoa(p.Age) + ")"
}

func (p *StructTestPerson) AddNumbers(L *LState) int {
	sum := 0
	for i := L.GetTop(); i >= 1; i-- {
		sum += L.CheckInt(i)
	}
	L.Push(lua.LString(p.Name + " counts: " + strconv.Itoa(sum)))
	p.LastAddSum = sum
	return 1
}

func (p *StructTestPerson) IncreaseAge() {
	p.Age++
}

func testReturn(t *testing.T, L *lua.LState, code string, values ...string) {
	top := L.GetTop()
	if err := L.DoString(code); err != nil {
		t.Fatalf("%s\n\n%s", err, debug.Stack())
	}

	valid := true
	newTop := L.GetTop()

	if newTop-top != len(values) {
		valid = false
	} else {
		for i, expect := range values {
			// TODO: strong typing
			val := L.Get(top + i + 1).String()
			if val != expect {
				valid = false
			}
		}
	}

	if !valid {
		got := make([]string, newTop-top)
		for i := 0; i < len(got); i++ {
			got[i] = L.Get(top + i + 1).String()
		}

		t.Fatalf("bad return values: expecting %#v, got %#v\n\n%s", values, got, debug.Stack())
	}

	L.SetTop(top)
}

func testError(t *testing.T, L *lua.LState, code, error string) {
	err := L.DoString(code)
	if err == nil {
		t.Fatalf("expecting error, got nil\n\n%s", debug.Stack())
	}

	if s := err.Error(); strings.Index(s, error) == -1 {
		t.Fatalf("error substring '%s' not found in '%s'\n\n%s", error, s, debug.Stack())
	}
}
