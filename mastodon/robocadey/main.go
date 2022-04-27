package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/McKael/madon/v2"
	"within.website/ln"
	"within.website/x/internal"
	"within.website/x/markov"
)

var (
	instance  = flag.String("instance", "", "mastodon instance")
	appID     = flag.String("app-id", "", "oauth2 app id")
	appSecret = flag.String("app-secret", "", "oauth2 app secret")
	token     = flag.String("token", "", "oauth2 token")
	state     = flag.String("state", "./robocadey.gob", "state file")
	readFrom  = flag.String("read-from", "", "if set, read from this JSON file")
)

var scopes = []string{"read", "write", "follow"}

func main() {
	internal.HandleStartup()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *readFrom != "" {
		os.Remove(*state)
		fin, err := os.Open(*readFrom)
		if err != nil {
			ln.FatalErr(ctx, err)
		}
		defer fin.Close()

		var lines []string
		c := markov.NewChain(3)
		err = json.NewDecoder(fin).Decode(&lines)
		if err != nil {
			ln.FatalErr(ctx, err)
		}

		for _, line := range lines {
			c.Write(line)
		}

		err = c.Save(*state)
		if err != nil {
			ln.FatalErr(ctx, err)
		}

		fmt.Println("data imported successfully")
		return
	}

	c, err := madon.RestoreApp("furry boost bot", *instance, *appID, *appSecret, &madon.UserToken{AccessToken: *token})
	if err != nil {
		ln.FatalErr(ctx, err, ln.Action("madon.RestoreApp"))
	}
	_ = c

	chain := markov.NewChain(3)
	err = chain.Load(*state)
	if err != nil {
		ln.FatalErr(ctx, err)
	}

	rand.Seed(time.Now().UnixMicro())

	if _, err := c.PostStatus(madon.PostStatusParams{
		Text: chain.Generate(150),
	}); err != nil {
		ln.FatalErr(ctx, err)
	}

	evChan := make(chan madon.StreamEvent, 10)
	stop := make(chan bool)
	done := make(chan bool)
	t := time.Tick(4 * time.Hour)

	err = c.StreamListener("user", "", evChan, stop, done)
	if err != nil {
		ln.FatalErr(ctx, err)
	}

	ln.Log(ctx, ln.F{
		"action": "streaming.toots",
	})

	for {
		select {
		case _, ok := <-done:
			if !ok {
				ln.Fatal(ctx, ln.F{"action": "stream.dead"})
			}

		case <-t:
			if _, err := c.PostStatus(madon.PostStatusParams{
				Text: chain.Generate(150),
			}); err != nil {
				ln.FatalErr(ctx, err)
			}

		case ev := <-evChan:
			switch ev.Event {
			case "error":
				ln.Fatal(ctx, ln.F{"err": ev.Error, "action": "processing.event"})
			case "notification":
				n := ev.Data.(madon.Notification)

				if n.Type == "mention" {
					time.Sleep(5 * time.Second)
					ln.Log(ctx, ln.F{
						"target": n.Account.Acct,
					})
					if _, err := c.PostStatus(madon.PostStatusParams{
						Text:      "@" + n.Account.Acct + " " + chain.Generate(150),
						InReplyTo: n.Status.ID,
					}); err != nil {
						ln.FatalErr(ctx, err)
					}
				}
			}
		}
	}
}
