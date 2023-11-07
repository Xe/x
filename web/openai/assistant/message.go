package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"within.website/x/web"
)

type Content struct {
	Type string `json:"type"`
	Text Text   `json:"text"`
}

type Text struct {
	Value       string          `json:"value"`
	Annotations json.RawMessage `json:"annotations"`
}

type Message struct {
	ID          string            `json:"id"`
	Object      string            `json:"object"`
	CreatedAt   int64             `json:"created_at"`
	ThreadID    string            `json:"thread_id"`
	Role        string            `json:"role"`
	Content     []Content         `json:"content"`
	FileIDs     []string          `json:"file_ids"`
	AssistantID *string           `json:"assistant_id,omitempty"`
	RunID       *string           `json:"run_id,omitempty"`
	Metadata    map[string]string `json:"metadata"`
}

type CreateMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (c *Client) CreateMessage(ctx context.Context, tid string, cm CreateMessage) (*Message, error) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(cm); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/threads/"+tid+"/messages", buf)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var m Message
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}

	return &m, nil
}

func (c *Client) GetMessage(ctx context.Context, tid, mid string) (*Message, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/threads/"+tid+"/messages/"+mid, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var m Message
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}

	return &m, nil
}

func (c *Client) ListMessages(ctx context.Context, tid string) ([]Message, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/threads/"+tid+"/messages", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	type response struct {
		Data []Message `json:"data"`
	}

	var result response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}
