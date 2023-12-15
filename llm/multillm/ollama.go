package multillm

import (
	"context"

	"within.website/x/llm"
	"within.website/x/web/ollama"
)

type Ollama struct {
	*ollama.Client
}

func (o *Ollama) Chat(ctx context.Context, req *Request) (*Response, error) {
	cr := &ollama.CompleteRequest{
		Model:    req.Model,
		Messages: make([]ollama.Message, len(req.Messages)),
		Options: map[string]any{
			"temperature": req.Temperature,
			"seed":        req.RandomSeed,
		},
	}

	for i, m := range req.Messages {
		cr.Messages[i] = ollama.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	resp, err := o.Client.Chat(ctx, cr)
	if err != nil {
		return nil, err
	}

	return &Response{
		Response: llm.Message{
			Role:    resp.Message.Role,
			Content: resp.Message.Content,
		},
		PromptTokens:     int(resp.PromptEvalCount),
		CompletionTokens: int(resp.EvalCount),
	}, nil
}
