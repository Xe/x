//go:build ignore

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"within.website/x/attention"
	"within.website/x/internal"
)

var (
	discordToken  = flag.String("discord-token", "", "discord token")
	grpcAddr      = flag.String("grpc-addr", ":9001", "GRPC listen address")
	httpAddr      = flag.String("http-addr", ":9002", "HTTP listen address")
	openAIAPIBase = flag.String("openai-api-base", "", "OpenAI API base URL")
	openAIAPIKey  = flag.String("openai-api-key", "", "OpenAI API key")
	openAIModel   = flag.String("openai-model", "gpt-oss:120b", "OpenAI model")
)

func main() {
	internal.HandleStartup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bot, err := New(ctx)
	if err != nil {
		log.Fatal(err)
	}

	_ = bot

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	})

	go func() {
		slog.Info("listening", "kind", "http", "addr", *httpAddr)
		log.Fatal(http.ListenAndServe(*httpAddr, mux))
	}()

	<-ctx.Done()
}

type Bot struct {
	dg *discordgo.Session
	ai *openai.Client

	aiModel string

	lock          sync.Mutex
	conversations map[string]state
}

type state struct {
	conv []openai.Message
	aa   *attention.Attenuator
}

func New(ctx context.Context) (*Bot, error) {
	dg, err := discordgo.New("Bot " + *discordToken)
	if err != nil {
		return nil, fmt.Errorf("discord: error creating discord session: %w", err)
	}

	go func() {
		<-ctx.Done()
		dg.Close()
	}()

	go func() {
		if err := dg.Open(); err != nil {
			log.Fatal(err)
		}
	}()

	dg.StateEnabled = true

	ai := openai.NewClient(
		option.WithAPIKey(*openAIAPIKey),
		option.WithBaseURL(*openAIAPIBase),
	)

	return &Bot{
		dg:      dg,
		ai:      &ai,
		aiModel: *openAIModel,
	}, nil
}
