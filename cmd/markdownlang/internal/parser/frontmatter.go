package parser

import (
	"errors"
	"strings"
)

// extractFrontMatter finds and extracts YAML front matter between --- delimiters.
func extractFrontMatter(source string) ([]byte, string, error) {
	lines := strings.Split(source, "\n")

	if len(lines) < 3 {
		return nil, "", errors.New("file too short to contain front matter. did you even try?")
	}

	// Find opening delimiter
	startIdx := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			startIdx = i
			break
		}
	}

	if startIdx == -1 {
		return nil, "", errors.New("no opening --- delimiter found. your front matter is missing in action")
	}

	// Find closing delimiter
	endIdx := -1
	for i := startIdx + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIdx = i
			break
		}
	}

	if endIdx == -1 {
		return nil, "", errors.New("no closing --- delimiter found. your front matter is eternal")
	}

	if endIdx == startIdx+1 {
		return nil, "", errors.New("front matter is empty. put something between the --- lines")
	}

	// Extract front matter
	fmLines := lines[startIdx+1 : endIdx]
	frontMatter := strings.Join(fmLines, "\n")

	// Extract content
	content := strings.Join(lines[endIdx+1:], "\n")

	return []byte(frontMatter), content, nil
}
