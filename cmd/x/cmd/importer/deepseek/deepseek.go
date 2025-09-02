package deepseek

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/subcommands"
)

//==============================================================================
// import-deepseek Subcommand
//==============================================================================

// ImportCmd is an exported struct that implements the subcommands.Command interface.
// It can be registered by any main package that imports this package.
type ImportCmd struct{}

// Name returns the name of the command.
func (*ImportCmd) Name() string { return "import-deepseek" }

// Synopsis returns a short one-line description of the command.
func (*ImportCmd) Synopsis() string { return "Imports DeepSeek JSON conversations into Markdown." }

// Usage returns a detailed help string for the command.
func (*ImportCmd) Usage() string {
	return `import-deepseek [input_json] [output_specifier]

Parses a JSON file containing an array of DeepSeek conversations and converts it to Markdown.
The command operates in two modes based on the number of arguments:

- With 0 arguments:
  Reads 'conversations.json' and writes individual .md files to the 'deepseek-conversations' directory.

- With 2 arguments:
  Reads from [input_json] and writes all conversations into a single [output_specifier] file.
`
}

// SetFlags defines any flags for the command. This command uses positional args, so it's empty.
func (c *ImportCmd) SetFlags(f *flag.FlagSet) {}

// Execute runs the logic of the command.
func (c *ImportCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	log.SetFlags(0)

	if f.NArg() != 0 && f.NArg() != 2 {
		log.Println("Error: this command requires either 0 or 2 arguments.")
		return subcommands.ExitUsageError
	}

	if err := c.run(f); err != nil {
		log.Printf("Error: %v", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

// run contains the core logic for the subcommand.
func (c *ImportCmd) run(f *flag.FlagSet) error {
	if f.NArg() == 0 {
		return c.runDefaultMode()
	}

	if f.NArg() == 2 {
		inputPath := f.Arg(0)
		outputPath := f.Arg(1)
		return c.runSpecifiedMode(inputPath, outputPath)
	}

	return nil // Should not be reached due to the check in Execute
}

// runDefaultMode handles the logic when no arguments are provided.
func (c *ImportCmd) runDefaultMode() error {
	inputPath := "conversations.json"
	outputDir := "deepseek-conversations"

	conversations, err := parseConversationsFromFile(inputPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("could not create output directory '%s': %w", outputDir, err)
	}

	fmt.Printf("Found %d conversations. Writing to directory '%s'...\n", len(conversations), outputDir)
	for _, conv := range conversations {
		fileName := sanitizeFilename(conv.Title) + ".md"
		filePath := filepath.Join(outputDir, fileName)
		file, err := os.Create(filePath)
		if err != nil {
			log.Printf("  - Failed to create file %s: %v\n", filePath, err)
			continue
		}
		defer file.Close()
		generateMarkdownForOne(conv, file)
		fmt.Printf("  - Wrote %s\n", filePath)
	}
	return nil
}

// runSpecifiedMode handles the logic when two arguments are provided.
func (c *ImportCmd) runSpecifiedMode(inputPath, outputPath string) error {
	conversations, err := parseConversationsFromFile(inputPath)
	if err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("could not create output file '%s': %w", outputPath, err)
	}
	defer file.Close()

	fmt.Printf("Found %d conversations. Writing all to '%s'...\n", len(conversations), outputPath)
	for i, conv := range conversations {
		generateMarkdownForOne(conv, file)
		if i < len(conversations)-1 {
			io.WriteString(file, "\n\n---\n\n")
		}
	}
	fmt.Println("Done.")
	return nil
}

//==============================================================================
// JSON Structs and Parsing
//==============================================================================

type Conversation struct {
	ID         string          `json:"id"`
	Title      string          `json:"title"`
	InsertedAt time.Time       `json:"inserted_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	Mapping    map[string]Node `json:"mapping"`
}

type Node struct {
	ID       string   `json:"id"`
	Parent   *string  `json:"parent"`
	Children []string `json:"children"`
	Message  *Message `json:"message"`
}

type Message struct {
	Files      []any      `json:"files"`
	Model      string     `json:"model"`
	InsertedAt time.Time  `json:"inserted_at"`
	Fragments  []Fragment `json:"fragments"`
}

type Fragment struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content,omitempty"`
	Results interface{} `json:"results,omitempty"`
}

type SearchResult struct {
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Snippet     string   `json:"snippet"`
	CiteIndex   *int     `json:"cite_index"`
	PublishedAt *float64 `json:"published_at"`
	SiteName    *string  `json:"site_name"`
	SiteIcon    string   `json:"site_icon"`
}

// parseConversationsFromFile reads and unmarshals conversations from a file.
func parseConversationsFromFile(path string) ([]Conversation, error) {
	var conversations []Conversation
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read input '%s': %w", path, err)
	}
	if err := json.Unmarshal(data, &conversations); err != nil {
		return nil, fmt.Errorf("could not parse JSON from '%s': %w", path, err)
	}
	return conversations, nil
}

//==============================================================================
// Markdown Generation and Utilities
//==============================================================================

func generateMarkdownForOne(conv Conversation, w io.Writer) {
	fmt.Fprintf(w, "# %s\n\n", conv.Title)
	fmt.Fprintf(w, "**ID**: `%s`  \n", conv.ID)
	fmt.Fprintf(w, "**Updated**: %s\n\n", conv.UpdatedAt.Format(time.RFC1123))
	if rootNode, ok := conv.Mapping["root"]; ok {
		traverseAndWrite(conv.Mapping, rootNode, w)
	}
}

func traverseAndWrite(mapping map[string]Node, node Node, w io.Writer) {
	citeMap := make(map[int]SearchResult)
	if node.Message != nil {
		// First pass to collect search results for citation linking.
		for _, fragment := range node.Message.Fragments {
			if fragment.Type == "SEARCH" {
				if results, ok := fragment.Results.([]interface{}); ok {
					for _, res := range results {
						if resultMap, ok := res.(map[string]interface{}); ok {
							var sr SearchResult
							if data, err := json.Marshal(resultMap); err == nil {
								json.Unmarshal(data, &sr)
								if sr.CiteIndex != nil {
									citeMap[*sr.CiteIndex] = sr
								}
							}
						}
					}
				}
			}
		}
		// Second pass to write the markdown.
		for _, fragment := range node.Message.Fragments {
			writeFragment(fragment, w, citeMap)
		}
	}
	for _, childID := range node.Children {
		if childNode, ok := mapping[childID]; ok {
			traverseAndWrite(mapping, childNode, w)
		}
	}
}

func writeFragment(fragment Fragment, w io.Writer, citeMap map[int]SearchResult) {
	switch fragment.Type {
	case "REQUEST":
		if content, ok := fragment.Content.(string); ok {
			fmt.Fprintf(w, "### 👤 Request\n\n%s\n\n", content)
		}
	case "SEARCH":
		fmt.Fprintf(w, "### 🔍 Search Results\n\n")
		if results, ok := fragment.Results.([]interface{}); ok {
			for _, res := range results {
				if resultMap, ok := res.(map[string]interface{}); ok {
					var sr SearchResult
					if data, err := json.Marshal(resultMap); err == nil {
						json.Unmarshal(data, &sr)
						fmt.Fprintf(w, "- **[%s](%s)**\n", sr.Title, sr.URL)
						fmt.Fprintf(w, "  > %s\n\n", sr.Snippet)
					}
				}
			}
		}
	case "THINK":
		if content, ok := fragment.Content.(string); ok {
			fmt.Fprintf(w, "### 🤔 Thought Process\n\n> %s\n\n", strings.ReplaceAll(content, "\n", "\n> "))
		}
	case "RESPONSE":
		if content, ok := fragment.Content.(string); ok {
			linkedContent := replaceCitations(content, citeMap)
			fmt.Fprintf(w, "### 🤖 Response\n\n%s\n\n", linkedContent)
		}
	}
}

func sanitizeFilename(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9\-_]+`)
	sanitized := re.ReplaceAllString(strings.ToLower(name), "-")
	sanitized = strings.Trim(sanitized, "-")
	if len(sanitized) > 80 {
		sanitized = sanitized[:80]
	}
	return sanitized
}

var superscripts = map[string]string{
	"0": "⁰", "1": "¹", "2": "²", "3": "³", "4": "⁴",
	"5": "⁵", "6": "⁶", "7": "⁷", "8": "⁸", "9": "⁹",
}

func replaceCitations(text string, citeMap map[int]SearchResult) string {
	re := regexp.MustCompile(`\[citation:(\d+)]`)
	return re.ReplaceAllStringFunc(text, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		num, _ := strconv.Atoi(parts[1])
		if result, ok := citeMap[num]; ok {
			sup := ""
			for _, char := range parts[1] {
				sup += superscripts[string(char)]
			}
			return fmt.Sprintf(`[%s](%s "%s")`, sup, result.URL, result.Title)
		}
		return match
	})
}
