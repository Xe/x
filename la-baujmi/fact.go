package main

import (
	"errors"
	"strings"

	"github.com/Xe/x/web/tokiponatokens"
)

// Selbri is a predicate relationship between its arguments. The name comes from
// the lojban term selbri: http://jbovlaste.lojban.org/dict/selbri.
type Selbri struct {
	Predicate string
	Arguments []string
}

// Fact converts a selbri into a prolog fact.
func (s Selbri) Fact() string {
	var sb strings.Builder

	sb.WriteString(s.Predicate + "(")

	var varCount byte

	for i, arg := range s.Arguments {
		if i != 0 {
			sb.WriteString(", ")
		}

		if arg == "seme" {
			sb.WriteByte(byte('A') + varCount)
			varCount++
			continue
		}
		sb.WriteString(arg)
	}

	sb.WriteString(").")
	return sb.String()
}

var (
	// ErrNoPredicate is raised when the given sentence does not have a verb.
	// This is a side effect of the way we are potentially misusing prolog
	// here.
	ErrNoPredicate = errors.New("la-baujmi: sentence must have a verb to function as the logical predicate")
)

// SentenceToSelbris creates logical facts derived from toki pona sentences.
// This is intended to be the first step in loading them into prolog.
func SentenceToSelbris(s tokiponatokens.Sentence) ([]Selbri, error) {
	var (
		subjects []string
		verbs    []string
		objects  []string
		context  string
	)

	for _, pt := range s {
		switch pt.Type {
		case tokiponatokens.PartSubject:
			if strings.Contains(pt.Braces(), " en ") {
				subjects = append(subjects, strings.Split(strings.Join(pt.Tokens, " "), " en ")...)
				continue
			}

			subjects = append(subjects, strings.Join(pt.Tokens, "_"))

		case tokiponatokens.PartVerbMarker:
			verbs = append(verbs, strings.Join(pt.Tokens, "_"))

		case tokiponatokens.PartObjectMarker:
			objects = append(objects, strings.Join(pt.Tokens, "_"))

		case tokiponatokens.PartPunctuation:
			if len(pt.Tokens) == 1 {
				switch pt.Tokens[0] {
				case "la":
					context = subjects[len(subjects)-1]
					subjects = subjects[:len(subjects)-1]

				case tokiponatokens.PunctComma:
					return nil, errors.New("please avoid commas in this function")
				}
			}
		}
	}

	if len(verbs) == 0 {
		return nil, ErrNoPredicate
	}

	var result []Selbri

	for _, v := range verbs {
		for _, s := range subjects {
			if len(objects) == 0 {
				// sumti: x1 is a/the argument of predicate function x2 filling place x3 (kind/number)
				var sumti []string
				if context != "" {
					sumti = append([]string{}, context)
				}
				sumti = append(sumti, s)

				r := Selbri{
					Predicate: v,
					Arguments: sumti,
				}

				result = append(result, r)
			}

			for _, o := range objects {
				// sumti: x1 is a/the argument of predicate function x2 filling place x3 (kind/number)
				var sumti []string
				if context != "" {
					sumti = append([]string{}, context)
				}
				sumti = append(sumti, s)
				sumti = append(sumti, o)

				r := Selbri{
					Predicate: v,
					Arguments: sumti,
				}

				result = append(result, r)
			}
		}
	}

	return result, nil
}
