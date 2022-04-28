package main

import (
	"context"
	"encoding/json"
	"flag"
	"math/rand"
	"net"
	"time"

	"github.com/McKael/madon/v2"
	"within.website/ln"
	"within.website/x/internal"
)

var (
	instance  = flag.String("instance", "", "mastodon instance")
	appID     = flag.String("app-id", "", "oauth2 app id")
	appSecret = flag.String("app-secret", "", "oauth2 app secret")
	token     = flag.String("token", "", "oauth2 token")
	sockPath  = flag.String("gpt2-sock", "/run/robocadey-gpt2.sock", "path to unix socket for robocadey-gpt2")
)

var scopes = []string{"read", "write", "follow"}

func getShitposts(sockPath string) ([]string, error) {
	var conn net.Conn
	var err error
	if sockPath != "" {
		conn, err = net.Dial("unix", sockPath)
	} else {
		conn, err = net.Dial("tcp", "[::1]:9999")
	}

	if err != nil {
		return nil, err
	}
	defer conn.Close()
	var result []string
	err = json.NewDecoder(conn).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func getShitpost(ctx context.Context) string {
	shitposts, err := getShitposts(*sockPath)
	if err != nil {
		ln.FatalErr(ctx, err)
	}

	return shitposts[rand.Intn(len(shitposts))]
}

func main() {
	internal.HandleStartup()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, err := madon.RestoreApp("furry boost bot", *instance, *appID, *appSecret, &madon.UserToken{AccessToken: *token})
	if err != nil {
		ln.FatalErr(ctx, err, ln.Action("madon.RestoreApp"))
	}
	_ = c

	rand.Seed(time.Now().UnixMicro())

	if _, err := c.PostStatus(madon.PostStatusParams{
		Text: getShitpost(ctx),
	}); err != nil {
		ln.FatalErr(ctx, err)
	}

	t := time.Tick(4 * time.Hour)

	for {
		evChan := make(chan madon.StreamEvent, 10)
		stop := make(chan bool)
		done := make(chan bool)

		err = c.StreamListener("user", "", evChan, stop, done)
		if err != nil {
			ln.FatalErr(ctx, err)
		}

		ln.Log(ctx, ln.F{
			"action": "streaming.toots",
		})

	outer:
		for {
			select {
			case _, ok := <-done:
				if !ok {
					ln.Fatal(ctx, ln.F{"action": "stream.dead"})
				}

			case <-t:
				if _, err := c.PostStatus(madon.PostStatusParams{
					Text: getShitpost(ctx),
				}); err != nil {
				}

			case ev := <-evChan:
				switch ev.Event {
				case "error":
					ln.Log(ctx, ln.F{"err": ev.Error, "action": "processing.event"})
					stop <- true
					close(evChan)
					close(stop)
					close(done)
					break outer
				case "notification":
					n := ev.Data.(madon.Notification)

					if n.Type == "mention" {
						ln.Log(ctx, ln.F{
							"target": n.Account.Acct,
						})
						if _, err := c.PostStatus(madon.PostStatusParams{
							Text:      "@" + n.Account.Acct + " " + getShitpost(ctx),
							InReplyTo: n.Status.ID,
						}); err != nil {
							ln.FatalErr(ctx, err)
						}
					}
				}
			}
		}
	}
}
