package gluaxmlpath

import (
	"github.com/yuin/gopher-lua"
	xmlpath "gopkg.in/xmlpath.v2"
)

type Node struct {
	base *xmlpath.Node
}
type Path struct {
	base *xmlpath.Path
}

type Iter struct {
	base *xmlpath.Iter
}

const luaNodeTypeName = "xmlpath.node"
const luaPathTypeName = "xmlpath.path"
const luaIterTypeName = "xmlpath.iter"

func registerType(L *lua.LState, module *lua.LTable) {
	//reg node
	nodemt := L.NewTypeMetatable(luaNodeTypeName)
	L.SetField(module, "node", nodemt)
	L.SetField(nodemt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"string": nodeString,
	}))
	//reg path
	pathmt := L.NewTypeMetatable(luaPathTypeName)
	L.SetField(module, "path", pathmt)
	L.SetField(pathmt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"iter": iter,
	}))
	//reg iter
	itermt := L.NewTypeMetatable(luaIterTypeName)
	L.SetField(module, "iter", itermt)
	L.SetField(itermt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		//"next": next,
		"node": node,
	}))
}
func newNode(L *lua.LState, n *xmlpath.Node) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = &Node{
		n,
	}
	L.SetMetatable(ud, L.GetTypeMetatable(luaNodeTypeName))
	return ud
}
func checkNode(L *lua.LState) *Node {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*Node); ok {
		return v
	}
	L.ArgError(1, "node expected")
	return nil
}
func newPath(L *lua.LState, p *xmlpath.Path) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = &Path{
		p,
	}
	L.SetMetatable(ud, L.GetTypeMetatable(luaPathTypeName))
	return ud
}
func checkPath(L *lua.LState) *Path {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*Path); ok {
		return v
	}
	L.ArgError(1, "path expected")
	return nil
}
func newIter(L *lua.LState, i *xmlpath.Iter) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = &Iter{
		i,
	}
	L.SetMetatable(ud, L.GetTypeMetatable(luaIterTypeName))
	return ud
}
func checkIter(L *lua.LState) *Iter {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*Iter); ok {
		return v
	}
	L.ArgError(1, "iter expected")
	return nil
}

//iter := path.iter(node)
func iter(L *lua.LState) int {
	path := checkPath(L)
	if L.GetTop() == 2 {
		ut := L.CheckUserData(2)
		if node, ok := ut.Value.(*Node); ok {
			it := path.base.Iter(node.base)
			ltab := L.NewTable()
			i := 1
			for it.Next() {
				L.RawSetInt(ltab, i, newNode(L, it.Node()))
				i++
			}
			L.Push(ltab)
			//L.Push(newIter(L, it))
			return 1
		}
	}
	L.ArgError(1, "node expected")
	return 0
}

//support lua standard iterator
//hasNext := iter.next()
// func next(L *lua.LState) int {
// 	iter := checkIter(L)
// 	L.Push(lua.LBool(iter.base.Next()))
// 	return 1
// }

//node := iter.node()
func node(L *lua.LState) int {
	iter := checkIter(L)
	L.Push(newNode(L, iter.base.Node()))
	return 1
}

//string := node.string()
func nodeString(L *lua.LState) int {
	node := checkNode(L)
	L.Push(lua.LString(node.base.String()))
	return 1
}
