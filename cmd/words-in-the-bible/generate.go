//go:build ignore

// Run with go generate to rebuild the sorted, lowercase Bible vocabulary.
package main

import (
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"within.website/x/internal"
)

func generateVocabulary() error {
	input, err := os.Open("data/bible/kjv.txt")
	if err != nil {
		return fmt.Errorf("open Bible text: %w", err)
	}
	defer input.Close()

	unique := map[string]struct{}{}
	sc := wordScanner(input)
	for sc.Scan() {
		unique[strings.ToLower(sc.Text())] = struct{}{}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("scan Bible text: %w", err)
	}

	words := make([]string, 0, len(unique))
	for word := range unique {
		words = append(words, word)
	}
	slices.Sort(words)
	if err := os.WriteFile("bible_words.txt", []byte(strings.Join(words, "\n")+"\n"), 0644); err != nil {
		return fmt.Errorf("write Bible vocabulary: %w", err)
	}
	return nil
}

func main() {
	internal.HandleStartup()
	if err := generateVocabulary(); err != nil {
		slog.Error("failed to generate Bible vocabulary", "err", err)
		os.Exit(1)
	}
}
