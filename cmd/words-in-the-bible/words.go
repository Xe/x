package main

import (
	"bufio"
	"io"
	"unicode"
	"unicode/utf8"
)

func isWordSeparator(r rune) bool {
	switch r {
	case ',', '.', ':', ';', '?', '!', '(', ')', '-', '"', '\'', '”', '“', '‘', '’':
		return true
	default:
		return unicode.IsSpace(r)
	}
}

func wordScanner(input io.Reader) *bufio.Scanner {
	sc := bufio.NewScanner(input)
	sc.Split(scanWords)
	return sc
}

// scanWords splits on whitespace and the punctuation used by the source texts.
func scanWords(data []byte, atEOF bool) (advance int, token []byte, err error) {
	start := 0
	for start < len(data) {
		if !atEOF && !utf8.FullRune(data[start:]) {
			return start, nil, nil
		}
		r, size := utf8.DecodeRune(data[start:])
		if !isWordSeparator(r) {
			break
		}
		start += size
	}

	for end := start; end < len(data); {
		if !atEOF && !utf8.FullRune(data[end:]) {
			return start, nil, nil
		}
		r, size := utf8.DecodeRune(data[end:])
		if isWordSeparator(r) {
			return end + size, data[start:end], nil
		}
		end += size
	}

	if atEOF && start < len(data) {
		return len(data), data[start:], nil
	}
	return start, nil, nil
}
