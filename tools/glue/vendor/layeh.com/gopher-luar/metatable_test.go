package luar

import (
	"testing"

	"github.com/yuin/gopher-lua"
)

func Test_metatable(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	tbl := []struct {
		Value    interface{}
		CustomMT bool
	}{
		{"hello", false},
		{123, false},
		{1.23, false},
		{struct{}{}, true},
		{&struct{}{}, true},
		{[]string{}, true},
		{make(chan string), true},
		{(*string)(nil), true},
		{func() {}, false},
		{map[string]int{}, true},
	}

	for _, v := range tbl {
		mt := MT(L, v.Value)
		if v.CustomMT && mt == nil {
			t.Fatalf("expected to have custom MT for %#v\n", v.Value)
		} else if !v.CustomMT && mt != nil {
			t.Fatalf("unexpected custom MT for %#v\n", v.Value)
		}
	}
}
