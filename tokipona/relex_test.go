package tokipona

import "testing"

func TestRelex(t *testing.T) {
	cases := []struct {
		tokiPona, english string
	}{
		{
			tokiPona: "sina sona",
			english:  "you know",
		},
		{
			tokiPona: "butt",
			english:  "butt",
		},
	}

	for _, cs := range cases {
		t.Run(cs.tokiPona, func(t *testing.T) {
			get := Relex(cs.tokiPona)
			if get != cs.english {
				t.Fatalf("wanted: %q got: %q", cs.english, get)
			}
		})
	}
}
