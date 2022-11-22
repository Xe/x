package mastodon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"within.website/x/web"
	"within.website/x/web/useragent"
)

// Client is the client for Mastodon
type Client struct {
	cli    *http.Client
	server *url.URL
	token  string
}

// Unauthenticated makes a new unauthenticated Mastodon client.
func Unauthenticated(botName, botURL, instanceURL string) (*Client, error) {
	u, err := url.Parse(instanceURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		cli:    &http.Client{Transport: useragent.Transport(botName, botURL, http.DefaultTransport)},
		server: u,
	}, nil
}

// Authenticated makes a new authenticated Mastodon client.
func Authenticated(botName, botURL, instanceURL, token string) (*Client, error) {
	u, err := url.Parse(instanceURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		cli: &http.Client{
			Transport: useragent.Transport(botName, botURL, authTransport{token, http.DefaultTransport}),
		},
		server: u,
		token:  token,
	}, nil
}

type authTransport struct {
	bearerToken string
	next        http.RoundTripper
}

var (
	_ http.RoundTripper = &authTransport{}
)

func (at authTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", at.bearerToken))

	return at.next.RoundTrip(r)
}

func (c *Client) doJSONPost(ctx context.Context, path string, wantCode int, data any) (*http.Response, error) {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("Accept", "application/json")

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		return nil, err
	}

	return c.doRequest(ctx, http.MethodPost, "/api/v1/apps", h, http.StatusOK, &buf)
}

func (c *Client) doRequest(ctx context.Context, method, path string, headers http.Header, wantCode int, body io.Reader) (*http.Response, error) {
	u, err := c.server.Parse(path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("mastodon: can't make request: %w", err)
	}

	for key, hn := range headers {
		for _, hv := range hn {
			req.Header.Set(key, hv)
		}
	}

	resp, err := c.cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mastodon: HTTP response error: %w", err)
	}

	if resp.StatusCode != wantCode {
		return nil, web.NewError(wantCode, resp)
	}

	return resp, nil
}
