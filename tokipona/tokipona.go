package tokipona

// Toki li wan toki.
//
// Toki is a sentence.
type Toki struct {
	Subject []Nimi
	Verb    []Nimi
	Object  []Nimi
}

// Nimi is a single word in Toki Pona.
type Nimi struct {
	Nimi string `json:"nimi"`

	Particle     string   `json:"particle"`
	Nouns        []string `json:"nouns"`
	Adjectives   []string `json:"adjectives"`
	PreVerbs     []string `json:"preverbs"`
	Verbs        []string `json:"verbs"`
	Prepositions []string `json:"prepositions"`
}
