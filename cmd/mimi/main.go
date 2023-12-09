package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"within.website/x/cmd/mimi/ollama"
	"within.website/x/internal"
	"within.website/x/llm"
	"within.website/x/llm/llamaguard"
	"within.website/x/llm/llava"
)

var (
	dataDir        = flag.String("data-dir", "./var", "data directory for the bot")
	discordToken   = flag.String("discord-token", "", "discord token")
	discordGuild   = flag.String("discord-guild", "192289762302754817", "discord guild")
	discordChannel = flag.String("discord-channel", "217096701771513856", "discord channel")
	llamaguardHost = flag.String("llamaguard-host", "http://ontos:11434", "llamaguard host")
	llavaHost      = flag.String("llava-host", "http://localhost:8080", "llava host")
	ollamaModel    = flag.String("ollama-model", "xe/mimi:f16", "ollama model tag")
	ollamaHost     = flag.String("ollama-host", "http://kaine:11434", "ollama host")
	openAIKey      = flag.String("openai-api-key", "", "openai key")
	openAITTSModel = flag.String("openai-tts-model", "nova", "openai tts model")
)

func p[T any](t T) *T {
	return &t
}

func main() {
	internal.HandleStartup()

	os.Setenv("OLLAMA_HOST", *ollamaHost)

	cli, err := ollama.ClientFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cli.Pull(ctx,
		&ollama.PullRequest{
			Name:   *ollamaModel,
			Stream: p(true),
		},
		func(pr ollama.ProgressResponse) error {
			slog.Debug("pull progress", "progress", pr.Total-pr.Completed, "total", pr.Total)
			return nil
		},
	); err != nil {
		log.Fatal(err)
	}

	dg, err := discordgo.New("Bot " + *discordToken)
	if err != nil {
		log.Fatal(err)
	}
	defer dg.Close()

	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

		if m.GuildID != *discordGuild {
			return
		}

		if m.ChannelID != *discordChannel {
			return
		}

		if m.Author.Bot {
			return
		}

		if m.Content == "!mimi" {
			s.ChannelMessageSend(m.ChannelID, "mimi!")
			return
		}

		if m.Content == "!mimi clear" {
			lock.Lock()
			delete(stateMap, m.ChannelID)
			lock.Unlock()
			s.ChannelMessageSend(m.ChannelID, "mimi state cleared")
			return
		}

		var sb strings.Builder
		var prompt strings.Builder

		if ns, ok := ParseNameslash(m.Content); ok {
			if err := json.NewEncoder(&prompt).Encode(map[string]any{
				"message":  ns.Message,
				"user":     ns.Name,
				"is_admin": m.Author.Username == "xeiaso",
			}); err != nil {
				slog.Error("json encode error", "error", err)
			}
		} else {
			if err := json.NewEncoder(&prompt).Encode(map[string]any{
				"message":  m.Content,
				"user":     m.Author.Username,
				"is_admin": m.Author.Username == "xeiaso",
			}); err != nil {
				slog.Error("json encode error", "error", err)
			}
		}

		if len(m.Attachments) > 0 {
			for i, a := range m.Attachments {
				switch a.ContentType {
				case "image/png", "image/jpeg", "image/gif":
				default:
					continue
				}

				resp, err := http.Get(a.URL)
				if err != nil {
					slog.Error("http get error", "error", err)
					continue
				}
				defer resp.Body.Close()

				lrq, err := llava.DefaultRequest(m.Content, resp.Body)
				if err != nil {
					slog.Error("llava error", "error", err)
					continue
				}

				lresp, err := llava.Describe(context.Background(), *llavaHost+"/completion", lrq)
				if err != nil {
					slog.Error("llava error", "error", err)
					continue
				}

				if err := json.NewEncoder(&prompt).Encode(map[string]any{
					"image": i,
					"desc":  lresp.Content,
				}); err != nil {
					slog.Error("json encode error", "error", err)
					continue
				}
			}
		}

		lock.Lock()
		defer lock.Unlock()

		st, ok := stateMap[m.ChannelID]
		if !ok {
			st = &State{}
			/*				Messages: []llm.Message{{
								Role:    "user",
								Content: prompt.String(),
							}},
						}*/

			stateMap[m.ChannelID] = st
		}

		gr, err := llamaguard.Check(*llamaguardHost, st.Messages)
		if err != nil {
			slog.Error("llamaguard error", "error", err)
		}

		if !gr.Safe {
			prompt.Reset()
			prompt.WriteString("Please write a detailed message explaining that the request violates rule ")
			for _, c := range gr.Categories {
				prompt.WriteString(c)
				prompt.WriteString(": ")
				prompt.WriteString(llamaguard.Rules[c])
			}
			prompt.WriteString(".\n\nAlso explain that the conversation will be reset.")
			defer delete(stateMap, m.ChannelID)
		}

		st.Messages = append(st.Messages, llm.Message{
			Role:    "user",
			Content: prompt.String(),
		})

		err = cli.Generate(ctx,
			&ollama.GenerateRequest{
				Model:   *ollamaModel,
				Context: st.Context,
				Prompt:  prompt.String(),
				Stream:  p(true),
				System:  "Your name is Mimi, a helpful catgirl assistant.",
			}, func(gr ollama.GenerateResponse) error {
				fmt.Fprint(&sb, gr.Response)

				if gr.Done {
					st.Context = gr.Context
					st.Messages = append(st.Messages, llm.Message{
						Role:    "assistant",
						Content: gr.Response,
					})

					slog.Info("generated message", "dur", gr.EvalDuration.String(), "tokens/sec", float64(gr.EvalCount)/gr.EvalDuration.Seconds())
				}
				return nil
			},
		)
		if err != nil {
			slog.Error("generate error", "error", err, "channel", m.ChannelID)
			return
		}

		gr, err = llamaguard.Check(*llamaguardHost, st.Messages)
		if err != nil {
			slog.Error("llamaguard error", "error", err)
			s.ChannelMessageSend(m.ChannelID, "llamaguard error")
			return
		}

		if !gr.Safe {
			sb.Reset()
			err = cli.Generate(ctx,
				&ollama.GenerateRequest{
					Model:   *ollamaModel,
					Context: st.Context,
					Prompt:  "Say that you're sorry and you can't help with that. The conversation will be reset.",
					Stream:  p(true),
					System:  "Your name is Mimi, a helpful catgirl assistant.",
				}, func(gr ollama.GenerateResponse) error {
					fmt.Fprint(&sb, gr.Response)

					if gr.Done {
						st.Context = gr.Context
						st.Messages = append(st.Messages, llm.Message{
							Role:    "assistant",
							Content: gr.Response,
						})

						slog.Info("generated message", "dur", gr.EvalDuration.String(), "tokens/sec", float64(gr.EvalCount)/gr.EvalDuration.Seconds())
					}
					return nil
				},
			)
			if err != nil {
				slog.Error("generate error", "error", err, "channel", m.ChannelID)
				return
			}

			s.ChannelMessageSend(m.ChannelID, "🔀"+sb.String())
			defer delete(stateMap, m.ChannelID)

			return
		}

		if _, err := s.ChannelMessageSend(m.ChannelID, sb.String()); err != nil {
			slog.Error("message send error", "err", err, "message", sb.String())
		}
		slog.Debug("context length", "len", len(st.Context))
	})

	if err := dg.Open(); err != nil {
		log.Fatal(err)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	cancel()
}

var lock sync.Mutex
var stateMap = map[string]*State{}

type State struct {
	Context  []int
	Messages []llm.Message
}

type Nameslash struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

func ParseNameslash(msg string) (Nameslash, bool) {
	parts := strings.Split(msg, "\\")
	if len(parts) != 2 {
		return Nameslash{}, false
	}
	return Nameslash{parts[0], parts[1]}, true
}
