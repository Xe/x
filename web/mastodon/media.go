package mastodon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"within.website/x/web"
)

func (c *Client) UploadMedia(ctx context.Context, fin io.Reader, fname, description, focus string) (*Attachment, error) {
	var buf bytes.Buffer

	w := multipart.NewWriter(&buf)
	fout, err := w.CreateFormFile("file", fname)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(fout, fin); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("can't prepare form: %w", err)
	}

	u, err := c.server.Parse("/api/v1/media")
	if err != nil {
		return nil, err
	}

	q := u.Query()

	if description != "" {
		q.Set("description", description)
	}
	if focus != "" {
		q.Set("focus", focus)
	}

	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), &buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("can't post to %s: %w", u, err)
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
	// This is okay
	default:
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result Attachment

	if err := json.NewDecoder(io.LimitReader(resp.Body, 1024*1024*2)).Decode(&result); err != nil {
		return nil, fmt.Errorf("can't decode JSON: %w", err)
	}

	return &result, nil
}
