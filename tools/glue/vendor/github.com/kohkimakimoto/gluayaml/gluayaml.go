package gluayaml

import (
	"github.com/yuin/gopher-lua"
	"gopkg.in/yaml.v2"
)

func Loader(L *lua.LState) int {
	tb := L.NewTable()
	L.SetFuncs(tb, map[string]lua.LGFunction{
		"parse": apiParse,
		"dump":  apiDump,
	})
	L.Push(tb)

	return 1
}

// apiParse parses yaml formatted text to the table.
func apiParse(L *lua.LState) int {
	str := L.CheckString(1)

	var value interface{}
	err := yaml.Unmarshal([]byte(str), &value)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(fromYAML(L, value))
	return 1
}

// apiDump dumps the yaml formatted text.
func apiDump(L *lua.LState) int {
	L.RaiseError("unimplemented function")
	return 0
}

func fromYAML(L *lua.LState, value interface{}) lua.LValue {
	switch converted := value.(type) {
	case bool:
		return lua.LBool(converted)
	case float64:
		return lua.LNumber(converted)
	case int:
		return lua.LNumber(converted)
	case string:
		return lua.LString(converted)
	case []interface{}:
		arr := L.CreateTable(len(converted), 0)
		for _, item := range converted {
			arr.Append(fromYAML(L, item))
		}
		return arr
	case map[interface{}]interface{}:
		tbl := L.CreateTable(0, len(converted))
		for key, item := range converted {
			if s, ok := key.(string); ok {
				tbl.RawSetH(lua.LString(s), fromYAML(L, item))
			}
		}
		return tbl
	}

	return lua.LNil
}
