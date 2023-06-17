package revolt

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"within.website/x/web"
)

func (c *Client) RequestWithPathAndContentType(ctx context.Context, method, path, contentType string, data []byte) ([]byte, error) {
	reqBody := bytes.NewBuffer(data)

	<-c.Ticker.C

	// Prepare request
	req, err := http.NewRequestWithContext(ctx, method, path, reqBody)
	if err != nil {
		return []byte{}, err
	}

	req.Header.Set("content-type", contentType)

	// Set auth headers
	if c.SelfBot == nil {
		req.Header.Set("x-bot-token", c.Token)
	} else if c.SelfBot.SessionToken != "" {
		req.Header.Set("x-session-token", c.SelfBot.SessionToken)
	}

	// Send request
	resp, err := c.HTTP.Do(req)

	if err != nil {
		return []byte{}, err
	}

	defer resp.Body.Close()

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return []byte{}, web.NewError(200, resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	return body, nil
}

// Send http request
func (c Client) Request(ctx context.Context, method, path string, data []byte) ([]byte, error) {
	reqBody := bytes.NewBuffer(data)

	<-c.Ticker.C

	// Prepare request
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reqBody)
	if err != nil {
		return []byte{}, err
	}

	req.Header.Set("content-type", "application/json")

	// Set auth headers
	if c.SelfBot == nil {
		req.Header.Set("x-bot-token", c.Token)
	} else if c.SelfBot.SessionToken != "" {
		req.Header.Set("x-session-token", c.SelfBot.SessionToken)
	}

	// Send request
	resp, err := c.HTTP.Do(req)

	if err != nil {
		return []byte{}, err
	}

	defer resp.Body.Close()

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return []byte{}, web.NewError(200, resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	return body, nil
}
