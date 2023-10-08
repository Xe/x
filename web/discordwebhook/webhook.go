// Package discordwebhook is a simple low-level HTTP client wrapper around Discord webhooks.
package discordwebhook

import (
	"bytes"
	"encoding/json"
	"net/http"

	"within.website/x/web"
)

// Webhook is the parent structure fired off to Discord.
type Webhook struct {
	Content         string              `json:"content,omitempty"`
	Username        string              `json:"username,omitempty"`
	AvatarURL       string              `json:"avatar_url"`
	Embeds          []Embeds            `json:"embeds,omitempty"`
	AllowedMentions map[string][]string `json:"allowed_mentions"`
}

// EmbedField is an individual field being embedded in a message.
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

// EmbedFooter is the message footer.
type EmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url"`
}

// Embeds is a set of embed fields and a footer.
type Embeds struct {
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	URL         string       `json:"url,omitempty"`
	Author      *EmbedAuthor `json:"author,omitempty"`
	Fields      []EmbedField `json:"fields,omitempty"`
	Footer      *EmbedFooter `json:"footer,omitempty"`
}

type EmbedAuthor struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	IconURL string `json:"icon_url"`
}

// Send returns a request for a Discord webhook.
func Send(whurl string, w Webhook) *http.Request {
	if len(w.Username) > 32 {
		w.Username = w.Username[:32]
	}

	data, err := json.Marshal(&w)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest(http.MethodPost, whurl, bytes.NewBuffer(data))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", "application/json")

	return req
}

// Validate validates the response from Discord.
func Validate(resp *http.Response) error {
	if resp.StatusCode != http.StatusNoContent {
		return web.NewError(http.StatusNoContent, resp)
	}

	return nil
}
