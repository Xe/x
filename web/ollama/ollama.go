package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
	}
}

func NewLocalClient() *Client {
	return NewClient("http://localhost:11434")
}

type Message struct {
	Content string   `json:"content"`
	Role    string   `json:"role"`
	Images  [][]byte `json:"images"`
}

type CompleteRequest struct {
	Model    string         `json:"model"`
	Messages []Message      `json:"messages"`
	Format   *string        `json:"format,omitempty"`
	Template *string        `json:"template,omitempty"`
	Stream   bool           `json:"stream"`
	Options  map[string]any `json:"options"`
}

type CompleteResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Message            Message   `json:"message"`
	Done               bool      `json:"done"`
	TotalDuration      float64   `json:"total_duration"`
	LoadDuration       float64   `json:"load_duration"`
	PromptEvalCount    int64     `json:"prompt_eval_count"`
	PromptEvalDuration int64     `json:"prompt_eval_duration"`
	EvalCount          int64     `json:"eval_count"`
	EvalDuration       int64     `json:"eval_duration"`
}

func (c *Client) Chat(ctx context.Context, inp *CompleteRequest) (*CompleteResponse, error) {
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(inp); err != nil {
		return nil, fmt.Errorf("ollama: error encoding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", buf)
	if err != nil {
		return nil, fmt.Errorf("ollama: error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama: error making request: %w", err)
	}
	defer resp.Body.Close()

	var result CompleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ollama: error decoding response: %w", err)
	}

	return &result, nil
}
