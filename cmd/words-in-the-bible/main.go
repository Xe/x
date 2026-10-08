package main

import (
	_ "embed"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"within.website/x/internal"
)

//go:generate go run generate.go words.go

var (
	dataDir = flag.String("data-dir", "./var", "data directory for cached views of words in the bible")

	//go:embed bible_words.txt
	bibleVocabulary string
)

func loadBibleWords() map[string]struct{} {
	words := make(map[string]struct{}, strings.Count(bibleVocabulary, "\n"))
	for word := range strings.SplitSeq(strings.TrimSuffix(bibleVocabulary, "\n"), "\n") {
		words[word] = struct{}{}
	}
	return words
}

func countWords(input io.Reader, bibleWords map[string]struct{}) (found, total int, err error) {
	sc := wordScanner(input)
	for sc.Scan() {
		total++
		if _, ok := bibleWords[strings.ToLower(sc.Text())]; ok {
			found++
		}
	}
	return found, total, sc.Err()
}

func main() {
	internal.HandleStartup()

	foundWords, totalWords, err := countWords(os.Stdin, loadBibleWords())
	if err != nil {
		slog.Error("failed to read input", "err", err)
		os.Exit(1)
	}

	percentage := 0.0
	if totalWords > 0 {
		percentage = float64(foundWords) / float64(totalWords) * 100
	}
	fmt.Printf("%d of %d of those words are in the bible (%.2f%%)\n", foundWords, totalWords, percentage)
}
