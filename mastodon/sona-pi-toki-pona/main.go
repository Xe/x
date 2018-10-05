package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	madon "github.com/McKael/madon/v2"
	"github.com/Xe/ln"
	"github.com/Xe/ln/opname"
	"github.com/Xe/x/web/tokipana"
	"github.com/jaytaylor/html2text"
	"github.com/joeshaw/envdecode"
	_ "github.com/joho/godotenv/autoload"
)

type lipuSona struct {
	AppID     string `env:"APP_ID,required"`
	AppSecret string `env:"APP_SECRET,required"`
	Token     string `env:"TOKEN,required"`
	Instance  string `env:"INSTANCE,required"`
}

func main() {
	ctx := opname.With(context.Background(), "main")
	var lipu lipuSona
	err := envdecode.StrictDecode(&lipu)
	if err != nil {
		ln.FatalErr(ctx, err)
	}

	c, err := madon.RestoreApp("sona-pi-toki-pona:", lipu.Instance, lipu.AppID, lipu.AppSecret, &madon.UserToken{AccessToken: lipu.Token})
	if err != nil {
		ln.FatalErr(opname.With(ctx, "restore-app"), err)
	}

	for {
		evChan := make(chan madon.StreamEvent, 10)
		stop := make(chan bool)
		done := make(chan bool)
		ctx = opname.With(ctx, "hashtag-stream")

		err = c.StreamListener("hashtag", "tokipona", evChan, stop, done)
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

					st, err := c.PostStatus(madon.PostStatusParams{
						Text:       fmt.Sprintf(translationTemplate, string(data)),
						InReplyTo:  s.ID,
						Visibility: "public",
					})
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
