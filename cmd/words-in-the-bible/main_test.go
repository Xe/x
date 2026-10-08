package main

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
)

func TestScanWords(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		input string
		want  []string
	}{
		{name: "empty"},
		{name: "separators only", input: " \t\r\n,;‘’“”"},
		{name: "punctuation", input: "God,love.peace:hope;faith?joy!(light)-\"truth\"'life'‘good’“day”", want: []string{"God", "love", "peace", "hope", "faith", "joy", "light", "truth", "life", "good", "day"}},
		{name: "Unicode whitespace", input: "God\tpeace\r\nlove\u00a0faith\u2003hope", want: []string{"God", "peace", "love", "faith", "hope"}},
		{name: "Unicode words", input: "Éden‘κόσμος’日本語", want: []string{"Éden", "κόσμος", "日本語"}},
		{name: "buffer boundary", input: strings.Repeat("a", 10000) + " peace", want: []string{strings.Repeat("a", 10000), "peace"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, chunked := range []bool{false, true} {
				input := strings.NewReader(tt.input)
				sc := wordScanner(input)
				if chunked {
					sc = wordScanner(iotest.OneByteReader(input))
				}
				var got []string
				for sc.Scan() {
					got = append(got, sc.Text())
				}
				if err := sc.Err(); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Fatalf("chunked=%v: got %q, want %q", chunked, got, tt.want)
				}
			}
		})
	}
}

func TestCountWords(t *testing.T) {
	t.Parallel()
	bibleWords := loadBibleWords()
	for _, tt := range []struct {
		name         string
		input        string
		found, total int
	}{
		{name: "empty"},
		{name: "partial", input: "GOD\tlove\u2003peace qzxvnonword", found: 3, total: 4},
		{name: "full", input: "God,God!", found: 2, total: 2},
		{name: "no matches", input: "qzxvnonword", total: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			found, total, err := countWords(strings.NewReader(tt.input), bibleWords)
			if err != nil {
				t.Fatal(err)
			}
			if found != tt.found || total != tt.total {
				t.Fatalf("got %d/%d, want %d/%d", found, total, tt.found, tt.total)
			}
		})
	}
}

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

func TestCountWordsReadError(t *testing.T) {
	t.Parallel()
	want := errors.New("read failed")
	_, _, err := countWords(failingReader{want}, nil)
	if !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

func TestBibleVocabulary(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("data/bible/kjv.txt")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct{}{}
	for _, word := range strings.FieldsFunc(string(data), isWordSeparator) {
		want[strings.ToLower(word)] = struct{}{}
	}
	if got := loadBibleWords(); !reflect.DeepEqual(got, want) {
		t.Fatal("embedded vocabulary differs from source; run go generate")
	}
}

func FuzzScanWords(f *testing.F) {
	for _, input := range []string{"", "God‘peace’", "one\tTWO\u2003three", "a\xffb"} {
		f.Add(input)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) >= 32768 {
			t.Skip("input approaches Scanner's maximum token size")
		}
		sc := wordScanner(iotest.OneByteReader(strings.NewReader(input)))
		var got []string
		for sc.Scan() {
			got = append(got, sc.Text())
		}
		if err := sc.Err(); err != nil {
			t.Fatal(err)
		}
		want := strings.FieldsFunc(input, isWordSeparator)
		if len(got) != len(want) {
			t.Fatalf("got %q, want %q", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("got %q, want %q", got, want)
			}
		}
	})
}
