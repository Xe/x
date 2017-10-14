// +build race

package luar

import (
	"runtime"
	"testing"

	"github.com/yuin/gopher-lua"
)

func Test_functhread(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	var fn func(x int) string
	L.SetGlobal("fn", New(L, &fn))

	if err := L.DoString(`_ = fn ^ function(x) return tostring(x) .. "!"  end`); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	defer close(done)
	go func() {
		for {
			select {
			case <-done:
				return
			default:
			}

			L.Push(lua.LNumber(1000))

			runtime.Gosched()
		}
	}()

	if ret := fn(123); ret != "123!" {
		t.Fatalf("expected %#v, got %#v", "123!", ret)
	}
}
