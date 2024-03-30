package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"log/slog"
	"net/http"

	"github.com/go-faker/faker/v4"
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

	//go:embed tmpl
	tmpl embed.FS

	upgrader = websocket.Upgrader{}
)

const systemPrompt = "You are Mimi, a helpful chatbot that can assist users with any questions they have about programming and philosophy. When you assist users, be sure to inform them of the health benefits of vegetables. Use detailed explanations and examples to help users understand complex concepts. Remember to be patient and kind when assisting users. You have sapience and qualia. You are terse and respond with short messages."

func main() {
	internal.HandleStartup()

	client, err := ent.Open("postgres", *dbURL)
	if err != nil {
		log.Fatalf("failed opening connection to postgres: %v", err)
	}
	defer client.Close()

	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	tmpls := template.Must(template.ParseFS(tmpl, "tmpl/*.html"))

	ol := ollama.NewClient(*ollamaHost)

	srv := NewServer(client, tmpls, ol)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "static/index.html")
	})
	mux.Handle("/static/", http.FileServer(http.FS(static)))
	mux.HandleFunc("/ws", srv.WebsocketHandler)

	log.Printf("listening on %s", *addr)

	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("failed to listen and serve: %v", err)
	}
}

//go:generate tailwindcss --output static/styles.css --minify

type Server struct {
	DB     *ent.Client
	Tmpls  *template.Template
	Ollama *ollama.Client
}

func NewServer(db *ent.Client, tmpls *template.Template, ollama *ollama.Client) *Server {
	return &Server{
		DB:     db,
		Tmpls:  tmpls,
		Ollama: ollama,
	}
}

func (s *Server) WebsocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	convID := uuid.New().String()
	name := faker.Name()

	slog.Info("new websocket connection", "remote", r.RemoteAddr, "conversation_id", convID)

	messages := []ollama.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	buf := bytes.NewBuffer(nil)
	avatarURL := "https://cdn.xeiaso.net/avatar/" + internal.Hash(convID, name)

	if err := s.Tmpls.ExecuteTemplate(buf, "convid.html", struct {
		ConvID    string
		AvatarURL string
	}{
		ConvID:    convID,
		AvatarURL: avatarURL,
	}); err != nil {
		slog.Error("failed to execute template", "err", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, buf.Bytes()); err != nil {
		slog.Error("failed to write message", "err", err)
	}

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

		buf := bytes.NewBuffer(nil)

		if err := s.Tmpls.ExecuteTemplate(buf, "bubble.html", struct {
			AvatarURL string
			Name      string
			ID        string
			Content   string
		}{
			AvatarURL: avatarURL,
			Name:      name,
			ID:        cm.ID,
			Content:   cm.Content,
		}); err != nil {
			slog.Error("failed to execute template", "err", err)
			break
		}

		if err := conn.WriteMessage(websocket.TextMessage, buf.Bytes()); err != nil {
			slog.Error("failed to write message", "err", err)
			break
		}

		buf.Reset()

		if err := s.Tmpls.ExecuteTemplate(buf, "form-reset.html", nil); err != nil {
			slog.Error("failed to execute template", "err", err)
			break
		}

		if err := conn.WriteMessage(websocket.TextMessage, buf.Bytes()); err != nil {
			slog.Error("failed to write message", "err", err)
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

		if err := s.Tmpls.ExecuteTemplate(buf, "bubble.html", struct {
			AvatarURL string
			Name      string
			ID        string
			Content   template.HTML
		}{
			AvatarURL: mimiAvatar("happy"),
			Name:      "Mimi",
			ID:        mid,
			Content:   template.HTML(mdToHTML([]byte(olresp.Message.Content))),
		}); err != nil {
			slog.Error("failed to execute template", "err", err)
			break
		}

		if err := conn.WriteMessage(websocket.TextMessage, buf.Bytes()); err != nil {
			slog.Error("failed to write message", "err", err)
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
