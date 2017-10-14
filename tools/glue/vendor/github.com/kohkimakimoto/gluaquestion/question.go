package gluaquestion

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/howeyc/gopass"
	"github.com/yuin/gopher-lua"
)

// Loader is glua module loader.
func Loader(L *lua.LState) int {
	tb := L.NewTable()
	L.SetFuncs(tb, map[string]lua.LGFunction{
		"ask":    ask,
		"secret": secret,
	})
	L.Push(tb)

	return 1
}

func ask(L *lua.LState) int {
	msg := L.CheckString(1)

	fmt.Printf(msg)
	reader := bufio.NewReader(os.Stdin)
	str, _ := reader.ReadString('\n')
	str = strings.TrimRight(str, "\r\n")
	L.Push(lua.LString(str))

	return 1
}

func secret(L *lua.LState) int {
	msg := L.CheckString(1)

	fmt.Printf(msg)
	pass, _ := gopass.GetPasswd()
	L.Push(lua.LString(pass))

	return 1
}
