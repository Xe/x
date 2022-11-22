package flagconfyg

import (
	"flag"
	"testing"
)

func TestFlagConfyg(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.PanicOnError)
	sc := fs.String("subscribe", "", "to pewdiepie")
	us := fs.String("unsubscribe", "all the time", "from t-series")

	const configFile = `subscribe pewdiepie

unsubscribe (
  t-series
)`

	err := Parse("test.cfg", []byte(configFile), fs)
	if err != nil {
		t.Fatal(err)
	}

	if *sc != "pewdiepie" {
		t.Errorf("wanted subscribe->pewdiepie, got: %s", *sc)
	}

	if *us != "t-series" {
		t.Errorf("wanted unsubscribe->t-series, got: %s", *us)
	}
}
