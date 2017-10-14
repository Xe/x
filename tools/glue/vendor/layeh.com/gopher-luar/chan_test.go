package luar

import (
	"testing"

	"github.com/yuin/gopher-lua"
)

func Test_chan(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	ch := make(chan string)
	go func() {
		ch <- "Tim"
		name, ok := <-ch
		if name != "John" || !ok {
			t.Fatal("invalid value")
		}

		close(ch)
	}()

	L.SetGlobal("ch", New(L, ch))

	testReturn(t, L, `return ch:receive()`, "Tim", "true")
	testReturn(t, L, `ch:send("John")`)
	testReturn(t, L, `return ch:receive()`, "nil", "false")
}

type TestChanString chan string

func (*TestChanString) Test() string {
	return "TestChanString.Test"
}

func (TestChanString) Test2() string {
	return "TestChanString.Test2"
}

func Test_chan_pointermethod(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	a := make(TestChanString)
	b := &a

	L.SetGlobal("b", New(L, b))

	testReturn(t, L, `return b:Test()`, "TestChanString.Test")
	testReturn(t, L, `return b:Test2()`, "TestChanString.Test2")
}
