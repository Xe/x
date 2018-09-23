package main

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type part struct {
	Part   string   `json:"part"`
	Sep    *string  `json:"sep"`
	Tokens []string `json:"tokens"`
	Parts  []*part  `json:"parts"`
}

const (
	partAddress      = `address`
	partSubject      = `subject`
	partObjectMarker = `objectMarker`
	partPrepPhrase   = `prepPhrase`
	partInterjection = `interjection`
	partCartouche    = `cartouche`
	partPunctuation  = `punctuation`

	punctPeriod      = `period`
	punctQuestion    = `question`
	punctExclamation = `exclamation`
)

// A sentence is a series of sentence parts.
type Sentence []part

// TokenizeTokiPona returns a series of toki pona tokens.
func TokenizeTokiPona(aurl, text string) ([]Sentence, error) {
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
