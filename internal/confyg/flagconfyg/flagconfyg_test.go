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

func TestDump(t *testing.T) {
	fs := flag.NewFlagSet("h", flag.PanicOnError)
	fs.String("test-string", "some value", "fill this in pls")
	fs.Bool("test-bool", false, "also fill this in pls")

	err := fs.Parse([]string{"-test-string=foo", "-test-bool"})
	if err != nil {
		t.Fatal(err)
	}

	data := Dump(fs)

	err = Parse("h.cfg", data, fs)
	if err != nil {
		t.Fatal(err)
	}
}
