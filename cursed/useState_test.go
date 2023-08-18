package cursed

import "testing"

func TestUseState(t *testing.T) {
	val, setVal := UseState[string]("hi")

	if val() != "hi" {
		t.Errorf("wanted %q but got %q", "hi", val())
	}

	setVal("goodbye")
	if val() != "goodbye" {
		t.Errorf("wanted %q but got %q", "goodbye", val())
	}
}
