// Package chatgpt is a simple binding to the ChatGPT API.
package chatgpt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"within.website/x/web"
)

type Request struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (m Message) ProxyFormat() string {
	return fmt.Sprintf("%s\\ %s", strings.Title(m.Role), m.Content)
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type Choice struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
	Index        int     `json:"index"`
}

type Response struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int      `json:"created"`
	Model   string   `json:"model"`
	Usage   Usage    `json:"usage"`
	Choices []Choice `json:"choices"`
}

type Client struct {
	httpCli *http.Client
	apiKey  string
}

func NewClient(apiKey string) Client {
	return Client{
		httpCli: &http.Client{},
		apiKey:  apiKey,
	}
}

func (c Client) Complete(ctx context.Context, r Request) (*Response, error) {
	if r.Model == "" {
		r.Model = "gpt-3.5-turbo"
	}

	data, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("chatgpt: [unexpected] can't make request???: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+c.apiKey)
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.httpCli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chatgpt: can't reach API: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}
	defer resp.Body.Close()

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("chatgpt: can't decode result: %w", err)
	}

	return &result, nil
}
