package tokipona

import "testing"

func TestLoadWords(t *testing.T) {
	_, err := LoadWords()
	if err != nil {
		t.Fatal(err)
	}
}
