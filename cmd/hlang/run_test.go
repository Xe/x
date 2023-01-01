package main

import (
	_ "embed"
	"testing"
)

//go:embed testdata/h.wasm
var bin []byte

func BenchmarkRun(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := run(bin); err != nil {
			b.Fatal(err)
		}
	}
}
