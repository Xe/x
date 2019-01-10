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
	for _, v := range a {
		var has bool
		for _, vv := range b {
			if v == vv {
				has = true
			}
		}
		if !has {
			return false
		}
	}
	return true
}

func selbrisEqual(a, b []Selbri) bool {
	if len(a) != len(b) {
		return false
	}
	for _, v := range a {
		var has bool
		for _, vv := range b {
			if v.Eq(vv) {
				has = true
			}
		}

		if !has {
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
		{
			name: "multiple verbs",
			json: []byte(`[{"part":"subject","tokens":["ona"]},{"part":"verbMarker","sep":"li","tokens":["sona"]},{"part":"verbMarker","sep":"li","tokens":["pona"]},{"part":"objectMarker","sep":"e","tokens":["mute"]},{"part":"punctuation","tokens":["period"]}]`),
			want: []Selbri{
				{
					Predicate: "sona",
					Arguments: []string{"ona", "mute"},
				},
				{
					Predicate: "pona",
					Arguments: []string{"ona", "mute"},
				},
			},
			wantFacts: []string{"sona(ona, mute).", "pona(ona, mute)."},
		},
		{
			name: "multiple subjects and verbs",
			json: []byte(`[{"part":"subject","tokens":["ona","en","sina","en","mi"]},{"part":"verbMarker","sep":"li","tokens":["sona"]},{"part":"verbMarker","sep":"li","tokens":["pona"]},{"part":"objectMarker","sep":"e","tokens":["mute"]},{"part":"punctuation","tokens":["period"]}]`),
			want: []Selbri{
				{
					Predicate: "sona",
					Arguments: []string{"ona", "mute"},
				},
				{
					Predicate: "pona",
					Arguments: []string{"ona", "mute"},
				},
				{
					Predicate: "sona",
					Arguments: []string{"sina", "mute"},
				},
				{
					Predicate: "pona",
					Arguments: []string{"sina", "mute"},
				},
				{
					Predicate: "sona",
					Arguments: []string{"mi", "mute"},
				},
				{
					Predicate: "pona",
					Arguments: []string{"mi", "mute"},
				},
			},
			wantFacts: []string{
				"sona(ona, mute).",
				"pona(ona, mute).",
				"sona(sina, mute).",
				"pona(sina, mute).",
				"sona(mi, mute).",
				"pona(mi, mute).",
			},
		},
		{
			name: "multiple subjects and verbs and objects",
			json: []byte(`[{"part":"subject","tokens":["ona","en","sina","en","mi"]},{"part":"verbMarker","sep":"li","tokens":["sona"]},{"part":"verbMarker","sep":"li","tokens":["pona"]},{"part":"objectMarker","sep":"e","tokens":["ijo","mute"]},{"part":"objectMarker","sep":"e","tokens":["ilo","mute"]},{"part":"punctuation","tokens":["period"]}]`),
			want: []Selbri{
				{
					Predicate: "sona",
					Arguments: []string{"ona", "ijo_mute"},
				},
				{
					Predicate: "sona",
					Arguments: []string{"ona", "ilo_mute"},
				},
				{
					Predicate: "pona",
					Arguments: []string{"ona", "ijo_mute"},
				},
				{
					Predicate: "pona",
					Arguments: []string{"ona", "ilo_mute"},
				},
				{
					Predicate: "sona",
					Arguments: []string{"sina", "ijo_mute"},
				},
				{
					Predicate: "sona",
					Arguments: []string{"sina", "ilo_mute"},
				},
				{
					Predicate: "pona",
					Arguments: []string{"sina", "ijo_mute"},
				},
				{
					Predicate: "pona",
					Arguments: []string{"sina", "ilo_mute"},
				},
				{
					Predicate: "sona",
					Arguments: []string{"mi", "ijo_mute"},
				},
				{
					Predicate: "sona",
					Arguments: []string{"mi", "ilo_mute"},
				},
				{
					Predicate: "pona",
					Arguments: []string{"mi", "ijo_mute"},
				},
				{
					Predicate: "pona",
					Arguments: []string{"mi", "ilo_mute"},
				},
			},
			wantFacts: []string{
				"sona(ona, ijo_mute).",
				"sona(ona, ilo_mute).",
				"sona(sina, ijo_mute).",
				"sona(sina, ilo_mute).",
				"sona(mi, ijo_mute).",
				"sona(mi, ilo_mute).",
				"pona(ona, ijo_mute).",
				"pona(ona, ilo_mute).",
				"pona(sina, ijo_mute).",
				"pona(sina, ilo_mute).",
				"pona(mi, ijo_mute).",
				"pona(mi, ilo_mute).",
			},
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

				t.Error("see logs")
			}

			var facts []string
			for _, s := range sb {
				facts = append(facts, s.Fact())
			}

			t.Run("facts", func(t *testing.T) {
				if !equal(cs.wantFacts, facts) {
					t.Logf("wanted: %v", cs.wantFacts)
					t.Logf("got:    %v", facts)
					t.Error("see -v")
				}
			})
		})
	}
}
