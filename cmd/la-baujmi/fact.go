package main

import (
	"errors"
	"strings"

	"within.website/x/web/tokiponatokens"
)

// Bridi is a predicate relationship between its arguments. The name comes from
// the lojban term selbri: http://jbovlaste.lojban.org/dict/selbri.
type Bridi struct {
	Predicate string
	Arguments []string
}

// Fact converts a selbri into a prolog fact.
func (s Bridi) Fact() string {
	var sb strings.Builder

	var varCount byte

	sb.WriteString("bridi(verb(")

	if s.Predicate == "seme" {
		sb.WriteByte(byte('A') + varCount)
		varCount++
	} else {
		sb.WriteString(s.Predicate)
	}
	sb.WriteString("), ")

	for i, arg := range s.Arguments {
		if i != 0 {
			sb.WriteString(", ")
		}

		switch arg {
		case "subject(seme)", "object(seme)":
			if strings.HasPrefix(arg, "subject") {
				sb.WriteString("subject(")
			} else {
				sb.WriteString("object(")
			}
			sb.WriteByte(byte('A') + varCount)
			varCount++
			sb.WriteByte(')')
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

// SentenceToBridis creates logical facts derived from toki pona sentences.
// This is intended to be the first step in loading them into prolog.
func SentenceToBridis(s tokiponatokens.Sentence) ([]Bridi, error) {
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
				temp := strings.Split(strings.Join(pt.Tokens, " "), " en ")

				for _, t := range temp {
					subjects = append(subjects, "subject("+t+")")
				}

				continue
			}

			if len(pt.Parts) != 0 {
				var sb strings.Builder
				sb.WriteString("subject(")

				for i, sp := range pt.Parts {
					if i != 0 {
						sb.WriteString(", ")
					}
					if sp.Sep != nil && *sp.Sep == "pi" {
						sb.WriteString("pi(")
						for j, tk := range sp.Tokens {
							if j != 0 {
								sb.WriteString(", ")
							}

							sb.WriteString(tk)
						}
						sb.WriteString(")")
					} else {
						sb.WriteString(strings.Join(sp.Tokens, "_"))
					}
				}

				sb.WriteString(")")

				subjects = append(objects, sb.String())
				continue
			}

			subjects = append(subjects, "subject("+strings.Join(pt.Tokens, "_")+")")

		case tokiponatokens.PartVerbMarker:
			verbs = append(verbs, strings.Join(pt.Tokens, "_"))

		case tokiponatokens.PartObjectMarker:
			if len(pt.Parts) != 0 {
				var sb strings.Builder
				sb.WriteString("object(")

				for i, sp := range pt.Parts {
					if i != 0 {
						sb.WriteString(", ")
					}
					if sp.Sep != nil && *sp.Sep == "pi" {
						sb.WriteString("pi(")
						for j, tk := range sp.Tokens {
							if j != 0 {
								sb.WriteString(", ")
							}

							sb.WriteString(tk)
						}
						sb.WriteString(")")
					} else {
						sb.WriteString(strings.Join(sp.Tokens, "_"))
					}
				}

				sb.WriteString(")")

				objects = append(objects, sb.String())
				continue
			}
			objects = append(objects, "object("+strings.Join(pt.Tokens, "_")+")")

		case tokiponatokens.PartPunctuation:
			if len(pt.Tokens) == 1 {
				switch pt.Tokens[0] {
				case "la":
					context = "context(" + subjects[len(subjects)-1] + ")"
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

	var result []Bridi

	for _, v := range verbs {
		for _, s := range subjects {
			if len(objects) == 0 {
				// sumti: x1 is a/the argument of predicate function x2 filling place x3 (kind/number)
				var sumti []string
				if context != "" {
					sumti = append([]string{}, context)
				}
				sumti = append(sumti, s)

				r := Bridi{
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

				r := Bridi{
					Predicate: v,
					Arguments: sumti,
				}

				result = append(result, r)
			}
		}
	}

	return result, nil
}
