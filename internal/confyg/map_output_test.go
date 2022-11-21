package confyg

import "testing"

func TestMapConfig(t *testing.T) {
	mc := MapConfig{}

	const configFile = `subscribe pewdiepie

unsubscribe (
  t-series
)`

	_, err := Parse("test.cfg", []byte(configFile), mc, mc)
	if err != nil {
		t.Fatal(err)
	}

	if mc["subscribe"][0] != "pewdiepie" {
		t.Errorf("wanted subscribe->pewdiepie, got: %s", mc["subscribe"][0])
	}

	if mc["unsubscribe"][0] != "t-series" {
		t.Errorf("wanted unsubscribe->t-series, got: %s", mc["unsubscribe"][0])
	}
}
