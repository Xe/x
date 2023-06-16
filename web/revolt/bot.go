package revolt

import (
	"context"
	"encoding/json"
	"time"

	"github.com/oklog/ulid/v2"
)

// Bot struct.
type Bot struct {
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

func (c *Client) BotEdit(ctx context.Context, id string, eb *EditBot) error {
	data, err := json.Marshal(eb)
	if err != nil {
		return err
	}

	if _, err := c.Request(ctx, "PATCH", "/bots/"+id, data); err != nil {
		return err
	}

	return nil
}

func (c *Client) BotDelete(ctx context.Context, id string) error {
	if _, err := c.Request(ctx, "DELETE", "/bots/"+id, []byte{}); err != nil {
		return err
	}

	return nil
}
