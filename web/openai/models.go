package openai

import (
	"context"
	"encoding/json"
	"net/http"
)

type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ListModels returns a list of all models available to the user.
func ListModels(ctx context.Context, cli *http.Client, openAIKey string) ([]Model, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return nil, err
	}

	type response struct {
		Object string  `json:"object"`
		Data   []Model `json:"data"`
	}

	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data, nil
}
