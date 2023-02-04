package mastodon

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
)

type CreateStatusParams struct {
	Status      string     `json:"status"`
	InReplyTo   string     `json:"in_reply_to_id"`
	MediaIDs    []string   `json:"media_ids"`
	SpoilerText string     `json:"spoiler_text"`
	Visibility  string     `json:"visibility"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
}

func (csp CreateStatusParams) Values() url.Values {
	result := url.Values{}

	result.Set("status", csp.Status)

	if csp.Visibility != "" {
		result.Set("visibility", csp.Visibility)
	}

	if csp.InReplyTo != "" {
		result.Set("in_reply_to_id", csp.InReplyTo)
	}

	if csp.SpoilerText != "" {
		result.Set("spoiler_text", csp.SpoilerText)
	}

	for _, id := range csp.MediaIDs {
		result.Add("media_ids[]", id)
	}

	return result
}

func (c *Client) CreateStatus(ctx context.Context, csp CreateStatusParams) (*Status, error) {
	vals := csp.Values()

	u, err := c.server.Parse("/api/v1/statuses")
	if err != nil {
		return nil, err
	}

	resp, err := c.cli.PostForm(u.String(), vals)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var result Status
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// FetchStatus fetches a Mastodon status over the internet using the federation protocol.
//
// This will not work if the target server has "secure" mode enabled.
func FetchStatus(ctx context.Context, statusURL string) (*Status, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Status
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1024*1024*2)).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
