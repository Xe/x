// Command priorworkgen generates the list of prior work (read: GitHub repositories) for contractual stuff.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/google/go-github/github"
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
	"within.website/x/internal"
)

var (
	ghToken = flag.String("gh-token", "", "github personal access token")
)

func main() {
	internal.HandleStartup()
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *ghToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	var repos []*github.Repository
	var np int

	for {
		// list all repositories for the authenticated user

		options := &github.RepositoryListOptions{
			ListOptions: github.ListOptions{Page: np},
			Type:        "owner",
		}
		mrepos, resp, err := client.Repositories.List(ctx, "", options)
		if err != nil {
			log.Printf("can't get next page: %v", err)
			break
		}
		np = resp.NextPage

		repos = append(repos, mrepos...)

		log.Printf("got info on %d repos", len(repos))

		if len(repos) > 150 {
			break
		}
	}

	for _, repo := range repos {
		if repo.GetFork() {
			continue
		}
		if repo.GetPrivate() {
			continue
		}

		if repo.Description == nil {
			desc := ""
			repo.Description = &desc
		}

		name := repo.GetName()
		desc := repo.GetDescription()
		refn := repo.GetGitURL()
		creat := repo.GetCreatedAt()
		lastm := repo.GetUpdatedAt()

		if name == "ircbot" {
			continue
		}

		const blurb = `name: ${name}
description: ${desc}
reference number: ${refn}
date of creation: ${creat}
date of last modification: ${lastm}
other owners: none
`

		mapping := func(inp string) string {
			switch inp {
			case "name":
				return name
			case "desc":
				if desc == "" {
					return "no description available"
				}

				return desc
			case "refn":
				return refn
			case "creat":
				return creat.String()
			case "lastm":
				return lastm.String()
			}

			return "<unknown input " + inp + ">"
		}

		fmt.Println(os.Expand(blurb, mapping))
	}
}
