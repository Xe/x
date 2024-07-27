package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"within.website/x/valid"
	"within.website/x/web"
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

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result CompleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ollama: error decoding response: %w", err)
	}

	return &result, nil
}

// HallucinateOpts contains the options for the Hallucinate function.
type HallucinateOpts struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

func p[T any](v T) *T {
	return &v
}

// Hallucinate prompts the model to hallucinate a "valid" JSON response to the given input.
func Hallucinate[T valid.Interface](ctx context.Context, c *Client, opts HallucinateOpts) (*T, error) {
	inp := &CompleteRequest{
		Model:    opts.Model,
		Messages: opts.Messages,
		Format:   p("json"),
		Stream:   true,
	}
	tries := 0
	for tries <= 5 {
		tries++

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
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

		if resp.StatusCode != http.StatusOK {
			return nil, web.NewError(http.StatusOK, resp)
		}

		whitespaceCount := 0

		dec := json.NewDecoder(resp.Body)
		buf = bytes.NewBuffer(nil)

		for {
			var cr CompleteResponse
			err := dec.Decode(&cr)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					return nil, fmt.Errorf("ollama: error decoding response: %w", err)
				} else {
					break
				}
			}

			//slog.Debug("got response", "response", cr.Message.Content)

			if _, err := fmt.Fprint(buf, cr.Message.Content); err != nil {
				return nil, fmt.Errorf("ollama: error writing response to buffer: %w", err)
			}

			for _, r := range cr.Message.Content {
				if r == '\n' {
					whitespaceCount++
				}
			}

			if whitespaceCount > 10 {
				cancel()
			}

			//slog.Debug("buffer is now", "buffer", buf.String())

			var result T
			if err := json.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(&result); err != nil {
				//slog.Debug("error decoding response", "err", err)
				continue
			}

			if err := result.Valid(); err != nil {
				slog.Debug("error validating response", "err", err)
				continue
			}

			//slog.Debug("got valid response", "response", result)
			cancel()

			return &result, nil
		}
	}

	return nil, fmt.Errorf("ollama: failed to hallucinate a valid response after 5 tries")
}

type EmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`

	Options map[string]any `json:"options"`
}

type EmbedResponse struct {
	Embedding []float64 `json:"embedding"`
}

func (c *Client) Embeddings(ctx context.Context, er *EmbedRequest) (*EmbedResponse, error) {
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(er); err != nil {
		return nil, fmt.Errorf("ollama: error encoding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/embeddings", buf)
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

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ollama: error decoding response: %w", err)
	}

	return &result, nil
}

type GenerateRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Images  [][]byte       `json:"images,omitempty"`
	Options map[string]any `json:"options"`

	Context   []int   `json:"context,omitempty"`
	Format    *string `json:"format,omitempty"`
	Template  *string `json:"template,omitempty"`
	System    *string `json:"system,omitempty"`
	Stream    bool    `json:"stream"`
	Raw       bool    `json:"raw"`
	KeepAlive string  `json:"keep_alive"`
}

type GenerateResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Response           string    `json:"response"`
	Done               bool      `json:"done"`
	Context            []int     `json:"context"`
	TotalDuration      int64     `json:"total_duration"`
	LoadDuration       int64     `json:"load_duration"`
	PromptEvalCount    int       `json:"prompt_eval_count"`
	PromptEvalDuration int64     `json:"prompt_eval_duration"`
	EvalCount          int       `json:"eval_count"`
	EvalDuration       int64     `json:"eval_duration"`
}

func (c *Client) Generate(ctx context.Context, gr *GenerateRequest) (*GenerateResponse, error) {
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(gr); err != nil {
		return nil, fmt.Errorf("ollama: error encoding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", buf)
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

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ollama: error decoding response: %w", err)
	}

	return &result, nil
}
