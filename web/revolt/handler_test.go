package revolt

import "testing"

func TestNullHandlerIsHandler(t *testing.T) {
	nh := NullHandler{}
	var i any = nh
	if _, ok := i.(Handler); !ok {
		t.Error("NullHandler does not implement Handler")
	}
}
