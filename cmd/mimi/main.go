package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"within.website/x/internal"
	"within.website/x/web/ollama"
)

var (
	dataDir         = flag.String("data-dir", "./var", "data directory for the bot")
	discordToken    = flag.String("discord-token", "", "discord token")
	flyDiscordGuild = flag.String("fly-discord-guild", "1194719413732130866", "fly discord guild ID")
	ollamaModel     = flag.String("ollama-model", "nous-hermes2-mixtral:8x7b-dpo-q5_K_M", "ollama model tag")
	ollamaHost      = flag.String("ollama-host", "http://xe-inference.flycast:80", "ollama host")
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

func main() {
	internal.HandleStartup()

	os.Setenv("OLLAMA_HOST", *ollamaHost)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = ctx

	dg, err := discordgo.New("Bot " + *discordToken)
	if err != nil {
		log.Fatal(err)
	}
	defer dg.Close()

	b := NewBot(dg, ollama.NewClient(*ollamaHost))

	dg.AddHandler(b.ReactionAddFlyIO)
	dg.AddHandler(b.HandleMentionFlyIO)

	if err := dg.Open(); err != nil {
		log.Fatal(err)
	}

	slog.Info("bot started")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	dg.Close()
	cancel()
}

type Bot struct {
	dg  *discordgo.Session
	ola *ollama.Client
}

func NewBot(dg *discordgo.Session, ola *ollama.Client) *Bot {
	return &Bot{
		dg:  dg,
		ola: ola,
	}
}

func (b *Bot) judgeIfAboutFlyIO(ctx context.Context, msg string) (bool, error) {
	resp, err := b.ola.Chat(ctx, &ollama.CompleteRequest{
		Model: *ollamaModel,
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

func (b *Bot) HandleMentionFlyIO(s *discordgo.Session, m *discordgo.MessageCreate) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Author.Bot {
		return
	}

	if m.GuildID != *flyDiscordGuild {
		return
	}

	if m.Content == "" {
		return
	}

	switch {
	case mentionsXe(m), mentionsTeam(m):
	default:
		return
	}

	aboutFlyIO, err := b.judgeIfAboutFlyIO(ctx, m.Content)
	if err != nil {
		slog.Error("cannot judge message", "error", err)
		return
	}

	if !aboutFlyIO {
		return
	}

	resp, err := b.scoldMessage(ctx)
	if err != nil {
		slog.Error("cannot fabricate scold message", "error", err)
		return
	}

	if _, err := s.ChannelMessageSendReply(m.ChannelID, resp, m.Reference()); err != nil {
		slog.Error("cannot send scold message", "error", err)
		return
	}
}

func (b *Bot) ReactionAddFlyIO(s *discordgo.Session, mra *discordgo.MessageReactionAdd) {
	if mra.GuildID != *flyDiscordGuild {
		return
	}

	if mra.Emoji.Name != "👉" {
		return
	}

	msg, err := b.dg.ChannelMessage(mra.ChannelID, mra.MessageID)
	if err != nil {
		slog.Error("cannot get message", "error", err)
		return
	}

	resp, err := b.scoldMessage(context.Background())
	if err != nil {
		slog.Error("cannot fabricate scold message", "error", err)
		return
	}

	if _, err := s.ChannelMessageSendReply(mra.ChannelID, resp, msg.Reference()); err != nil {
		slog.Error("cannot send scold message", "error", err)
		return
	}
}

func (b *Bot) scoldMessage(ctx context.Context) (string, error) {
	resp, err := b.ola.Chat(ctx, &ollama.CompleteRequest{
		Model: *ollamaModel,
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
