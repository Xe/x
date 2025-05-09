package licenses

import "testing"

func TestList(t *testing.T) {
	if _, err := List(); err != nil {
		t.Fatal(err)
	}
}
