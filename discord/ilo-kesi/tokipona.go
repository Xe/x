package main

import (
	"encoding/json"
	"os"
	"strings"
)

type Word struct {
	Name     string   `json:"name"`
	Gloss    string   `json:"gloss"`
	Grammar  []string `json:"grammar"`
	Category string   `json:"category,omitempty"`
	Type     string   `json:"type,omitempty"`
}

func loadWords(fname string) ([]Word, error) {
	fin, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer fin.Close()

	var result []Word
	err = json.NewDecoder(fin).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (i ilo) tokiNiTokiPonaAnuSeme(lipu string) bool {
	return strings.HasPrefix(lipu, "ilo Kesi o,")
}
