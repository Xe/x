package revolt

import (
	"encoding/json"
	"time"

	"github.com/oklog/ulid/v2"
)

// Bot struct.
type Bot struct {
	Client    *Client
	CreatedAt time.Time

	Id              string `json:"_id"`
	OwnerId         string `json:"owner"`
	Token           string `json:"token"`
	IsPublic        bool   `json:"public"`
	InteractionsUrl string `json:"interactionsURL"`
}

// Fetched bots struct.
type FetchedBots struct {
	Bots  []*Bot  `json:"bots"`
	Users []*User `json:"users"`
}

// Calculate creation date and edit the struct.
func (b *Bot) CalculateCreationDate() error {
	ulid, err := ulid.Parse(b.Id)

	if err != nil {
		return err
	}

	b.CreatedAt = time.UnixMilli(int64(ulid.Time()))
	return nil
}

// Edit the bot.
func (b *Bot) Edit(eb *EditBot) error {
	data, err := json.Marshal(eb)

	if err != nil {
		return err
	}

	_, err = b.Client.Request("PATCH", "/bots/"+b.Id, data)
	return err
}

// Delete the bot.
func (b *Bot) Delete() error {
	_, err := b.Client.Request("DELETE", "/bots/"+b.Id, []byte{})
	return err
}
