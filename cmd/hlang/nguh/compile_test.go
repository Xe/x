package nguh

import (
	"testing"

	"github.com/eaburns/peggy/peg"
)

func TestCompile(t *testing.T) {
	inp := &peg.Node{Text: "h"}

	_, err := Compile(inp)
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkCompile(b *testing.B) {
	inp := &peg.Node{Kids: []*peg.Node{
		{Text: "h"},
		{Text: "h"},
		{Text: "h"},
	},
	}

	for i := 0; i < b.N; i++ {
		if _, err := Compile(inp); err != nil {
			b.Fatal(err)
		}
	}
}
