package main

import (
	"encoding/json"
	"log"
	"testing"

	"github.com/kr/pretty"
	"within.website/x/web/tokiponatokens"
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

func bridisEqual(a, b []Bridi) bool {
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

// Eq checks two Bridi instances for equality.
func (lhs Bridi) Eq(rhs Bridi) bool {
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

func TestSentenceToBridis(t *testing.T) {
	cases := []struct {
		name      string
		json      []byte
		want      []Bridi
		wantFacts []string
	}{
		{
			name: "basic",
			json: []byte(`[{"part":"subject","tokens":["ona"]},{"part":"verbMarker","sep":"li","tokens":["sona"]},{"part":"objectMarker","sep":"e","tokens":["mute"]},{"part":"punctuation","tokens":["period"]}]`),
			want: []Bridi{
				{
					Predicate: "sona",
					Arguments: []string{"subject(ona)", "object(mute)"},
				},
			},
			wantFacts: []string{"bridi(verb(sona), subject(ona), object(mute))."},
		},
		{
			name: "zen",
			json: []byte(`[{"part":"subject","tokens":["tenpo","ni"]},{"part":"punctuation","tokens":["la"]},{"part":"subject","tokens":["seme"]},{"part":"verbMarker","sep":"li","tokens":["ala"]}]`),
			want: []Bridi{
				{
					Predicate: "ala",
					Arguments: []string{"context(subject(tenpo_ni))", "subject(seme)"},
				},
			},
			wantFacts: []string{"bridi(verb(ala), context(subject(tenpo_ni)), subject(A))."},
		},
		{
			name: "pi_subject",
			json: []byte(`[{"part":"subject","parts":[{"part":"subject","tokens":["ilo","mi"]},{"part":"subject","sep":"pi","tokens":["kasi","nasa"]}]},{"part":"verbMarker","sep":"li","tokens":["pona","ale"]}]`),
			want: []Bridi{
				{
					Predicate: "pona_ale",
					Arguments: []string{"subject(ilo_mi, pi(kasi, nasa))"},
				},
			},
			wantFacts: []string{"bridi(verb(pona_ale), subject(ilo_mi, pi(kasi, nasa)))."},
		},
		{
			name: "pi_object",
			json: []byte(`[{"part":"subject","tokens":["mi"]},{"part":"verbMarker","sep":"li","tokens":["esun"]},{"part":"objectMarker","sep":"e","parts":[{"part":"objectMarker","tokens":["ilo"]},{"part":"objectMarker","sep":"pi","tokens":["kalama","musi"]}]},{"part":"punctuation","tokens":["period"]}]`),
			want: []Bridi{
				{
					Predicate: "esun",
					Arguments: []string{"subject(mi)", "object(ilo, pi(kalama, musi))"},
				},
			},
			wantFacts: []string{"bridi(verb(esun), subject(mi), object(ilo, pi(kalama, musi)))."},
		},
		{
			name: "multiple verbs",
			json: []byte(`[{"part":"subject","tokens":["ona"]},{"part":"verbMarker","sep":"li","tokens":["sona"]},{"part":"verbMarker","sep":"li","tokens":["pona"]},{"part":"objectMarker","sep":"e","tokens":["mute"]},{"part":"punctuation","tokens":["period"]}]`),
			want: []Bridi{
				{
					Predicate: "sona",
					Arguments: []string{"subject(ona)", "object(mute)"},
				},
				{
					Predicate: "pona",
					Arguments: []string{"subject(ona)", "object(mute)"},
				},
			},
			wantFacts: []string{
				"bridi(verb(sona), subject(ona), object(mute)).",
				"bridi(verb(pona), subject(ona), object(mute)).",
			},
		},
		{
			name: "multiple subjects and verbs",
			json: []byte(`[{"part":"subject","tokens":["ona","en","sina","en","mi"]},{"part":"verbMarker","sep":"li","tokens":["sona"]},{"part":"verbMarker","sep":"li","tokens":["pona"]},{"part":"objectMarker","sep":"e","tokens":["mute"]},{"part":"punctuation","tokens":["period"]}]`),
			want: []Bridi{
				{
					Predicate: "sona",
					Arguments: []string{"subject(ona)", "object(mute)"},
				},
				{
					Predicate: "pona",
					Arguments: []string{"subject(ona)", "object(mute)"},
				},
				{
					Predicate: "sona",
					Arguments: []string{"subject(sina)", "object(mute)"},
				},
				{
					Predicate: "pona",
					Arguments: []string{"subject(sina)", "object(mute)"},
				},
				{
					Predicate: "sona",
					Arguments: []string{"subject(mi)", "object(mute)"},
				},
				{
					Predicate: "pona",
					Arguments: []string{"subject(mi)", "object(mute)"},
				},
			},
			wantFacts: []string{
				"bridi(verb(sona), subject(ona), object(mute)).",
				"bridi(verb(sona), subject(sina), object(mute)).",
				"bridi(verb(sona), subject(mi), object(mute)).",
				"bridi(verb(pona), subject(ona), object(mute)).",
				"bridi(verb(pona), subject(sina), object(mute)).",
				"bridi(verb(pona), subject(mi), object(mute)).",
			},
		},
		{
			name: "multiple subjects and verbs and objects",
			json: []byte(`[{"part":"subject","tokens":["ona","en","sina","en","mi"]},{"part":"verbMarker","sep":"li","tokens":["sona"]},{"part":"verbMarker","sep":"li","tokens":["pona"]},{"part":"objectMarker","sep":"e","tokens":["ijo","mute"]},{"part":"objectMarker","sep":"e","tokens":["ilo","mute"]},{"part":"punctuation","tokens":["period"]}]`),
			want: []Bridi{
				{
					Predicate: "sona",
					Arguments: []string{"subject(ona)", "object(ijo_mute)"},
				},
				{
					Predicate: "sona",
					Arguments: []string{"subject(ona)", "object(ilo_mute)"},
				},
				{
					Predicate: "pona",
					Arguments: []string{"subject(ona)", "object(ijo_mute)"},
				},
				{
					Predicate: "pona",
					Arguments: []string{"subject(ona)", "object(ilo_mute)"},
				},
				{
					Predicate: "sona",
					Arguments: []string{"subject(sina)", "object(ijo_mute)"},
				},
				{
					Predicate: "sona",
					Arguments: []string{"subject(sina)", "object(ilo_mute)"},
				},
				{
					Predicate: "pona",
					Arguments: []string{"subject(sina)", "object(ijo_mute)"},
				},
				{
					Predicate: "pona",
					Arguments: []string{"subject(sina)", "object(ilo_mute)"},
				},
				{
					Predicate: "sona",
					Arguments: []string{"subject(mi)", "object(ijo_mute)"},
				},
				{
					Predicate: "sona",
					Arguments: []string{"subject(mi)", "object(ilo_mute)"},
				},
				{
					Predicate: "pona",
					Arguments: []string{"subject(mi)", "object(ijo_mute)"},
				},
				{
					Predicate: "pona",
					Arguments: []string{"subject(mi)", "object(ilo_mute)"},
				},
			},
			wantFacts: []string{
				"bridi(verb(sona), subject(ona), object(ijo_mute)).",
				"bridi(verb(sona), subject(ona), object(ilo_mute)).",
				"bridi(verb(sona), subject(sina), object(ijo_mute)).",
				"bridi(verb(sona), subject(sina), object(ilo_mute)).",
				"bridi(verb(sona), subject(mi), object(ijo_mute)).",
				"bridi(verb(sona), subject(mi), object(ilo_mute)).",
				"bridi(verb(pona), subject(ona), object(ijo_mute)).",
				"bridi(verb(pona), subject(ona), object(ilo_mute)).",
				"bridi(verb(pona), subject(sina), object(ijo_mute)).",
				"bridi(verb(pona), subject(sina), object(ilo_mute)).",
				"bridi(verb(pona), subject(mi), object(ijo_mute)).",
				"bridi(verb(pona), subject(mi), object(ilo_mute)).",
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

			sb, err := SentenceToBridis(s)
			if err != nil {
				t.Fatal(err)
			}

			if !bridisEqual(cs.want, sb) {
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
