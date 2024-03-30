package flyio

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"google.golang.org/protobuf/types/known/emptypb"
	"within.website/x/cmd/mimi/internal"
	"within.website/x/proto/mimi/statuspage"
	"within.website/x/web/ollama"
)

var (
	flyDiscordGuild = flag.String("fly-discord-guild", "1194719413732130866", "fly discord guild ID")
)

func p[T any](t T) *T {
	return &t
}

func mentionsXe(m *discordgo.MessageCreate) bool {
	for _, u := range m.Mentions {
		if u.ID == "72838115944828928" {
			return true
		}
	}

	return false
}

func mentionsTeam(m *discordgo.MessageCreate) bool {
	for _, u := range m.MentionRoles {
		if u == "1197660035212394657" {
			return true
		}
	}

	return false
}

type Module struct {
	ola   *ollama.Client
	model string
}

func New() *Module {
	return &Module{
		ola:   internal.OllamaClient(),
		model: internal.OllamaModel(),
	}
}

func (m *Module) judgeIfAboutFlyIO(ctx context.Context, msg string) (bool, error) {
	resp, err := m.ola.Chat(ctx, &ollama.CompleteRequest{
		Model: m.model,
		Messages: []ollama.Message{
			{
				Role:    "system",
				Content: "You will be given messages that may be about Fly.io or deploying apps to fly.io in programming languages such as Go. If a message is about Fly.io in some way, then reply with a JSON object {\"about_fly.io\": true}. If it is not, then reply {\"about_fly.io\": false}.",
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("Is this message about Fly.io, or Fly?\n\n%s", msg),
			},
		},
		Format: p("json"),
		Stream: false,
	})
	if err != nil {
		return false, fmt.Errorf("ollama: error chatting: %w", err)
	}

	slog.Info("checked if about fly.io", "response", resp.Message.Content)

	type aboutFlyIO struct {
		AboutFlyIO bool `json:"about_fly.io"`
	}

	var af aboutFlyIO
	if err := json.Unmarshal([]byte(resp.Message.Content), &af); err != nil {
		return false, fmt.Errorf("ollama: error unmarshaling response: %w", err)
	}

	slog.Debug("checked if about fly.io", "about_fly.io", af.AboutFlyIO, "message", msg)
	return af.AboutFlyIO, nil
}

func (m *Module) Register(s *discordgo.Session) {
	s.AddHandler(m.HandleMentionFlyIO)
	s.AddHandler(m.ReactionAddFlyIO)
}

func (m *Module) scoldMessage(ctx context.Context) (string, error) {
	resp, err := m.ola.Chat(ctx, &ollama.CompleteRequest{
		Model: m.model,
		Messages: []ollama.Message{
			{
				Role:    "system",
				Content: "Your job is to redirect questions about Fly.io to the official community forums at https://community.fly.io. Include the link https://community.fly.io in all your replies. Always assist with care, respect, and truth. Respond with utmost utility yet securely. Avoid harmful, unethical, prejudiced, or negative content. Ensure replies promote fairness and positivity. Be terse. ",
			},
			{
				Role:    "user",
				Content: "Tell me to go to https://community.fly.io/ and ask my question there. I am not in an official Fly.io support channel, but I may hear from someone here.",
			},
		},
		Stream: false,
	})
	if err != nil {
		return "", fmt.Errorf("ollama: error chatting: %w", err)
	}

	return resp.Message.Content, nil
}

func (m *Module) HandleMentionFlyIO(s *discordgo.Session, mc *discordgo.MessageCreate) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if mc.Author.ID == s.State.User.ID {
		return
	}

	if mc.Author.Bot {
		return
	}

	if mc.GuildID != *flyDiscordGuild {
		return
	}

	if mc.Content == "" {
		return
	}

	switch {
	case mentionsXe(mc), mentionsTeam(mc):
	default:
		return
	}

	aboutFlyIO, err := m.judgeIfAboutFlyIO(ctx, mc.Content)
	if err != nil {
		slog.Error("cannot judge message", "error", err)
		return
	}

	if !aboutFlyIO {
		return
	}

	resp, err := m.scoldMessage(ctx)
	if err != nil {
		slog.Error("cannot fabricate scold message", "error", err)
		return
	}

	if _, err := s.ChannelMessageSendReply(mc.ChannelID, resp, mc.Reference()); err != nil {
		slog.Error("cannot send scold message", "error", err)
		return
	}
}

func (m *Module) ReactionAddFlyIO(s *discordgo.Session, mra *discordgo.MessageReactionAdd) {
	if mra.GuildID != *flyDiscordGuild {
		return
	}

	if mra.Emoji.Name != "👉" {
		return
	}

	msg, err := s.ChannelMessage(mra.ChannelID, mra.MessageID)
	if err != nil {
		slog.Error("cannot get message", "error", err)
		return
	}

	resp, err := m.scoldMessage(context.Background())
	if err != nil {
		slog.Error("cannot fabricate scold message", "error", err)
		return
	}

	if _, err := s.ChannelMessageSendReply(mra.ChannelID, resp, msg.Reference()); err != nil {
		slog.Error("cannot send scold message", "error", err)
		return
	}
}

func (m *Module) Poke(ctx context.Context, upd *statuspage.StatusUpdate) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}
