package namcu

import (
	"fmt"
	"testing"
)

func TestLojbanDigit(t *testing.T) {
	cases := []struct {
		input  int
		output string
	}{
		{
			input:  1,
			output: "pa",
		},
		{
			input:  10,
			output: "pano",
		},
		{
			input:  1337,
			output: "pacicize",
		},
	}

	for _, cs := range cases {
		t.Run(fmt.Sprint(cs), func(t *testing.T) {
			result := lojbanDigit(cs.input)

			if result != cs.output {
				t.Errorf("expected %[1]d -> %[2]q, got: %[1]d -> %[3]q", cs.input, cs.output, result)
			}
		})
	}
}
