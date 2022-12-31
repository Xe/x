package nguh

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/eaburns/peggy/peg"
)

func TestCompile(t *testing.T) {
	inp := &peg.Node{Text: "h"}

	result, err := Compile(inp)
	if err != nil {
		t.Fatal(err)
	}

	json.NewEncoder(os.Stdout).Encode(result)
}
