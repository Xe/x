package luar

import (
	"testing"

	"github.com/yuin/gopher-lua"
)

func Test_map(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	thangs := map[string]int{
		"ABC": 123,
		"DEF": 456,
	}

	L.SetGlobal("thangs", New(L, thangs))

	testReturn(t, L, `return thangs.ABC`, "123")
	testReturn(t, L, `return thangs.DEF`, "456")
	testReturn(t, L, `return thangs.GHI`, "nil")

	if err := L.DoString(`thangs.GHI = 789`); err != nil {
		t.Fatal(err)
	}

	testReturn(t, L, `thangs.ABC = nil`)

	if v := thangs["GHI"]; v != 789 {
		t.Fatalf(`expecting thangs["GHI"] = 789, got %d`, v)
	}

	if _, ok := thangs["ABC"]; ok {
		t.Fatal(`expecting thangs["ABC"] to be unset`)
	}
}

func Test_map_iterator(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	countries := map[string]string{
		"JP": "Japan",
		"CA": "Canada",
		"FR": "France",
	}

	L.SetGlobal("countries", New(L, countries))

	const code = `
		sorted = {}
		for k, v in countries() do
			table.insert(sorted, v)
		end
		table.sort(sorted)`

	if err := L.DoString(code); err != nil {
		t.Fatal(err)
	}

	testReturn(t, L, `return #sorted, sorted[1], sorted[2], sorted[3]`, "3", "Canada", "France", "Japan")
}
