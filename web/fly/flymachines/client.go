package flymachines

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
)

const (
	internalURL = "http://_api.internal:4280"
	publicURL   = "https://api.machines.dev"
)

type Error struct {
	ErrorString string `json:"error"`
	StatusCode  int    `json:"status_code"`
	ReqID       string `json:"req_id"`
	Method      string `json:"method"`
	URL         string `json:"url"`
}

func NewError(resp *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("flymachines: can't read response body while creating error: %w", err)
	}

	result := &Error{
		StatusCode: resp.StatusCode,
		ReqID:      resp.Header.Get("Fly-Request-Id"),
		Method:     resp.Request.Method,
		URL:        resp.Request.URL.String(),
	}

	if err := json.Unmarshal(body, result); err != nil {
		result.ErrorString = string(body)
	}

	return result
}

func (e *Error) Error() string {
	return fmt.Sprintf("flymachines: %s %s (%d): %s", e.Method, e.URL, e.StatusCode, e.ErrorString)
}

func (e *Error) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("error", e.ErrorString),
		slog.Int("status_code", e.StatusCode),
		slog.String("req_id", e.ReqID),
		slog.String("method", e.Method),
		slog.String("url", e.URL),
	)
}

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
		slog.Debug("can't reach internal API, falling back to public API", "err", err)
		return NewClient(token, publicURL, cli)
	}

	if resp.StatusCode != http.StatusNotFound {
		slog.Debug("can't reach internal API, falling back to public API", "status_code", resp.StatusCode)
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
