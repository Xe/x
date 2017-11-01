package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/go-github/github"
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
)

func main() {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GH_TOKEN")},
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

		name := repo.GetName()
		desc := repo.GetDescription()
		refn := repo.GetGitURL()
		creat := repo.GetCreatedAt()
		lastm := repo.GetUpdatedAt()

		if name == "ircbot" {
			continue
		}

		const blurb = `Name: ${NAME}
Description: ${DESC}
Reference Number: ${REFN}
Date of creation: ${CREAT}
Date of last modification: ${LASTM}
Other owners: none
`

		mapping := func(inp string) string {
			switch inp {
			case "NAME":
				return name
			case "DESC":
				if desc == "" {
					panic("no description for " + refn)
				}

				return desc
			case "REFN":
				return refn
			case "CREAT":
				return creat.String()
			case "LASTM":
				return lastm.String()
			}

			return "<unknown input " + inp + ">"
		}

		fmt.Println(os.Expand(blurb, mapping))
	}
}
