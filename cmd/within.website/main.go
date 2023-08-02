// Command within.website is the vanity import server for https://within.website.
package main

import (
	"embed"
	"flag"
	"html/template"
	"net/http"
	"os"

	"go.jetpack.io/tyson"
	"golang.org/x/exp/slog"
	"tailscale.com/tsweb"
	"within.website/x/internal"
	"within.website/x/web/vanity"
)

var (
	domain      = flag.String("domain", "within.website", "domain this is run on")
	port        = flag.String("port", "2134", "HTTP port to listen on")
	tysonConfig = flag.String("tyson-config", "./config.ts", "TySON config file")

	//go:embed tmpl/*
	templateFiles embed.FS
)

type Repo struct {
	Kind        string `json:"kind"`
	Domain      string `json:"domain"`
	User        string `json:"user"`
	Repo        string `json:"repo"`
	Description string `json:"description"`
}

func (r Repo) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("kind", r.Kind),
		slog.String("domain", r.Domain),
		slog.String("user", r.User),
		slog.String("repo", r.Repo),
	)
}

func (r Repo) RegisterHandlers(lg *slog.Logger) {
	switch r.Kind {
	case "gitea":
		http.Handle("/"+r.Repo, vanity.GogsHandler(*domain+"/"+r.Repo, r.Domain, r.User, r.Repo, "https"))
		http.Handle("/"+r.Repo+"/", vanity.GogsHandler(*domain+"/"+r.Repo, r.Domain, r.User, r.Repo, "https"))
	case "github":
		http.Handle("/"+r.Repo, vanity.GitHubHandler(*domain+"/"+r.Repo, r.User, r.Repo, "https"))
		http.Handle("/"+r.Repo+"/", vanity.GitHubHandler(*domain+"/"+r.Repo, r.User, r.Repo, "https"))
	}
	lg.Debug("registered repo handler", "repo", r)
}

func main() {
	internal.HandleStartup()

	lg := slog.Default().With("domain", *domain, "configPath", *tysonConfig)

	tmpls := template.Must(template.ParseFS(templateFiles, "tmpl/*.tmpl"))

	var repos []Repo
	if err := tyson.Unmarshal(*tysonConfig, &repos); err != nil {
		lg.Error("can't unmarshal config", "err", err)
		os.Exit(1)
	}

	for _, repo := range repos {
		repo.RegisterHandlers(lg)
	}

	http.HandleFunc("/debug/varz", tsweb.VarzHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")
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
			Repos []Repo
		}{
			Title: "within.website Go packages",
			Repos: repos,
		})
	})

	http.HandleFunc("/.x.botinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")
		tmpls.ExecuteTemplate(w, "botinfo.tmpl", struct {
			Title string
		}{
			Title: "x repo bots",
		})
	})

	lg.Info("listening", "port", *port)
	http.ListenAndServe(":"+*port, http.DefaultServeMux)
}
