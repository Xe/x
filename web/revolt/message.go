package revolt

import (
	"encoding/json"
	"time"

	"github.com/oklog/ulid/v2"
)

// Message struct
type Message struct {
	Client    *Client
	CreatedAt time.Time

	Id          string          `json:"_id"`
	Nonce       string          `json:"nonce"`
	ChannelId   string          `json:"channel"`
	AuthorId    string          `json:"author"`
	Content     interface{}     `json:"content"`
	Edited      interface{}     `json:"edited"`
	Embeds      []*MessageEmbed `json:"embeds"`
	Attachments []*Attachment   `json:"attachments"`
	Mentions    []string        `json:"mentions"`
	Replies     []string        `json:"replies"`
}

// Attachment struct.
type Attachment struct {
	Id          string `json:"_id"`
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
	ulid, err := ulid.Parse(c.Id)

	if err != nil {
		return err
	}

	c.CreatedAt = time.UnixMilli(int64(ulid.Time()))
	return nil
}

// Edit message content.
func (m *Message) Edit(content string) error {
	_, err := m.Client.Request("PATCH", "/channels/"+m.ChannelId+"/messages/"+m.Id, []byte("{\"content\": \""+content+"\"}"))

	if err != nil {
		return err
	}

	m.Content = content
	return nil
}

// Delete the message.
func (m Message) Delete() error {
	_, err := m.Client.Request("DELETE", "/channels/"+m.ChannelId+"/messages/"+m.Id, []byte{})
	return err
}

// Reply to the message.
func (m Message) Reply(mention bool, sm *SendMessage) (*Message, error) {
	if sm.Nonce == "" {
		sm.CreateNonce()
	}

	sm.AddReply(m.Id, mention)

	respMessage := &Message{}
	respMessage.Client = m.Client
	msgData, err := json.Marshal(sm)

	if err != nil {
		return respMessage, err
	}

	resp, err := m.Client.Request("POST", "/channels/"+m.ChannelId+"/messages", msgData)

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
			respMessage.Delete()
		}()
	}

	return respMessage, nil
}
