package vastai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
)

type Client struct {
	apiKey string
	cli    *http.Client
	apiURL string
}

func New(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		cli:    &http.Client{},
		apiURL: "https://cloud.vast.ai/api",
	}
}

func (c *Client) WithClient(cli *http.Client) *Client {
	c.cli = cli
	return c
}

func (c *Client) WithAPIURL(apiURL string) *Client {
	c.apiURL = apiURL
	return c
}

// Do performs a HTTP request with the appropriate authentication and user agent headers.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", fmt.Sprintf("go/%s pkg:within.website/x/web/vastai", runtime.Version()))

	return c.cli.Do(req)
}

func (c *Client) doRequestNoResponse(ctx context.Context, method, path string) error {
	req, err := http.NewRequestWithContext(ctx, method, c.apiURL+path, nil)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return NewError(resp)
	}

	return nil
}

// zilch returns the zero value of a given type.
func zilch[T any]() T { return *new(T) }

func doJSON[Output any](ctx context.Context, c *Client, method, path string, wantStatusCode int) (Output, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.apiURL+path, nil)
	if err != nil {
		return zilch[Output](), err
	}

	resp, err := c.Do(req)
	if err != nil {
		return zilch[Output](), err
	}
	defer resp.Body.Close()

	if resp.StatusCode != wantStatusCode {
		return zilch[Output](), NewError(resp)
	}

	var output Output

	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return zilch[Output](), err
	}

	return output, nil
}

func doJSONBody[Input any, Output any](ctx context.Context, c *Client, method, path string, input Input, wantStatusCode int) (Output, error) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(input); err != nil {
		return zilch[Output](), err
	}

	req, err := http.NewRequestWithContext(ctx, method, c.apiURL+path, buf)
	if err != nil {
		return zilch[Output](), err
	}

	resp, err := c.Do(req)
	if err != nil {
		return zilch[Output](), err
	}
	defer resp.Body.Close()

	if resp.StatusCode != wantStatusCode {
		return zilch[Output](), NewError(resp)
	}

	var output Output

	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return zilch[Output](), err
	}

	return output, nil
}
