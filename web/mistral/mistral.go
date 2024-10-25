package mistral

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"within.website/x/llm"
	"within.website/x/web"
)

type Client struct {
	*http.Client
	apiKey string
}

func NewClient(apiKey string) *Client {
	return &Client{
		Client: &http.Client{},
		apiKey: apiKey,
	}
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	return c.Client.Do(req)
}

type CompleteRequest struct {
	Model       string        `json:"model"`
	Messages    []llm.Message `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
	TopP        *float64      `json:"top_p,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	Stream      *bool         `json:"stream,omitempty"`
	SafeMode    *bool         `json:"safe_mode,omitempty"`
	RandomSeed  *int          `json:"random_seed,omitempty"`
}

type CompleteResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []CompletionChoice `json:"choices"`
	Usage   UsageInfo          `json:"usage"`
}

type CompletionChoice struct {
	Index        int       `json:"index"`
	Message      []Message `json:"message"`
	FinishReason string    `json:"finish_reason"`
}

type Message struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (c *Client) Chat(ctx context.Context, req *CompleteRequest) (*CompleteResponse, error) {
	var data bytes.Buffer
	if err := json.NewEncoder(&data).Encode(req); err != nil {
		return nil, fmt.Errorf("mistral: error encoding request: %w", err)
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.mistral.chat/v1/chat/completions", &data)
	if err != nil {
		return nil, fmt.Errorf("mistral: error creating request: %w", err)
	}

	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")

	resp, err := c.Do(r)
	if err != nil {
		return nil, fmt.Errorf("mistral: error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var res CompleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("mistral: error decoding response: %w", err)
	}

	return &res, nil
}
