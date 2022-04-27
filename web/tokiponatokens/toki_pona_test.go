package tokiponatokens

import (
	"testing"

	"github.com/kr/pretty"
)

func TestTokenizeTokiPona(t *testing.T) {
	t.Skip("no")
	data, err := Tokenize("https://us-central1-golden-cove-408.cloudfunctions.net/function-1", "mi olin e sina.")
	if err != nil {
		t.Fatal(err)
	}

	pretty.Println(data)
}
