package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"

	madon "github.com/McKael/madon/v2"
	"github.com/Xe/x/internal"
	_ "github.com/Xe/x/tokipona"
	"github.com/Xe/x/web/tokipana"
	"github.com/jaytaylor/html2text"
	_ "github.com/joho/godotenv/autoload"
	"within.website/ln"
	"within.website/ln/opname"
)

var (
	instance  = flag.String("instance", "", "mastodon instance")
	appID     = flag.String("app-id", "", "oauth2 app id")
	appSecret = flag.String("app-secret", "", "oauth2 app secret")
	token     = flag.String("token", "", "oauth2 token")
	hashtag   = flag.String("hashtag", "tokipona", "hashtag to monitor")
)

func main() {
	internal.HandleStartup()

	ctx := opname.With(context.Background(), "main")
	ctx = ln.WithF(ctx, ln.F{"hashtag": *hashtag})

	c, err := madon.RestoreApp("sona-pi-toki-pona", *instance, *appID, *appSecret, &madon.UserToken{AccessToken: *token})
	if err != nil {
		ln.FatalErr(opname.With(ctx, "restore-app"), err)
	}

	ln.Log(ctx, ln.Info("waiting for messages"))

	for {
		evChan := make(chan madon.StreamEvent, 10)
		stop := make(chan bool)
		done := make(chan bool)
		ctx = opname.With(context.Background(), "hashtag-stream")

		err = c.StreamListener("hashtag", *hashtag, evChan, stop, done)
		if err != nil {
			ln.FatalErr(ctx, err)
		}

		for {
			select {
			case _, _ = <-done:
				goto redo
			case ev := <-evChan:
				switch ev.Event {
				case "error":
					ln.Error(opname.With(ctx, "event-parse"), err)
				case "update":
					s := ev.Data.(madon.Status)
					ctx = opname.With(ctx, "update")
					ctx = ln.WithF(ctx, ln.F{
						"originating_status_id":  s.ID,
						"originating_status_url": s.URL,
					})

					found := false
					for _, f := range *s.Account.Fields {
						if f.Name == "enable-bot" && f.Value == "sona-pi-toki-pona" {
							found = true
						}
					}
					if !found {
						ln.Log(ctx, ln.Info("ignoring message"))
						continue
					}

					text, err := html2text.FromString(s.Content, html2text.Options{PrettyTables: true})
					if err != nil {
						ln.Error(ctx, err)
						continue
					}

					req := tokipana.Translate(text)
					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						ln.Error(ctx, err)
						continue
					}
					err = tokipana.Validate(resp)
					if err != nil {
						ln.Error(ctx, err)
						continue
					}
					data, err := ioutil.ReadAll(resp.Body)
					resp.Body.Close()
					if err != nil {
						ln.Error(ctx, err)
						continue
					}

					st, err := c.PostStatus(
						madon.PostStatusParams{
							Text:       fmt.Sprintf(translationTemplate, string(data)),
							InReplyTo:  s.ID,
							Visibility: "public",
						},
					)
					if err != nil {
						ln.Error(ctx, err)
						continue
					}

					ln.Log(ctx, ln.Info("posted translation of toki pona text"), ln.F{"status_id": st.ID, "status_url": st.URL})
				}
			}
		}
	}

redo:
}

const translationTemplate = `Translated into English: %s`
