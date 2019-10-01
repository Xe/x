package elfs

import "testing"

func TestNext(t *testing.T) {
	n := Next()
	t.Log(n)
	if len(n) == 0 {
		t.Fatalf("MakeName had a zero output")
	}
}
