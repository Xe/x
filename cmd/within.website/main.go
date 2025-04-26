// Command within.website is the vanity import server for https://within.website.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/a-h/templ"
	"go.jetpack.io/tyson"
	"within.website/x/internal"
	"within.website/x/web/vanity"
	"within.website/x/xess"
)

var (
	domain      = flag.String("domain", "within.website", "domain this is run on")
	port        = flag.String("port", "2134", "HTTP port to listen on")
	tysonConfig = flag.String("tyson-config", "./config.ts", "TySON config file")
)

type Repo struct {
	Kind        string `json:"kind"`
	Domain      string `json:"domain"`
	User        string `json:"user"`
	Repo        string `json:"repo"`
	Description string `json:"description"`
}

func (r Repo) URL() string {
	return fmt.Sprintf("https://%s/%s/%s", r.Domain, r.User, r.Repo)
}

func (r Repo) GodocURL() string {
	return fmt.Sprintf("https://pkg.go.dev/within.website/%s", r.Repo)
}

func (r Repo) GodocBadge() string {
	return fmt.Sprintf("https://pkg.go.dev/badge/within.website/%s.svg", r.Repo)
}

func (r Repo) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("kind", r.Kind),
		slog.String("domain", r.Domain),
		slog.String("user", r.User),
		slog.String("repo", r.Repo),
	)
}

func (r Repo) RegisterHandlers(mux *http.ServeMux, lg *slog.Logger) {
	switch r.Kind {
	case "gitea":
		mux.Handle("/"+r.Repo, vanity.GogsHandler(*domain+"/"+r.Repo, r.Domain, r.User, r.Repo, "https"))
		mux.Handle("/"+r.Repo+"/", vanity.GogsHandler(*domain+"/"+r.Repo, r.Domain, r.User, r.Repo, "https"))
	case "github":
		mux.Handle("/"+r.Repo, vanity.GitHubHandler(*domain+"/"+r.Repo, r.User, r.Repo, "https"))
		mux.Handle("/"+r.Repo+"/", vanity.GitHubHandler(*domain+"/"+r.Repo, r.User, r.Repo, "https"))
	}
	lg.Debug("registered repo handler", "repo", r)
}

//go:generate go tool templ generate

func main() {
	internal.HandleStartup()

	lg := slog.Default().With("domain", *domain, "configPath", *tysonConfig)

	var repos []Repo
	if err := tyson.Unmarshal(*tysonConfig, &repos); err != nil {
		lg.Error("can't unmarshal config", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	for _, repo := range repos {
		repo.RegisterHandlers(mux, lg)
	}

	xess.Mount(mux)

	mux.Handle("/{$}", templ.Handler(
		xess.Base(
			"within.website Go packages",
			nil,
			nil,
			Index(repos),
			footer(),
		),
	))

	mux.Handle("/", templ.Handler(
		xess.Simple("Not found", NotFound()),
		templ.WithStatus(http.StatusNotFound)),
	)

	mux.Handle("/.x.botinfo", templ.Handler(
		xess.Simple("x repo bots", BotInfo()),
	))

	lg.Info("listening", "port", *port)
	http.ListenAndServe(":"+*port, mux)
}
