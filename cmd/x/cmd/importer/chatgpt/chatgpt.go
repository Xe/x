package chatgpt

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/subcommands"
)

// The overall Conversation structure
type Conversation struct {
	Title       string          `json:"title"`
	CreateTime  float64         `json:"create_time"`
	UpdateTime  float64         `json:"update_time"`
	Mapping     map[string]Node `json:"mapping"`
	CurrentNode string          `json:"current_node"`
	ID          string          `json:"id"`
	// Other fields are omitted for simplicity
}

// Node represents a single entry in the 'mapping' part of the JSON.
type Node struct {
	ID       string   `json:"id"`
	Message  *Message `json:"message"`
	Parent   *string  `json:"parent"`
	Children []string `json:"children"`
}

// Message represents the actual message content and metadata.
type Message struct {
	ID         string   `json:"id"`
	Author     Author   `json:"author"`
	CreateTime *float64 `json:"create_time"`
	UpdateTime *float64 `json:"update_time"`
	Content    Content  `json:"content"`
	Status     string   `json:"status"`
	EndTurn    *bool    `json:"end_turn"`
	Weight     float64  `json:"weight"`
	Metadata   Metadata `json:"metadata"`
	Recipient  string   `json:"recipient"`
	Channel    *string  `json:"channel"`
}

// Author contains information about who sent the message.
type Author struct {
	Role string `json:"role"`
}

// Content holds the message's text or other parts.
type Content struct {
	ContentType string        `json:"content_type"`
	Parts       []interface{} `json:"parts"`
}

// Metadata stores extra information about the message.
type Metadata struct {
	MessageType *string `json:"message_type"`
	Prompt      *string `json:"prompt"`
}

// GetMessages returns a slice of Message objects in chronological order.
func (c *Conversation) GetMessages() []*Message {
	var orderedMessages []*Message
	currentNodeID := c.CurrentNode

	for {
		node, exists := c.Mapping[currentNodeID]
		if !exists {
			break
		}

		if node.Message != nil {
			orderedMessages = append(orderedMessages, node.Message)
		}

		if node.Parent == nil {
			break
		}
		currentNodeID = *node.Parent
	}

	for i, j := 0, len(orderedMessages)-1; i < j; i, j = i+1, j-1 {
		orderedMessages[i], orderedMessages[j] = orderedMessages[j], orderedMessages[i]
	}

	if len(orderedMessages) == 0 {
		var allMessages []*Message
		for _, node := range c.Mapping {
			if node.Message != nil && node.Message.CreateTime != nil {
				allMessages = append(allMessages, node.Message)
			}
		}

		sort.Slice(allMessages, func(i, j int) bool {
			return *allMessages[i].CreateTime < *allMessages[j].CreateTime
		})
		return allMessages
	}

	return orderedMessages
}

// ImportCmd holds the command-line flags for the subcommand.
type ImportCmd struct {
	jsonFile   string
	outputFile string
}

// Name returns the name of the subcommand.
func (*ImportCmd) Name() string { return "import-chatgpt" }

// Synopsis returns a short description of the subcommand.
func (*ImportCmd) Synopsis() string {
	return "Renders a JSON conversation history to a Markdown file."
}

// Usage returns a detailed help message for the subcommand.
func (*ImportCmd) Usage() string {
	return `import-chatgpt --json_file <path-to-json-file> --output_file <path-to-output-file>
  Renders a JSON chat history to a Markdown file.
`
}

// SetFlags is used to define command-line flags.
func (p *ImportCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.jsonFile, "json_file", "conversations.json", "The path to the JSON conversation file.")
	f.StringVar(&p.outputFile, "output_file", "conversations.md", "The path to the output Markdown file.")
}

// Execute is the main logic of the subcommand.
func (p *ImportCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	data, err := os.ReadFile(p.jsonFile)
	if err != nil {
		log.Printf("error reading file: %v", err)
		return subcommands.ExitFailure
	}

	var conversations []Conversation
	err = json.Unmarshal(data, &conversations)
	if err != nil {
		log.Printf("error unmarshaling JSON: %v", err)
		return subcommands.ExitFailure
	}

	fmt.Printf("Successfully read %d conversations from file.\n", len(conversations))

	output, err := os.Create(p.outputFile)
	if err != nil {
		log.Printf("error creating output file: %v", err)
		return subcommands.ExitFailure
	}
	defer output.Close()

	for i, conv := range conversations {
		output.WriteString(fmt.Sprintf("# Conversation %d\n\n", i+1))
		if conv.Title != "" {
			output.WriteString(fmt.Sprintf("## %s\n\n", conv.Title))
		}

		messages := conv.GetMessages()
		for _, msg := range messages {
			if msg.CreateTime != nil && len(msg.Content.Parts) > 0 {
				ts := time.Unix(int64(*msg.CreateTime), 0)
				role := "**" + msg.Author.Role + "**"
				contentParts := []string{}
				for _, part := range msg.Content.Parts {
					contentParts = append(contentParts, fmt.Sprintf("%v", part))
				}
				content := strings.Join(contentParts, "\n")

				output.WriteString(fmt.Sprintf("### %s (%s)\n%s\n\n", role, ts.Format("2006-01-02 15:04:05"), content))
			}
		}
	}

	fmt.Printf("Successfully rendered conversations to %s\n", p.outputFile)
	return subcommands.ExitSuccess
}
