package revolt

import (
	"context"
	"encoding/json"
	"time"

	"github.com/oklog/ulid/v2"
)

// Message struct
type Message struct {
	CreatedAt time.Time

	ID          string          `json:"_id"`
	Nonce       string          `json:"nonce"`
	ChannelId   string          `json:"channel"`
	AuthorId    string          `json:"author"`
	Content     string          `json:"content,omitempty"`
	Edited      interface{}     `json:"edited"`
	Embeds      []*MessageEmbed `json:"embeds"`
	Attachments []*Attachment   `json:"attachments"`
	Mentions    []string        `json:"mentions"`
	Replies     []string        `json:"replies"`
	Masquerade  *Masquerade     `json:"masquerade"`
}

type MessageAppend struct {
	Embeds []*MessageEmbed `json:"embeds"`
}

type Masquerade struct {
	Name      string `json:"name"`
	AvatarURL string `json:"avatar"`
	Color     string `json:"colour,omitempty"`
}

// Attachment struct.
type Attachment struct {
	ID          string `json:"_id"`
	Tag         string `json:"tag"`
	Size        int    `json:"size"`
	FileName    string `json:"filename"`
	Metadata    *AttachmentMetadata
	ContentType string `json:"content_type"`
}

// Attachment metadata struct.
type AttachmentMetadata struct {
	Type   string `json:"type"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Message edited struct.
type MessageEdited struct {
	Date int `json:"$date"`
}

// Message embed struct.
type MessageEmbed struct {
	Type        string `json:"type"`
	Url         string `json:"url"`
	Special     *MessageSpecialEmbed
	Title       string                `json:"title"`
	Description string                `json:"description"`
	Image       *MessageEmbeddedImage `json:"image"`
	Video       *MessageEmbeddedVideo `json:"video"`
	IconUrl     string                `json:"icon_url"`
	Color       string                `json:"color"`
}

// Message special embed struct.
type MessageSpecialEmbed struct {
	Type        string `json:"type"`
	Id          string `json:"id"`
	ContentType string `json:"content_type"`
}

// Message embedded image struct
type MessageEmbeddedImage struct {
	Size   string `json:"size"`
	Url    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Message embedded video struct
type MessageEmbeddedVideo struct {
	Url    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Calculate creation date and edit the struct.
func (c *Message) CalculateCreationDate() error {
	ulid, err := ulid.Parse(c.ID)

	if err != nil {
		return err
	}

	c.CreatedAt = time.UnixMilli(int64(ulid.Time()))
	return nil
}

// Edit message content.
func (c *Client) MessageEdit(ctx context.Context, channelID, messageID, content string) error {
	data, err := json.Marshal(map[string]string{
		"content": content,
	})
	if err != nil {
		return err
	}

	_, err = c.Request(ctx, "PATCH", "/channels/"+channelID+"/messages/"+messageID, data)
	if err != nil {
		return err
	}

	return nil
}

// Delete the message.
func (c *Client) MessageDelete(ctx context.Context, channelID, messageID string) error {
	_, err := c.Request(ctx, "DELETE", "/channels/"+channelID+"/messages/"+messageID, []byte{})
	return err
}

// Reply to the message.
func (c *Client) MessageReply(ctx context.Context, channelID, messageID string, mention bool, sm *SendMessage) (*Message, error) {
	if sm.Nonce == "" {
		sm.CreateNonce()
	}

	sm.AddReply(messageID, mention)

	respMessage := &Message{}
	msgData, err := json.Marshal(sm)

	if err != nil {
		return respMessage, err
	}

	resp, err := c.Request(ctx, "POST", "/channels/"+channelID+"/messages", msgData)

	if err != nil {
		return respMessage, err
	}

	err = json.Unmarshal(resp, respMessage)

	if err != nil {
		return respMessage, err
	}

	if sm.DeleteAfter != 0 {
		go func() {
			time.Sleep(time.Second * time.Duration(sm.DeleteAfter))
			c.MessageDelete(ctx, channelID, respMessage.ID)
		}()
	}

	return respMessage, nil
}
