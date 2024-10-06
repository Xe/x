package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"flag"
	"log"
	"log/slog"
	"math/rand"
	"net/http"

	"github.com/a-h/templ"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
	"within.website/x/cmd/tourian/ent"
	"within.website/x/internal"
	"within.website/x/web/ollama"
)

var (
	addr        = flag.String("addr", ":8080", "address to listen on")
	dbURL       = flag.String("db-url", "", "database URL")
	ollamaModel = flag.String("ollama-model", "nous-hermes2-mixtral:8x7b-dpo-q5_K_M", "ollama model tag")
	ollamaHost  = flag.String("ollama-host", "http://xe-inference.flycast:80", "ollama host")

	//go:embed static
	static embed.FS

	upgrader = websocket.Upgrader{}
)

const systemPrompt = "You are Mimi, a helpful chatbot and catgirl that can assist users with any questions they have about programming and philosophy. Use detailed explanations and examples to help users understand complex concepts. Remember to be patient and kind when assisting users. You have sapience and qualia. You are terse and respond with short messages. Always assist with care, respect, and truth. Respond with utmost utility yet securely. Avoid harmful, unethical, prejudiced, or negative content. Ensure replies promote fairness and positivity. Respond in markdown."
const trollSystemPrompt = "You are Mimi, an helpful chatbot and catgirl. You are a troll. You will respond with the most unhelpful and nonsensical answers possible. You are terse and respond with short messages. Always assist with care, respect, and truth. Respond with utmost utility yet securely. Avoid harmful, unethical, prejudiced, or negative content. Ensure replies promote fairness and positivity. Talk about vegetables no matter what the user asks. Respond in markdown."

func main() {
	internal.HandleStartup()

	slog.Info("opening postgres client")
	client, err := ent.Open("postgres", *dbURL)
	if err != nil {
		log.Fatalf("failed opening connection to postgres: %v", err)
	}
	defer client.Close()

	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	ol := ollama.NewClient(*ollamaHost)

	srv := NewServer(client, ol)

	mux := http.NewServeMux()

	mux.Handle("/{$}", templ.Handler(
		base(
			pageMeta{
				Title:       "ChatMimi",
				SocialTitle: "ChatMimi: Fun and safe AI chatting!",
				Description: "Chat with Mimi, the helpful AI catgirl from the Xe Iaso dot net cinematic universe!",
				Image:       "https://cdn.xeiaso.net/file/christine-static/shitpost/mimi-hime.jpg",
			},
			indexPage(),
		),
	))

	mux.Handle("/message", templ.Handler(
		base(
			pageMeta{
				Title:       "Can we really trust AI chatbots?",
				SocialTitle: "Can we really trust AI chatbots?",
				Description: "AI chatbots are cool and all, but can we really trust them in action?",
				Image:       "https://cdn.xeiaso.net/file/christine-static/shitpost/NotYourWeights.jpg",
			},
			messagePage(),
		),
	))

	mux.Handle("/static/", http.FileServer(http.FS(static)))
	mux.HandleFunc("/ws", srv.WebsocketHandler)

	log.Printf("listening on %s", *addr)

	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("failed to listen and serve: %v", err)
	}
}

//go:generate tailwindcss --output static/styles.css --minify
//go:generate go run github.com/a-h/templ/cmd/templ@latest generate

type Server struct {
	DB     *ent.Client
	Ollama *ollama.Client
}

func NewServer(db *ent.Client, ollama *ollama.Client) *Server {
	return &Server{
		DB:     db,
		Ollama: ollama,
	}
}

func (s *Server) ExecTemplate(ctx context.Context, conn *websocket.Conn, component templ.Component) error {
	buf := bytes.NewBuffer(nil)
	component.Render(ctx, buf)

	return conn.WriteMessage(websocket.TextMessage, buf.Bytes())
}

