package main

import (
	"database/sql"
	"embed"
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"

	"golang.org/x/exp/slog"
	_ "modernc.org/sqlite"
	"tailscale.com/hostinfo"
	"tailscale.com/tsnet"
	"within.website/x/internal"
	"within.website/x/web/openai/chatgpt"
	"within.website/x/web/openai/moderation"
)

var (
	dbLoc       = flag.String("db-loc", "./data.db", "")
	openAIModel = flag.String("openai-model", "gpt-3.5-turbo", "OpenAI model to use")
	openAIToken = flag.String("openai-token", "", "OpenAI API token")
	slogLevel   = flag.String("slog-level", "INFO", "log level")
	tsHostname  = flag.String("ts-hostname", "apeirophobia", "hostname to use on the tailnet")
	tsDir       = flag.String("ts-dir", "", "directory to store Tailscale state")

	//go:embed schema.sql
	schema string

	//go:embed static
	staticFiles embed.FS

	//go:embed tmpl/*.tmpl
	templateFiles embed.FS
)

func main() {
	internal.HandleStartup()
	hostinfo.SetApp("within.website/x/cmd/apeirophobia")

	var programLevel slog.Level
	if err := (&programLevel).UnmarshalText([]byte(*slogLevel)); err != nil {
		fmt.Fprintf(os.Stderr, "invalid log level %s: %v, using info\n", *slogLevel, err)
		programLevel = slog.LevelInfo
	}

	h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel})
	slog.SetDefault(slog.New(h))

	slog.Debug("starting up", "hostname", *tsHostname)

	http.DefaultServeMux.Handle("/static/", http.FileServer(http.FS(staticFiles)))

	srv := &tsnet.Server{
		Hostname: *tsHostname,
		Dir:      *tsDir,
		Logf: func(format string, vals ...any) {
			slog.Debug(fmt.Sprintf(format, vals...), "group", "tsnet")
		},
	}

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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)
			tmpls.ExecuteTemplate(w, "404.tmpl", struct {
				Title string
			}{
				Title: "Not found: " + r.URL.Path,
			})

			return
		}
		tmpls.ExecuteTemplate(w, "index.tmpl", struct {
			Title string
		}{
			Title: "Apeirophobia",
		})
	})

	if err := srv.Start(); err != nil {
		slog.Error("error starting tsnet server", "err", err)
		os.Exit(1)
	}
	defer srv.Close()

	ln, err := srv.Listen("tcp", ":80")
	if err != nil {
		slog.Error("error listening over HTTP", "err", err)
		os.Exit(1)
	}
	defer ln.Close()

	slog.Info("listening", "hostname", *tsHostname)

	if err := http.Serve(ln, nil); err != nil {
		slog.Error("error running HTTP server", "err", err)
		os.Exit(1)
	}
}

type WikiHandler struct {
	db         *sql.DB
	tmpls      *template.Template
	srv        *tsnet.Server
	chatGPT    *chatgpt.Client
	moderation *moderation.Client
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

	log := slog.Default().WithGroup("wikiHandler").With("article", article, "remote_ip", r.RemoteAddr)

	prompt := userPrompt(article)
	modResp, err := wh.moderation.Check(r.Context(), prompt)
	if err != nil {
		log.Error("can't check moderation API", "err", err)
		wh.errorPage(err, w)
		return
	}

	if modResp.Flagged() {
		log.Error("filtered")
		w.WriteHeader(http.StatusBadRequest)
		wh.tmpls.ExecuteTemplate(w, "error.tmpl", struct {
			Title string
			Error string
		}{
			Title: "Filtered",
			Error: modResp.Reasons(),
		})
	}

}
