package gluassh

import (
	"github.com/yuin/gopher-lua"
)

// SSH run option
type Option struct {
	// Use sudo
	Sudo bool
	// sudo user
	User string
	// sudo password
	Password string
	// Use stdout
	UseStdout bool
	// Use stderr
	UseStderr bool
	// function receives output.
	OutputFunc *lua.LFunction
}

func NewOption() *Option {
	opt := &Option{
		Sudo:  false,
		UseStdout: true,
		UseStderr: true,
	}

	return opt
}

type LFuncWriter struct {
	outType int
	fn *lua.LFunction
	L *lua.LState
}

func NewLFuncWriter(outType int, fn *lua.LFunction, L *lua.LState) *LFuncWriter {
	return &LFuncWriter{
		L: L,
		outType: outType,
		fn: fn,
	}
}

func (w *LFuncWriter) Write(data []byte) (int, error) {
	if w.outType == 1 {
		err := w.L.CallByParam(lua.P{
			Fn:      w.fn,
			NRet:    0,
			Protect: true,
		}, lua.LString(string(data)))
		if err != nil {
			return len(data), err
		}

	} else {
		err := w.L.CallByParam(lua.P{
			Fn:      w.fn,
			NRet:    0,
			Protect: true,
		}, lua.LString(string(data)))
		if err != nil {
			return len(data), err
		}
	}

	return len(data), nil
}
