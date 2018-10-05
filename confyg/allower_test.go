package confyg

import (
	"fmt"
	"testing"
)

func TestAllower(t *testing.T) {
	al := AllowerFunc(func(verb string, block bool) bool {
		switch verb {
		case "project":
			if block {
				return false
			}

			return true
		}

		return false
	})

	cases := []struct {
		verb  string
		block bool
		want  bool
	}{
		{
			verb:  "project",
			block: false,
			want:  true,
		},
		{
			verb:  "nonsense",
			block: true,
			want:  false,
		},
	}

	for _, cs := range cases {
		t.Run(fmt.Sprint(cs), func(t *testing.T) {
			result := al.Allow(cs.verb, cs.block)

			if result != cs.want {
				t.Fatalf("wanted Allow(%q, %v) == %v, got: %v", cs.verb, cs.block, cs.want, result)
			}
		})
	}
}
