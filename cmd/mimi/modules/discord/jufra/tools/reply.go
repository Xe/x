package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"within.website/x/web/ollama"
)

type replyArgs struct {
	Message string `json:"message"`
}

func (ra replyArgs) Valid() error {
	if ra.Message == "" {
		return errors.New("tools: replyArgs is invalid: missing message")
	}

	return nil
}

type Reply struct{}

func (Reply) Execute(ctx context.Context, sess *discordgo.Session, mc *discordgo.MessageCreate, conv []ollama.Message, tc ollama.ToolCall) error {
	var args replyArgs

	if err := json.Unmarshal(tc.Arguments, &args); err != nil {
		return fmt.Errorf("error parsing reply args: %w", err)
	}

	if err := args.Valid(); err != nil {
		return err
	}

	if _, err := sess.ChannelMessageSend(mc.ChannelID, args.Message, discordgo.WithContext(ctx)); err != nil {
		return err
	}

	return nil
}

func (Reply) Describe() ollama.Function {
	return ollama.Function{
		Name:        "reply",
		Description: "Reply to the message",
		Parameters: ollama.Param{
			Type: "object",
			Properties: ollama.Properties{
				"message": {
					Type:        "string",
					Description: "The message to send",
				},
			},
			Required: []string{"message"},
		},
	}
}

var (
	_ Impl = Reply{}
)
