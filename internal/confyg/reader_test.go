package confyg

import (
	"bytes"
	"fmt"
	"testing"
)

func TestReader(t *testing.T) {
	done := false
	acc := 0

	al := AllowerFunc(func(verb string, block bool) bool {
		switch verb {
		case "test":
			return !block

		case "acc":
			return true
		default:
			return false
		}
	})

	r := ReaderFunc(func(errs *bytes.Buffer, fs *FileSyntax, line *Line, verb string, args []string) {
		switch verb {
		case "test":
			done = len(args) == 1
		case "acc":
			acc++
		default:
			fmt.Fprintf(errs, "%s:%d unknown verb %s\n", fs.Name, line.Start.Line, verb)
		}
	})
	const configFile = `test "42"

acc (
  1
  2
  3
)`

	fs, err := Parse("test.cfg", []byte(configFile), r, al)
	if err != nil {
		t.Fatal(err)
	}

	_ = fs

	t.Logf("done: %v", done)
	if !done {
		t.Fatal("done was not flagged")
	}

	t.Logf("acc: %v", acc)
	if acc != 3 {
		t.Fatal("acc was not changed")
	}
}
