package tokipona

import (
	"bytes"
	"embed"
	"encoding/json"
)

//go:embed tokipona.json
var static embed.FS

// Word is a single word in the Toki Pona dictionary.
type Word struct {
	Name     string   `json:"name"`
	Gloss    string   `json:"gloss"`
	Grammar  []string `json:"grammar"`
	Category string   `json:"category,omitempty"`
	Type     string   `json:"type,omitempty"`
}

// LoadWords loads the embedded Toki Pona dictionary.
func LoadWords() ([]Word, error) {
	data, err := static.ReadFile("tokipona.json")
	fin := bytes.NewBuffer(data)

	var result []Word
	err = json.NewDecoder(fin).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
