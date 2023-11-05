package flymachines

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"

	"within.website/x/web"
)

const (
	internalURL = "http://_api.internal:4280"
	publicURL   = "https://api.machines.dev"
)

type Client struct {
	token, apiURL string
	cli           *http.Client
}

// NewClient returns a new client for the Fly machines API with the given API token and URL.
//
// This is a fairly low-level operation for if you know what URL you need, you probably want to use New instead.
func NewClient(token, apiURL string, cli *http.Client) *Client {
	return &Client{token, apiURL, cli}
}

// New returns a new client for the Fly machines API with the given API token.
//
// This will automatically detect if you have access to the internal API either by a
// WireGuard tunnel or by being on the Fly network.
func New(token string, cli *http.Client) *Client {
	resp, err := http.Get(internalURL)
	if err != nil {
		return NewClient(token, publicURL, cli)
	}

	if resp.StatusCode != http.StatusNotFound {
		return NewClient(token, publicURL, cli)
	}

	return NewClient(token, internalURL, cli)
}

// Do performs a HTTP request with the appropriate authentication and user agent headers.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", "within.website/x/web/fly/flymachines in "+os.Args[0])
	return c.cli.Do(req)
}

func (c *Client) doRequestNoResponse(ctx context.Context, method, path string, wantStatusCode int) error {
	req, err := http.NewRequestWithContext(ctx, method, c.apiURL+path, nil)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != wantStatusCode {
		return web.NewError(wantStatusCode, resp)
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
		return zilch[Output](), web.NewError(wantStatusCode, resp)
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
		return zilch[Output](), web.NewError(wantStatusCode, resp)
	}

	var output Output

	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return zilch[Output](), err
	}

	return output, nil
}
