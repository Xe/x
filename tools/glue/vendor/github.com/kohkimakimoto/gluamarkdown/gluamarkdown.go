package gluamarkdown

import (
	"github.com/russross/blackfriday"
	"github.com/yuin/gopher-lua"
	"io/ioutil"
)

func Loader(L *lua.LState) int {
	tb := L.NewTable()
	L.SetFuncs(tb, map[string]lua.LGFunction{
		"dostring": doString,
		"dofile": doFile,
	})
	L.Push(tb)

	return 1
}

func doString(L *lua.LState) int {
	inputStr := L.CheckString(1)

	output := blackfriday.MarkdownCommon([]byte(inputStr))
	L.Push(lua.LString(string(output)))

	return 1
}

func doFile(L *lua.LState) int {
	inputfile := L.CheckString(1)

	input, err := ioutil.ReadFile(inputfile)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
	}

	output := blackfriday.MarkdownCommon(input)
	L.Push(lua.LString(string(output)))

	return 1

}
