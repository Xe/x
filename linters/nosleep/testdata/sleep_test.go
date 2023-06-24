package testdata

import (
	"testing"
	"time"
)

func TestNosleep(t *testing.T) {
	time.Sleep(time.Second) // want "use of time.Sleep in testing code"
}

func TestNosleepIgnore(t *testing.T) {
	time.Sleep(time.Second) //nosleep:bypass This test requires us to use a sleep statement here. I hate it too.
}
