package tokipona

import (
	"strings"
	"unicode"
)

// Relex does a very literal translation of Toki Pona to English. See http://tokipona.net/tp/Relex.aspx for the idea.
func Relex(inp string) string {
	f := func(c rune) bool {
		return !unicode.IsLetter(c)
	}

	words := strings.FieldsFunc(inp, f)
	result := []string{}

	for _, word := range words {
		if en, ok := relexMap[word]; ok {
			result = append(result, en)
		} else {
			result = append(result, word)
		}

	}

	return strings.Join(result, " ")
}
