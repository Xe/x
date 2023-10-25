package llm

import (
	"fmt"
	"strings"
)

type Session struct {
	Messages []ChatMLer `json:"messages"`
}

type ChatMLer interface {
	ChatML() string
}

func (s Session) ChatML() string {
	var sb strings.Builder

	for _, message := range s.Messages {
		fmt.Fprintf(&sb, "%s\n", message.ChatML())
	}

	return sb.String()
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (m Message) ChatML() string {
	if m.Content == "" {
		return fmt.Sprintf("<|im_start|>%s\n", m.Role)
	}
	return fmt.Sprintf("<|im_start|>%s\n%s<|im_end|>", m.Role, m.Content)
}
