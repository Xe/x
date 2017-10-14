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

	testReturn(t, L, `return ch()`, "Tim", "true")
	testReturn(t, L, `ch("John")`)
	testReturn(t, L, `return ch()`, "nil", "false")
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

func Test_chan_invaliddirection(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	ch := make(chan string)

	L.SetGlobal("send", New(L, (chan<- string)(ch)))
	testError(t, L, `send()`, "receive from send-only type chan<- string")

	L.SetGlobal("receive", New(L, (<-chan string)(ch)))
	testError(t, L, `receive("hello")`, "send to receive-only type <-chan string")
}
