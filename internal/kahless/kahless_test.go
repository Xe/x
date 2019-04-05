package kahless

import "testing"

func TestDial(t *testing.T) {
	cli, err := Dial()
	if err != nil {
		t.Fatal(err)
	}
	cli.Close()
}
