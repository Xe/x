package elfs

import "testing"

func TestNext(t *testing.T) {
	n := Next()
	if len(n) == 0 {
		t.Fatalf("MakeName had a zero output")
	}
}
