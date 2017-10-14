package gluatemplate

import (
	"bytes"
	"fmt"
	"github.com/yuin/gopher-lua"
	"text/template"
)

func Loader(L *lua.LState) int {
	tb := L.NewTable()
	L.SetFuncs(tb, map[string]lua.LGFunction{
		"dostring": doString,
		"dofile":   doFile,
	})
	L.Push(tb)

	return 1
}

// render
func doString(L *lua.LState) int {
	tmplcontent := L.CheckString(1)
	var dict interface{}

	tmpl, err := template.New("T").Parse(tmplcontent)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	if L.GetTop() >= 2 {
		dict = toGoValue(L.CheckTable(2))
	}

	var b bytes.Buffer
	if err := tmpl.Execute(&b, dict); err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	s := b.String()

	L.Push(lua.LString(s))
	return 1
}

func doFile(L *lua.LState) int {
	tmplfile := L.CheckString(1)
	var dict interface{}

	tmpl, err := template.ParseFiles(tmplfile)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	if L.GetTop() >= 2 {
		dict = toGoValue(L.CheckTable(2))
	}

	var b bytes.Buffer
	if err := tmpl.Execute(&b, dict); err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	s := b.String()

	L.Push(lua.LString(s))
	return 1
}

// This code refers to https://github.com/yuin/gluamapper/blob/master/gluamapper.go
func toGoValue(lv lua.LValue) interface{} {
	switch v := lv.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(v)
	case lua.LString:
		return string(v)
	case lua.LNumber:
		return float64(v)
	case *lua.LTable:
		maxn := v.MaxN()
		if maxn == 0 { // table
			ret := make(map[interface{}]interface{})
			v.ForEach(func(key, value lua.LValue) {
				keystr := fmt.Sprint(toGoValue(key))
				ret[keystr] = toGoValue(value)
			})
			return ret
		} else { // array
			ret := make([]interface{}, 0, maxn)
			for i := 1; i <= maxn; i++ {
				ret = append(ret, toGoValue(v.RawGetInt(i)))
			}
			return ret
		}
	default:
		return v
	}
}
