package gluafs

import (
	"fmt"
	"github.com/yookoala/realpath"
	"github.com/yuin/gopher-lua"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

func Loader(L *lua.LState) int {
	tb := L.NewTable()
	L.SetFuncs(tb, map[string]lua.LGFunction{
		"exists":   exists,
		"read":     read,
		"write":    write,
		"mkdir":    mkdir,
		"remove":   remove,
		"symlink":  symlink,
		"dirname":  dirname,
		"basename": basename,
		"realpath": fnRealpath,
		"getcwd":   getcwd,
		"chdir":    chdir,
		"file":     file,
		"dir":      dir,
		"glob":     glob,
	})
	L.Push(tb)

	return 1
}

func exists(L *lua.LState) int {
	var ret bool

	path := L.CheckString(1)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		ret = false
	} else {
		ret = true
	}

	L.Push(lua.LBool(ret))

	return 1
}

func read(L *lua.LState) int {
	path := L.CheckString(1)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LString(string(content)))
	return 1
}

func write(L *lua.LState) int {
	p := L.CheckString(1)
	content := []byte(L.CheckString(2))
	var mode os.FileMode = 0666

	top := L.GetTop()
	if top == 3 {
		m, err := oct2decimal(L.CheckInt(3))
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		mode = os.FileMode(m)
	}

	err := ioutil.WriteFile(p, content, mode)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LTrue)
	return 1
}

func mkdir(L *lua.LState) int {
	dir := L.CheckString(1)
	var mode os.FileMode = 0777

	top := L.GetTop()
	if top >= 2 {
		m, err := oct2decimal(L.CheckInt(2))
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		mode = os.FileMode(m)
	}

	recursive := false
	if top >= 3 {
		recursive = L.ToBool(3)
	}

	var err error
	if recursive {
		err = os.MkdirAll(dir, mode)
	} else {
		err = os.Mkdir(dir, mode)
	}

	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LTrue)

	return 1
}

func remove(L *lua.LState) int {
	p := L.CheckString(1)

	recursive := false
	if L.GetTop() >= 2 {
		recursive = L.ToBool(2)
	}

	var err error
	if recursive {
		err = os.Remove(p)
	} else {
		err = os.RemoveAll(p)
	}

	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LTrue)

	return 1
}

func symlink(L *lua.LState) int {
	target := L.CheckString(1)
	link := L.CheckString(2)

	err := os.Symlink(target, link)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LTrue)
	return 1
}

func dirname(L *lua.LState) int {
	filep := L.CheckString(1)
	dirna := filepath.Dir(filep)
	L.Push(lua.LString(dirna))

	return 1
}

func basename(L *lua.LState) int {
	filep := L.CheckString(1)
	dirna := filepath.Base(filep)
	L.Push(lua.LString(dirna))

	return 1
}

func fnRealpath(L *lua.LState) int {
	filep := L.CheckString(1)

	real, err := realpath.Realpath(filep)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LString(real))

	return 1
}

func getcwd(L *lua.LState) int {
	dir, err := os.Getwd()
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LString(dir))

	return 1
}

func chdir(L *lua.LState) int {
	dir := L.CheckString(1)

	err := os.Chdir(dir)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LTrue)
	return 1
}

func file(L *lua.LState) int {
	// same: debug.getinfo(2,'S').source
	var dbg *lua.Debug
	var err error
	var ok bool

	dbg, ok = L.GetStack(1)
	if !ok {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprint(dbg)))
		return 2
	}
	_, err = L.GetInfo("S", dbg, lua.LNil)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LString(dbg.Source))
	return 1
}

func dir(L *lua.LState) int {
	// same: debug.getinfo(2,'S').source
	var dbg *lua.Debug
	var err error
	var ok bool

	dbg, ok = L.GetStack(1)
	if !ok {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprint(dbg)))
		return 2
	}
	_, err = L.GetInfo("S", dbg, lua.LNil)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	dirname := filepath.Dir(dbg.Source)
	L.Push(lua.LString(dirname))

	return 1
}

func glob(L *lua.LState) int {
	ptn := L.CheckString(1)
	fn := L.CheckFunction(2)

	files, err := filepath.Glob(ptn)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	for _, f := range files {
		tb := L.NewTable()
		tb.RawSetString("path", lua.LString(f))
		abspath, err := filepath.Abs(f)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		tb.RawSetString("realpath", lua.LString(abspath))

		err = L.CallByParam(lua.P{
			Fn:      fn,
			NRet:    0,
			Protect: true,
		}, tb)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
	}

	L.Push(lua.LTrue)
	return 1
}

func isDir(path string) (ret bool) {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}

	return fi.IsDir()
}

func oct2decimal(oct int) (uint64, error) {
	return strconv.ParseUint(fmt.Sprintf("%d", oct), 8, 32)
}
