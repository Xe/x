package gluassh

import (
	"github.com/yuin/gluamapper"
	"github.com/yuin/gopher-lua"
	"errors"
)

const LConnectionClass = "SSHConnectionClass*"

type Connection struct {

}

// Loader is glua module loader.
func Loader(L *lua.LState) int {
	// register ssh connection class
	registerConnectionClass(L)

	// load module
	tb := L.NewTable()
	L.SetFuncs(tb, map[string]lua.LGFunction{
		"run":    run,
		"get":    get,
		"put":    put,
	})

	// set up meta table
	mt := L.NewTable()
	L.SetField(mt, "__call",L.NewFunction(call))
	L.SetMetatable(tb, mt)

	L.Push(tb)

	return 1
}

func registerConnectionClass(L *lua.LState) {
	mt := L.NewTypeMetatable(LConnectionClass)
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
//		"run":  connRun,
//		"get":  connGet,
//		"put":  connPut,
	}))
}

func call(L *lua.LState) int {
	// TODO: implements
	if L.GetTop() != 2 {
		L.RaiseError("calling ssh requires 1 argment that is a server config.")
	}

//	tb := L.Get(1)
	return 0
}

func newConnection(L *lua.LState) int {
	connection := &Connection{}
	ud := L.NewUserData()
	ud.Value = connection
	L.SetMetatable(ud, L.GetTypeMetatable(LConnectionClass))
	L.Push(ud)
	return 1
}

// This is a Lua module function.
// run execute command on a remote host via SSH.
func run(L *lua.LState) int {
	// communicator
	comm, err := commFromLTable(L.CheckTable(1))
	if err != nil {
		panic(err)
	}
	defer comm.Close()

	// option and command
	option := NewOption()
	cmd := ""

	if L.GetTop() == 4 {
		if err := gluamapper.Map(L.CheckTable(2), option); err != nil {
			panic(err)
		}
		cmds, err := toStrings(L.Get(3))
		if err != nil {
			panic(err)
		}
		cmd = concatCommandLines(cmds)

		fn := L.CheckFunction(4)
		option.OutputFunc = fn

	} else if L.GetTop() == 3 {
		if err := gluamapper.Map(L.CheckTable(2), option); err != nil {
			panic(err)
		}
		cmds, err := toStrings(L.Get(3))
		if err != nil {
			panic(err)
		}
		cmd = concatCommandLines(cmds)
	} else if L.GetTop() == 2 {
		cmds, err := toStrings(L.Get(2))
		if err != nil {
			panic(err)
		}
		cmd = concatCommandLines(cmds)
	} else {
		L.RaiseError("run method requires 2 arugments at least.")
	}

	// run
	ret, err := comm.Run(cmd, option, L)
	if err != nil {
		L.RaiseError("ssh error: %s", err)
	}

	tb := updateLTableByRunResult(L.NewTable(), ret)
	L.Push(tb)

	return 1
}

// This is a Lua module function.
func get(L *lua.LState) int {
	// communicator
	comm, err := commFromLTable(L.CheckTable(1))
	if err != nil {
		panic(err)
	}
	defer comm.Close()

	remote := L.ToString(2)
	local := L.ToString(3)

	err = comm.Get(remote, local)
	if err != nil {
		panic(err)
	}

	return 0
}

// This is a Lua module function.
func put(L *lua.LState) int {
	// communicator
	comm, err := commFromLTable(L.CheckTable(1))
	if err != nil {
		panic(err)
	}
	defer comm.Close()

	local := L.ToString(2)
	remote := L.ToString(3)

	err = comm.Put(local, remote)
	if err != nil {
		panic(err)
	}

	return 0
}

func commFromLTable(table *lua.LTable) (*Communicator, error) {
	// config
	cfg := NewConfig()
	// update config by passed table.
	if err := gluamapper.Map(table, cfg); err != nil {
		return nil, err
	}

	// communicator
	comm, err := NewComm(cfg)
	if err != nil {
		return nil, err
	}

	return comm, nil
}

func toStrings(v lua.LValue) ([]string, error) {
	var ret []string

	if lv, ok := v.(*lua.LTable); ok {
		lv.ForEach(func(tk lua.LValue, tv lua.LValue) {
			if ls, ok := tv.(lua.LString); ok {
				ret = append(ret, string(ls))
			}
		})
	} else {
		if ls, ok := v.(lua.LString); ok {
			ret = append(ret, string(ls))
		} else {
			return ret, errors.New("Could not transfer lua value to string slice")
		}
	}

	return ret, nil
}

func concatCommandLines(cmdlines []string) string {
	cmdline := ""

	for i, v := range cmdlines {
		if i == 0 {
			cmdline = v
		} else {
			cmdline = cmdline + " && " + v
		}
	}

	return cmdline
}

func updateLTableByRunResult(tb *lua.LTable, ret *Result) *lua.LTable {
	if ret != nil {
		tb.RawSetString("out", lua.LString(ret.Out.String()))
		tb.RawSetString("err", lua.LString(ret.Err.String()))
		tb.RawSetString("status", lua.LNumber(ret.Status))
		tb.RawSetString("successful", lua.LBool(ret.Successful()))
		tb.RawSetString("failed", lua.LBool(ret.Failed()))
	} else {
		tb.RawSetString("out", lua.LNil)
		tb.RawSetString("err", lua.LNil)
		tb.RawSetString("status", lua.LNil)
		tb.RawSetString("successful", lua.LBool(false))
		tb.RawSetString("failed", lua.LBool(true))
	}

	return tb
}
