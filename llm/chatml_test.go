package llm

import (
	"strings"
	"testing"
)

func TestChatML(t *testing.T) {
	session := Session{
		Messages: []ChatMLer{
			Message{
				Role:    "user",
				Content: "hello",
			},
			Message{
				Role: "assistant",
			},
		},
	}

	expected := `<|im_start|>user
hello
<|im_end|>
<|im_start|>assistant`

	if strings.TrimSpace(session.ChatML()) != strings.TrimSpace(expected) {
		t.Errorf("Expected\n\n%s\n\ngot\n\n%s", expected, session.ChatML())
	}
}
