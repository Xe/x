package tools

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"within.website/x/web/ollama"
)

type Impl interface {
	Execute(ctx context.Context, sess *discordgo.Session, mc *discordgo.MessageCreate, conv []ollama.Message, tc ollama.ToolCall) error
	Describe() ollama.Function
}
