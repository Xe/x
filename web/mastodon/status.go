package mastodon

import (
	"context"
	"encoding/json"
	"fmt"
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

	for i, id := range csp.MediaIDs {
		qID := fmt.Sprintf("[%d]media_ids", i)
		result.Set(qID, id)
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
