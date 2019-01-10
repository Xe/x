package main

import (
	"encoding/json"
	"log"
	"testing"

	"github.com/Xe/x/web/tokiponatokens"
	"github.com/kr/pretty"
)

// equal tells whether a and b contain the same elements.
// A nil argument is equivalent to an empty slice.
func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func selbrisEqual(a, b []Selbri) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Eq(b[i]) {
			return false
		}
	}
	return true
}

// Eq checks two Selbri instances for equality.
func (lhs Selbri) Eq(rhs Selbri) bool {
	switch {
	case lhs.Predicate != rhs.Predicate:
		return false
	case len(lhs.Arguments) != len(rhs.Arguments):
		return false
	case len(lhs.Arguments) == len(rhs.Arguments):
		return equal(lhs.Arguments, rhs.Arguments)
	}

	return true
}

func TestSentenceToSelbris(t *testing.T) {
	cases := []struct {
		name      string
		json      []byte
		want      []Selbri
		wantFacts []string
	}{
		{
			name: "basic",
			json: []byte(`[{"part":"subject","tokens":["ona"]},{"part":"verbMarker","sep":"li","tokens":["sona"]},{"part":"objectMarker","sep":"e","tokens":["mute"]},{"part":"punctuation","tokens":["period"]}]`),
			want: []Selbri{
				{
					Predicate: "sona",
					Arguments: []string{"ona", "mute"},
				},
			},
			wantFacts: []string{"sona(ona, mute)."},
		},
		{
			name: "zen",
			json: []byte(`[{"part":"subject","tokens":["tenpo","ni"]},{"part":"punctuation","tokens":["la"]},{"part":"subject","tokens":["seme"]},{"part":"verbMarker","sep":"li","tokens":["ala"]}]`),
			want: []Selbri{
				{
					Predicate: "ala",
					Arguments: []string{"tenpo_ni", "seme"},
				},
			},
			wantFacts: []string{"ala(tenpo_ni, A)."},
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			var s tokiponatokens.Sentence
			err := json.Unmarshal(cs.json, &s)
			if err != nil {
				t.Fatal(err)
			}

			sb, err := SentenceToSelbris(s)
			if err != nil {
				t.Fatal(err)
			}

			if !selbrisEqual(cs.want, sb) {
				log.Println("want:")
				pretty.Println(cs.want)
				log.Println("got:")
				pretty.Println(sb)

				t.Fatal("see logs")
			}

			var facts []string
			for _, s := range sb {
				facts = append(facts, s.Fact())
			}

			t.Run("facts", func(t *testing.T) {
				if !equal(cs.wantFacts, facts) {
					t.Logf("wanted: %v", cs.wantFacts)
					t.Logf("got:    %v", facts)
					t.Fatal("see -v")
				}
			})
		})
	}
}
