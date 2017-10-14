package gluaxmlpath

import (
	"bytes"
	"github.com/yuin/gopher-lua"
	xmlpath "gopkg.in/xmlpath.v2"
)

var api = map[string]lua.LGFunction{
	"loadxml": loadXml,
	"compile": compile,
}

func loadXml(L *lua.LState) int {
	xmlStr := L.CheckString(1)
	r := bytes.NewReader([]byte(xmlStr))
	node, err := xmlpath.ParseHTML(r)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(newNode(L, node))
	return 1
}

func compile(L *lua.LState) int {
	xpathStr := L.CheckString(1)
	path, err := xmlpath.Compile(xpathStr)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(newPath(L, path))
	return 1
}
