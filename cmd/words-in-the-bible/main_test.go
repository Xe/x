package main

import (
	"errors"
	"io/fs"
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
	for _, text := range []string{"bible", "hpmor", "quran"} {
		t.Run(text, func(t *testing.T) {
			t.Parallel()
			words, err := loadWords(text)
			if err != nil {
				t.Fatal(err)
			}
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
					found, total, err := countWords(strings.NewReader(tt.input), words)
					if err != nil {
						t.Fatal(err)
					}
					if found != tt.found || total != tt.total {
						t.Fatalf("got %d/%d, want %d/%d", found, total, tt.found, tt.total)
					}
				})
			}
		})
	}
}

func TestSelectedText(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		text, input string
		found       int
	}{
		{text: "bible", input: "Harry", found: 0},
		{text: "bible", input: "Allah", found: 0},
		{text: "hpmor", input: "Harry", found: 1},
		{text: "hpmor", input: "Allah", found: 0},
		{text: "quran", input: "Harry", found: 0},
		{text: "quran", input: "Allah", found: 1},
	} {
		t.Run(tt.text+"/"+tt.input, func(t *testing.T) {
			words, err := loadWords(tt.text)
			if err != nil {
				t.Fatal(err)
			}
			found, total, err := countWords(strings.NewReader(tt.input), words)
			if err != nil {
				t.Fatal(err)
			}
			if found != tt.found || total != 1 {
				t.Fatalf("got %d/%d, want %d/1", found, total, tt.found)
			}
		})
	}
}

func TestLoadWordsUnknownText(t *testing.T) {
	t.Parallel()
	words, err := loadWords("unknown")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("got %v, want file-not-found error", err)
	}
	if words != nil {
		t.Fatal("failed load returned a vocabulary")
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

func TestTextVocabularies(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		text, source string
	}{
		{text: "bible", source: "data/bible/kjv.txt"},
		{text: "hpmor", source: "data/hpmor/hpmor.txt"},
		{text: "quran", source: "data/quran/quran.txt"},
	} {
		t.Run(tt.text, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(tt.source)
			if err != nil {
				t.Fatal(err)
			}
			want := map[string]struct{}{}
			for _, word := range strings.FieldsFunc(string(data), isWordSeparator) {
				want[strings.ToLower(word)] = struct{}{}
			}
			got, err := loadWords(tt.text)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatal("embedded vocabulary differs from source; run go generate")
			}
		})
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
