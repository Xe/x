package namegen

import "testing"

func TestNext(t *testing.T) {
	name := Next()
	t.Log(name)
	if name == "" {
		t.Fatal("expected a name")
	}
}
