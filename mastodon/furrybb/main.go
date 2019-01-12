package main

import (
	"context"
	"flag"

	"github.com/McKael/madon"
	"github.com/Xe/x/internal"
	_ "github.com/joho/godotenv/autoload"
	"within.website/ln"
)

var (
	instance  = flag.String("instance", "", "mastodon instance")
	appID     = flag.String("app-id", "", "oauth2 app id")
	appSecret = flag.String("app-secret", "", "oauth2 app secret")
	token     = flag.String("token", "", "oauth2 token")
	hashtag   = flag.String("hashtag", "furry", "hashtag to monitor")
)

var scopes = []string{"read", "write", "follow"}
var ctx = context.Background()

func main() {
	internal.HandleStartup()

	c, err := madon.RestoreApp("furry boost bot", *instance, *appID, *appSecret, &madon.UserToken{AccessToken: *token})
	if err != nil {
		ln.Fatal(ctx, ln.F{"err": err, "action": "madon.RestoreApp"})
	}

	evChan := make(chan madon.StreamEvent, 10)
	stop := make(chan bool)
	done := make(chan bool)

	err = c.StreamListener("public", "", evChan, stop, done)
	if err != nil {
		ln.Fatal(ctx, ln.F{"err": err, "action": "c.StreamListener"})
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

		case ev := <-evChan:
			switch ev.Event {
			case "error":
				ln.Fatal(ctx, ln.F{"err": ev.Error, "action": "processing.event"})
			case "update":
				s := ev.Data.(madon.Status)

				for _, tag := range s.Tags {
					if tag.Name == cfg.Hashtag {
						err = c.ReblogStatus(s.ID)
						if err != nil {
							ln.Fatal(ctx, ln.F{"err": err, "action": "c.ReblogStatus", "id": s.ID})
						}

						ln.Log(ctx, ln.F{
							"action": "reblogged",
							"id":     s.ID,
						})
					}
				}
			}
		}
	}
}
