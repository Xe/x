package luar

import (
	"github.com/yuin/gopher-lua"
)

func ExampleLState() {
	const code = `
	print(sum(1, 2, 3, 4, 5))
	`

	L := lua.NewState()
	defer L.Close()

	sum := func(L *LState) int {
		total := 0
		for i := 1; i <= L.GetTop(); i++ {
			total += L.CheckInt(i)
		}
		L.Push(lua.LNumber(total))
		return 1
	}

	L.SetGlobal("sum", New(L, sum))

	if err := L.DoString(code); err != nil {
		panic(err)
	}
	// Output:
	// 15
}

func ExampleNewType() {
	L := lua.NewState()
	defer L.Close()

	type Song struct {
		Title  string
		Artist string
	}

	L.SetGlobal("Song", NewType(L, Song{}))
	L.DoString(`
		s = Song()
		s.Title = "Montana"
		s.Artist = "Tycho"
		print(s.Artist .. " - " .. s.Title)
	`)
	// Output:
	// Tycho - Montana
}
