package run

import (
	_ "embed"
	"testing"
)

//go:embed testdata/h.wasm
var bin []byte

func TestRun(t *testing.T) {
	if _, err := Run(bin); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkRun(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := Run(bin); err != nil {
			b.Fatal(err)
		}
	}
}
