// Package multillm is a common interface for doing multiple large
// language model requests with common inputs and types.
package multillm

import (
	"context"

	"within.website/x/llm"
)

type Request struct {
	Model       string        `json:"model"`
	Messages    []llm.Message `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
	RandomSeed  *int          `json:"random_seed,omitempty"`
}

type Response struct {
	Response         llm.Message `json:"response"`
	PromptTokens     int         `json:"prompt_tokens"`
	CompletionTokens int         `json:"completion_tokens"`
}

type Chatter interface {
	Chat(ctx context.Context, req *Request) (*Response, error)
}

type MultiChatModel struct {
	Provider string   `json:"provider"`
	Models   []string `json:"models"`
}

type MultiChatRequest struct {
	Models   []MultiChatModel `json:"models"`
	Messages []llm.Message    `json:"messages"`
}
