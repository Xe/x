package main

import (
	"embed"
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
	dataDir  = flag.String("data-dir", "./var", "data directory for cached views of words in the bible")
	holyText = flag.String("holy-text", "bible", "which holy text to use (bible, hpmor, quran)")

	//go:embed data/words/*.txt
	holyTexts embed.FS
)

func loadWords(fname string) (map[string]struct{}, error) {
	fname = fmt.Sprintf("data/words/%s.txt", fname)
	data, err := holyTexts.ReadFile(fname)
	if err != nil {
		return nil, fmt.Errorf("read vocabulary %q: %w", fname, err)
	}

	vocabulary := string(data)
	words := make(map[string]struct{}, strings.Count(vocabulary, "\n"))
	for word := range strings.SplitSeq(strings.TrimSuffix(vocabulary, "\n"), "\n") {
		words[word] = struct{}{}
	}
	return words, nil
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

	words, err := loadWords(*holyText)
	if err != nil {
		slog.Error("failed to read words", "err", err, "holy-text", *holyText)
		os.Exit(1)
	}

	foundWords, totalWords, err := countWords(os.Stdin, words)
	if err != nil {
		slog.Error("failed to read input", "err", err)
		os.Exit(1)
	}

	percentage := 0.0
	if totalWords > 0 {
		percentage = float64(foundWords) / float64(totalWords) * 100
	}
	fmt.Printf("%d of %d of those words are in the %s (%.2f%%)\n", foundWords, totalWords, *holyText, percentage)
}
