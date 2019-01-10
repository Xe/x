package tokiponatokens

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

// Part is an individual part of a sentence.
type Part struct {
	Type   string   `json:"part"`
	Sep    *string  `json:"sep"`
	Tokens []string `json:"tokens"`
	Parts  []*Part  `json:"parts"`
}

func (p Part) Braces() string {
	switch p.Type {
	case PartPunctuation:
		switch p.Tokens[0] {
		case PunctExclamation:
			return "!"
		case PunctPeriod:
			return "."
		case PunctQuestion:
			return "?"
		case PunctComma:
			return ","
		case "la":
			return "la"
		}

		panic("unknown punctuation " + p.Tokens[0])
	case PartAddress:
		if p.Parts == nil {
			if p.Sep == nil {
				return strings.Join(p.Tokens, " ")
			}

			return strings.Title(strings.Join(p.Tokens, ""))
		}
	}

	var sb strings.Builder

	if p.Sep != nil {
		sb.WriteString(*p.Sep)
		sb.WriteRune(' ')
	}

	if len(p.Tokens) != 0 {
		sb.WriteString(strings.Join(p.Tokens, " "))
		sb.WriteRune(' ')
	}

	for _, pt := range p.Parts {
		sb.WriteRune('^')
		sb.WriteString(strings.TrimSpace(pt.Braces()))
		sb.WriteRune('^')
		sb.WriteRune(' ')
	}

	return sb.String()
}

// Individual part type values.
const (
	// Who/what the sentence is addressed to in Parts.
	PartAddress      = `address`
	PartSubject      = `subject`
	PartObjectMarker = `objectMarker`
	PartVerbMarker   = `verbMarker`
	PartPrepPhrase   = `prepPhrase`
	PartInterjection = `interjection`
	// A foreign name.
	PartCartouche = `cartouche`
	// Most sentences will end in this.
	PartPunctuation = `punctuation`
)

// Punctuation constants.
const (
	PunctPeriod      = `period`
	PunctQuestion    = `question`
	PunctExclamation = `exclamation`
	PunctComma       = `comma`
)

// Sentence is a series of sentence parts. This correlates to one Toki Pona sentence.
type Sentence []Part

// Tokenize returns a series of toki pona tokens.
func Tokenize(aurl, text string) ([]Sentence, error) {
	buf := bytes.NewBuffer([]byte(text))
	req, err := http.NewRequest(http.MethodPost, aurl, buf)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "text/plain")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result []Sentence
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
