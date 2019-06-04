package main

import (
	"context"
	"flag"
	"net/http"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/mmikulicic/stringlist"
	"within.website/ln"
	"within.website/ln/opname"
	"within.website/x/internal"
	"within.website/x/vanity"
)

//go:generate go-bindata -pkg main static

var (
	domain         = flag.String("domain", "within.website", "domain this is run on")
	githubUsername = flag.String("github-user", "Xe", "GitHub username for GitHub repos")
	gogsDomain     = flag.String("gogs-url", "https://git.xeserv.us", "Gogs domain to use")
	gogsUsername   = flag.String("gogs-username", "xena", "Gogs username for above Gogs instance")
	port           = flag.String("port", "2134", "HTTP port to listen on")

	githubRepos = stringlist.Flag("github-repo", "list of GitHub repositories to use")
	gogsRepos   = stringlist.Flag("gogs-repo", "list of Gogs repositories to use")
)

var githubReposDefault = []string{
	"ln",
	"x",
	"xultybau",
	"johaus",
	"confyg",
	"derpigo",
}

var gogsReposDefault = []string{
	"gorqlite",
}

func main() {
	internal.HandleStartup()
	ctx := opname.With(context.Background(), "main")
	ctx = ln.WithF(ctx, ln.F{
		"domain": *domain,
	})

	if len(*githubRepos) == 0 {
		*githubRepos = githubReposDefault
	}

	if len(*gogsRepos) == 0 {
		*gogsRepos = gogsReposDefault
	}

	for _, repo := range *githubRepos {
		http.Handle("/"+repo, vanity.GitHubHandler(*domain+"/"+repo, *githubUsername, repo, "https"))
		http.Handle("/"+repo+"/", vanity.GitHubHandler(*domain+"/"+repo, *githubUsername, repo, "https"))

		ln.Log(ctx, ln.F{"github_repo": repo, "github_user": *githubUsername}, ln.Info("adding github repo"))
	}

	for _, repo := range *gogsRepos {
		http.Handle("/"+repo, vanity.GogsHandler(*domain+"/"+repo, *gogsDomain, *gogsUsername, repo, "https"))
		http.Handle("/"+repo+"/", vanity.GogsHandler(*domain+"/"+repo, *gogsDomain, *gogsUsername, repo, "https"))

		ln.Log(ctx, ln.F{"gogs_domain": *gogsDomain, "gogs_username": *gogsUsername, "gogs_repo": repo}, ln.Info("adding gogs repo"))
	}

	http.Handle("/static/", http.FileServer(
		&assetfs.AssetFS{
			Asset:    Asset,
			AssetDir: AssetDir,
		},
	))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte(indexTemplate))
	})

	http.HandleFunc("/.x.botinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")
		w.Write([]byte(botInfoPage))
	})

	ln.Log(ctx, ln.F{"port": *port}, ln.Info("Listening on HTTP"))
	http.ListenAndServe(":"+*port, nil)

}

const indexTemplate = `<!DOCTYPE html>
<html>
	<head>
		<title>within.website Go Packages</title>
		<link rel="stylesheet" href="/static/gruvbox.css">
		<meta name="viewport" content="width=device-width, initial-scale=1.0" />
	</head>
	<body id="top">
		<main>
			<h1><code>within.website</code> Go Packages</h1>

			<ul>
				<li><a href="https://within.website/confyg">confyg</a> - A generic configuration file parser based on the go modfile parser</li>
				<li><a href="https://within.website/derpigo">derpigo</a> - A simple wrapper to the <a href="https://derpibooru.org">Derpibooru</a> API</li>
				<li><a href="https://within.website/johaus">johaus</a> - <a href="http://lojban.org">Lojban</a> parsing</li>
				<li><a href="https://within.website/ln">ln</a> - Key->value based logging made context-aware and simple</li>
				<li><a href="https://within.website/x">x</a> - Experiments, toys and tinkering (many subpackages)</li>
			</ul>

			<hr />

			<footer class="is-text-center">
				<p>Need help with these packages? Inquire <a href="https://github.com/Xe">Within</a>.</p>
			</footer>
		</main>
	</body>
</html>`

const botInfoPage = `<link rel="stylesheet" href="/static/gruvbox.css">
<main>
<h1>x repo bots</h1>

Hello, if you are reading this, you have found this URL in your access logs.

If one of these programs is doing something you don't want them to do, please <a href="https://christine.website/contact">contact me</a> or open an issue <a href="https://github.com/Xe/x">here</a>.
</main>`
