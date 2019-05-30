package main

import (
	"context"
	"flag"
	"net/http"

	"github.com/Xe/x/internal"
	"github.com/Xe/x/vanity"
	"github.com/mmikulicic/stringlist"
	"within.website/ln"
	"within.website/ln/opname"
)

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

	ln.Log(ctx, ln.F{"port": *port}, ln.Info("Listening on HTTP"))
	http.ListenAndServe(":"+*port, nil)
}
