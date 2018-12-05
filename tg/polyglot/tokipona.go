package main

import (
	"strings"

	"github.com/Xe/x/web/tokiponatokens"
)

func tokiBraces(s tokiponatokens.Sentence) string {
	var sb strings.Builder
	sb.WriteRune('(')

	for i, pt := range s {
		switch pt.Type {
		case tokiponatokens.PartSubject:
			sb.WriteRune('{')
			sb.WriteString(strings.TrimSpace(pt.Braces()))
			sb.WriteRune('}')

		case tokiponatokens.PartObjectMarker:
			sb.WriteRune('[')
			sb.WriteString(strings.TrimSpace(pt.Braces()))
			sb.WriteRune(']')

		case tokiponatokens.PartPunctuation:
			sb.WriteString(strings.TrimSpace(pt.Braces()))

		default:
			sb.WriteRune('<')
			sb.WriteString(strings.TrimSpace(pt.Braces()))
			sb.WriteRune('>')
		}

		if i+1 != len(s) {
			sb.WriteRune(' ')
		}
	}

	sb.WriteRune(')')
	return sb.String()
}
