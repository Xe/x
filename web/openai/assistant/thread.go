package assistant

import (
	"context"
	"encoding/json"
	"net/http"

	"within.website/x/web"
)

type Thread struct {
	ID        string            `json:"id"`
	Object    string            `json:"object"`
	CreatedAt int64             `json:"created_at"`
	Metadata  map[string]string `json:"metadata"`
}

func (c *Client) CreateThread(ctx context.Context) (*Thread, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/threads", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	var t Thread
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return nil, err
	}

	return &t, nil
}

func (c *Client) GetThread(ctx context.Context, id string) (*Thread, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/threads/"+id, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	var t Thread
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	return &t, nil
}

func (c *Client) UpdateThread(ctx context.Context, id string, metadata map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, "https://api.openai.com/v1/threads/"+id, nil)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return web.NewError(http.StatusOK, resp)
	}

	return nil
}

func (c *Client) DeleteThread(ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, "https://api.openai.com/v1/threads/"+id, nil)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return web.NewError(http.StatusOK, resp)
	}

	return nil
}
