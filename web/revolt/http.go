package revolt

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

// Send http request
func (c Client) Request(method, path string, data []byte) ([]byte, error) {
	reqBody := bytes.NewBuffer(data)

	// Prepare request
	req, err := http.NewRequest(method, API_URL+path, reqBody)
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
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return []byte{}, err
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return []byte{}, fmt.Errorf("%s: %s", resp.Status, body)
	}

	return body, nil
}
