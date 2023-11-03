package flymachines

import (
	"net/http"
	"os"
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
