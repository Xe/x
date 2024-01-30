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

func main() {
	internal.HandleStartup()

	os.Setenv("OLLAMA_HOST", *ollamaHost)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dg, err := discordgo.New("Bot " + *discordToken)
	if err != nil {
		log.Fatal(err)
	}
	defer dg.Close()

	b := NewBot(dg, ollama.NewClient(*ollamaHost))

	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
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

		if len(m.Mentions) == 0 {
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

		resp, err := b.scoldMessage(ctx, m.Content)
		if err != nil {
			slog.Error("cannot fabricate scold message", "error", err)
			return
		}

		if _, err := s.ChannelMessageSendReply(m.ChannelID, resp, m.Reference()); err != nil {
			slog.Error("cannot send scold message", "error", err)
			return
		}
	})

	if err := dg.Open(); err != nil {
		log.Fatal(err)
	}

	slog.Info("bot started")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
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
				Content: "You will be given messages that may be about Fly.io or deploying apps to fly.io in programming lanugages such as Go. If a message is about Fly.io in some way, then reply with a JSON object {\"about_fly.io\": true}. If it is not, then reply {\"about_fly.io\": false}.",
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("Is this message about Fly.io?\n\n%s", msg),
			},
		},
		Format: p("json"),
		Stream: false,
	})
	if err != nil {
		return false, fmt.Errorf("ollama: error chatting: %w", err)
	}

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

func (b *Bot) scoldMessage(ctx context.Context, content string) (string, error) {
	resp, err := b.ola.Chat(ctx, &ollama.CompleteRequest{
		Model: *ollamaModel,
		Messages: []ollama.Message{
			{
				Role:    "system",
				Content: "Your job is to redirect questions about Fly.io to the community forums at https://community.fly.io. Don't include the link in your response, just tell the user to go there. Rephrase the question.",
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("Please redirect this question to the community forums:\n\n%s", content),
			},
		},
		Stream: false,
	})
	if err != nil {
		return "", fmt.Errorf("ollama: error chatting: %w", err)
	}

	return resp.Message.Content, nil
}
