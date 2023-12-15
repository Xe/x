package multillm

import (
	"context"

	"within.website/x/llm"
	"within.website/x/web/mistral"
)

type Mistral struct {
	*mistral.Client
}

func (m *Mistral) Chat(ctx context.Context, req *Request) (*Response, error) {
	cr := &mistral.CompleteRequest{
		Model:       req.Model,
		Messages:    make([]llm.Message, len(req.Messages)),
		Temperature: req.Temperature,
		RandomSeed:  req.RandomSeed,
	}

	for i, m := range req.Messages {
		cr.Messages[i] = llm.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	resp, err := m.Client.Chat(ctx, cr)
	if err != nil {
		return nil, err
	}

	return &Response{
		Response: llm.Message{
			Role:    resp.Choices[0].Message[0].Role,
			Content: resp.Choices[0].Message[0].Content,
		},
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
	}, nil
}
