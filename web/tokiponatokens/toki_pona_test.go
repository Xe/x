package tokiponatokens

import "testing"

func TestTokenizeTokiPona(t *testing.T) {
	_, err := Tokenize("https://us-central1-golden-cove-408.cloudfunctions.net/function-1", "mi olin e sina.")
	if err != nil {
		t.Fatal(err)
	}
}
