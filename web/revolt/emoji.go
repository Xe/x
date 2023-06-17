package revolt

import (
	"context"
	"encoding/json"
)

// Emoji struct.
type Emoji struct {
	ID        string `json:"_id"`
	Name      string `json:"name"`
	Animated  bool   `json:"animated"`
	NSFW      bool   `json:"nsfw"`
	CreatorID string `json:"creator_id"`
	URL       string `json:"url"`
}

// Emoji grabs a single emoji by URL.
func (c *Client) Emoji(ctx context.Context, id string) (*Emoji, error) {
	emoji := &Emoji{}

	resp, err := c.Request(ctx, "GET", "/custom/emoji/"+id, []byte{})

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(resp, emoji)
	if err != nil {
		return nil, err
	}

	emoji.URL = c.Settings.Features.Autumn.URL + "/emojis/" + emoji.ID

	return emoji, err
}

type CreateEmoji struct {
	Name   string `json:"name"`
	NSFW   bool   `json:"nsfw"`
	Parent Parent `json:"parent"`
}

type Parent struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// CreateEmoji creates a new emoji.
func (c *Client) CreateEmoji(ctx context.Context, uploadID string, emoji CreateEmoji) (*Emoji, error) {
	data, err := json.Marshal(emoji)
	if err != nil {
		return nil, err
	}

	var emojiData Emoji

	resp, err := c.Request(ctx, "PUT", "/custom/emoji/"+uploadID, data)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(resp, &emojiData); err != nil {
		return nil, err
	}

	emojiData.URL = c.Settings.Features.Autumn.URL + "/emojis/" + emojiData.ID

	return &emojiData, nil
}

// DeleteEmoji deletes an emoji.
func (c *Client) DeleteEmoji(ctx context.Context, id string) error {
	_, err := c.Request(ctx, "DELETE", "/custom/emoji/"+id, []byte{})
	return err
}
