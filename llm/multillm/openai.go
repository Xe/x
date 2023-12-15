package multillm

import (
	"context"
	"fmt"

	"within.website/x/llm"
	"within.website/x/web/openai/chatgpt"
)

type OpenAI struct {
	*chatgpt.Client
}

func convertToChatGPTMessage(m llm.Message) chatgpt.Message {
	return chatgpt.Message{
		Role:    m.Role,
		Content: m.Content,
	}
}

func (oaic *OpenAI) Chat(ctx context.Context, req *Request) (*Response, error) {
	chatReq := chatgpt.Request{
		Model:       req.Model,
		Temperature: req.Temperature,
		Seed:        req.RandomSeed,
		Messages:    make([]chatgpt.Message, len(req.Messages)),
	}

	for i, m := range req.Messages {
		chatReq.Messages[i] = convertToChatGPTMessage(m)
	}

	chatResp, err := oaic.Client.Complete(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("multillm: error chatting: %w", err)
	}

	return &Response{
		Response: llm.Message{
			Role:    chatResp.Choices[0].Message.Role,
			Content: chatResp.Choices[0].Message.Content,
		},
	}, nil
}