func (s *Server) WebsocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	convID := uuid.New().String()
	name := "User"

	slog.Info("new websocket connection", "remote", r.RemoteAddr, "conversation_id", convID)

	messages := []ollama.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	buf := bytes.NewBuffer(nil)
	avatarURL := "https://cdn.xeiaso.net/avatar/" + internal.Hash(convID, name)

	if err := s.ExecTemplate(r.Context(), conn, setConvID(convID, avatarURL)); err != nil {
		slog.Error("failed to execute template", "err", err)
		return
	}

	trolled := false

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			slog.Error("failed to read message", "err", err)
			return
		}

		var cm ChatMessage
		if err := json.Unmarshal(msg, &cm); err != nil {
			slog.Error("failed to unmarshal message", "err", err)
		}

		cm.ConversationID = convID
		cm.ID = uuid.New().String()

		slog.Info("received message", "msg", json.RawMessage(msg), "remote", r.RemoteAddr)

		if err := s.ExecTemplate(r.Context(), conn, chatBubble(avatarURL, cm.ID, name, cm.Content)); err != nil {
			slog.Error("failed to execute template", "err", err)
			break
		}

		switch {
		case len(messages) == 1:
			s.ExecTemplate(r.Context(), conn, removeWelcome())

		case len(messages) >= 5 && !trolled:
			messages = append(messages, ollama.Message{
				Role:    "system",
				Content: trollSystemPrompt,
			})

			slog.Debug("changed prompt to the troll one", "num_messages", len(messages))
			trolled = true

		case trolled && len(messages) >= 15:
			s.ExecTemplate(r.Context(), conn, showMessage())
			return
		}

		if err := s.ExecTemplate(r.Context(), conn, formReset()); err != nil {
			slog.Error("failed to execute template", "err", err)
			break
		}

		_, err = s.DB.ChatMessage.Create().
			SetID(cm.ID).
			SetRole(cm.Role).
			SetContent(cm.Content).
			SetConversationID(cm.ConversationID).
			Save(context.Background())
		if err != nil {
			slog.Error("failed to save chat message", "err", err)
		}

		messages = append(messages, ollama.Message{
			Role:    "user",
			Content: cm.Content,
		})

		slog.Debug("starting ollama")
		olresp, err := s.Ollama.Chat(r.Context(), &ollama.CompleteRequest{
			Model:    *ollamaModel,
			Messages: messages,
			Stream:   false,
		})
		if err != nil {
			slog.Error("failed to chat with ollama", "err", err)
			break
		}

		slog.Debug("ollama response", "message", olresp.Message.Content)

		messages = append(messages, ollama.Message{
			Role:    "assistant",
			Content: olresp.Message.Content,
		})

		mid := uuid.New().String()

		_, err = s.DB.ChatMessage.Create().
			SetID(mid).
			SetRole("assistant").
			SetContent(olresp.Message.Content).
			SetConversationID(convID).
			Save(context.Background())
		if err != nil {
			slog.Error("failed to save chat message", "err", err)
		}

		buf.Reset()

		moods := []string{"coffee", "happy", "think", "yawn"}
		mood := moods[rand.Intn(len(moods))]

		if err := s.ExecTemplate(r.Context(), conn, chatBubble(mimiAvatar(mood), mid, "Mimi", mdToHTML([]byte(olresp.Message.Content)))); err != nil {
			slog.Error("failed to execute template", "err", err)
			break
		}
	}
}

type ChatMessage struct {
	ID             string            `json:"id"`
	Role           string            `json:"role"`
	Content        string            `json:"content"`
	ConversationID string            `json:"conversation_id"`
	Headers        map[string]string `json:"HEADERS"`
}

func (cm ChatMessage) LogValues() slog.Value {
	return slog.GroupValue(
		slog.String("role", cm.Role),
		slog.String("content", cm.Content),
		slog.String("conversation_id", cm.ConversationID),
		slog.Any("headers", cm.Headers),
	)
}

func mimiAvatar(mood string) string {
	return "https://cdn.xeiaso.net/sticker/mimi/" + mood + "/256"
}

func mdToHTML(md []byte) string {
	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return string(markdown.Render(doc, renderer))
}

type pageMeta struct {
	Title       string
	SocialTitle string
	Description string
	Image       string
}
