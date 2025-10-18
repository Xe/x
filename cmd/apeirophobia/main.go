package main

import (
	"database/sql"
	"embed"
	_ "embed"
	"flag"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"strings"

	_ "modernc.org/sqlite"
	"within.website/x/internal"
	"within.website/x/web/openai/chatgpt"
)

var (
	bind          = flag.String("bind", ":9094", "host:port bind addr")
	dbLoc         = flag.String("db-loc", "./data.db", "SQLite database file")
	openAIBaseURL = flag.String("openai-base-url", "", "If set, base OpenAI API URL")
	openAIModel   = flag.String("openai-model", "gpt-3.5-turbo", "OpenAI model to use")
	openAIToken   = flag.String("openai-token", "", "OpenAI API token")

	//go:embed schema.sql
	schema string

	//go:embed static
	staticFiles embed.FS

	//go:embed tmpl/*.tmpl
	templateFiles embed.FS
)

func main() {
	internal.HandleStartup()

	slog.Debug("starting up", "model", *openAIModel)

	http.Handle("/static/", http.FileServer(http.FS(staticFiles)))

	tmpls := template.Must(template.ParseFS(templateFiles, "tmpl/*.tmpl"))

	db, err := sql.Open("sqlite", *dbLoc)
	if err != nil {
		slog.Error("error opening database", "err", err, "dbLoc", *dbLoc)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("error testing database", "err", err, "dbLoc", *dbLoc)
		os.Exit(1)
	}

	if _, err := db.Exec(schema); err != nil {
		slog.Error("error loading database schema", "err", err, "dbLoc", *dbLoc)
		os.Exit(1)
	}

	http.HandleFunc("/{$}", func(w http.ResponseWriter, r *http.Request) {
		tmpls.ExecuteTemplate(w, "index.tmpl", struct {
			Title string
		}{
			Title: "Apeirophobia",
		})
	})

	if err := http.ListenAndServe(*bind, nil); err != nil {
		slog.Error("error running HTTP server", "err", err)
		os.Exit(1)
	}
}

type WikiHandler struct {
	db      *sql.DB
	tmpls   *template.Template
	chatGPT *chatgpt.Client
}

func (wh WikiHandler) errorPage(err error, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	wh.tmpls.ExecuteTemplate(w, "error.tmpl", struct {
		Title string
		Error string
	}{
		Title: "Internal Server Error",
		Error: err.Error(),
	})
}

func (wh WikiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	article, found := strings.CutPrefix(r.URL.Path, "/wiki/")
	if !found {
		w.WriteHeader(http.StatusNotFound)
		wh.tmpls.ExecuteTemplate(w, "404.tmpl", struct {
			Title string
		}{
			Title: "Not found: " + r.URL.Path,
		})

		return
	}

	lg := slog.Default().WithGroup("wikiHandler").With("article", article, "remote_ip", r.RemoteAddr)

	prompt := userPrompt(article)

	_ = lg
	_ = prompt
}
