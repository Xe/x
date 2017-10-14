package gluaflag

import "github.com/yuin/gopher-lua"

func toStringSlice(t *lua.LTable) []string {
	args := make([]string, 0, t.Len())
	if zv := t.RawGet(lua.LNumber(0)); zv.Type() != lua.LTNil {
		args = append(args, zv.String())
	}

	t.ForEach(func(k, v lua.LValue) {
		if key, ok := k.(lua.LNumber); !ok || int(key) < 1 {
			return
		}
		args = append(args, v.String())
	})
	return args
}

func toTable(L *lua.LState, s []string) *lua.LTable {
	table := L.NewTable()
	for _, str := range s {
		table.Append(lua.LString(str))
	}
	return table
}

func forEachStrings(L *lua.LState, fn *lua.LFunction) []string {
	p := lua.P{
		Fn:      fn,
		NRet:    -1,
		Protect: true,
	}
	res := []string{}

	for L.CallByParam(p) != nil {
		switch L.GetTop() {
		case 0:
			break
		case 1:
			res = append(res, L.Get(-1).String())
			L.Pop(1)
		case 2:
			res = append(res, L.Get(-2).String())
			L.Pop(2)
		default:
			L.RaiseError("Iterator should return 1 or 2 arguments: got %v", L.GetTop())
		}
	}

	return res
}

func checkFlagSet(L *lua.LState, i int) *FlagSet {
	ud := L.CheckUserData(i)
	if gf, ok := ud.Value.(*FlagSet); ok {
		return gf
	}

	L.RaiseError("expected flagset userdata, got: `%T`", ud.Value)
	return nil
}
