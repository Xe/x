package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"within.website/x/web"
	"within.website/x/web/openai/chatgpt"
)

type Assistant struct {
	ID           string            `json:"id"`
	Object       string            `json:"object"`
	Created      int64             `json:"created"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Model        string            `json:"model"`
	Instructions string            `json:"instructions"`
	Tools        []Tool            `json:"tools"`
	FileIDs      []string          `json:"file_ids"`
	Metadata     map[string]string `json:"metadata"`
}

type Tool struct {
	Type     string            `json:"type"`
	Function *chatgpt.Function `json:"function,omitempty"`
}

type Client struct {
	// OpenAIKey is the API key used to authenticate with the OpenAI API.
	OpenAIKey string

	// Client is the HTTP client used to make requests to the OpenAI API.
	Client *http.Client
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+c.OpenAIKey)
	req.Header.Set("OpenAI-Beta", "assistants=v1")

	return c.Client.Do(req)
}

type CreateAssistant struct {
	Instructions string `json:"instructions"`
	Name         string `json:"name"`
	Tools        []Tool `json:"tools"`
	Model        string `json:"model"`
}

func (c *Client) CreateAssistant(ctx context.Context, ca CreateAssistant) (*Assistant, error) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(ca); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/assistants", buf)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result Assistant
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) GetAssistant(ctx context.Context, id string) (*Assistant, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/assistants/"+id, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result Assistant
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) DeleteAssistant(ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, "https://api.openai.com/v1/assistants/"+id, nil)
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

func (c *Client) ModifyAssistant(ctx context.Context, id string, ca CreateAssistant) (*Assistant, error) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(ca); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/assistants/"+id, buf)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result Assistant
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) ListAssistants(ctx context.Context) ([]Assistant, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/assistants", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result struct {
		Data []Assistant `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}
