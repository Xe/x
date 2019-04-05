package kahless

import "testing"

func TestDial(t *testing.T) {
	t.Skip("ugh testing in docker sucks")
	cli, err := Dial()
	if err != nil {
		t.Fatal(err)
	}
	cli.Close()
}
