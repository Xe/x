package main

import (
	"github.com/McKael/madon"
	"github.com/Xe/ln"
	"github.com/caarlos0/env"
	_ "github.com/joho/godotenv/autoload"
)

var cfg = &struct {
	Instance  string `env:"INSTANCE,required"`
	AppID     string `env:"APP_ID,required"`
	AppSecret string `env:"APP_SECRET,required"`
	Token     string `env:"TOKEN,required"`
	Hashtag   string `env:"HASHTAG,required"`
}{}

var scopes = []string{"read", "write", "follow"}

func main() {
	err := env.Parse(cfg)
	if err != nil {
		ln.Fatal(ln.F{"err": err, "action": "startup"})
	}

	c, err := madon.RestoreApp("furry boostbot", cfg.Instance, cfg.AppID, cfg.AppSecret, &madon.UserToken{AccessToken: cfg.Token})
	if err != nil {
		ln.Fatal(ln.F{"err": err, "action": "madon.RestoreApp"})
	}

	evChan := make(chan madon.StreamEvent, 10)
	stop := make(chan bool)
	done := make(chan bool)

	err = c.StreamListener("public", "", evChan, stop, done)
	if err != nil {
		ln.Fatal(ln.F{"err": err, "action": "c.StreamListener"})
	}

	ln.Log(ln.F{
		"action": "streaming.toots",
	})

	for {
		select {
		case _, ok := <-done:
			if !ok {
				ln.Fatal(ln.F{"action": "stream.dead"})
			}

		case ev := <-evChan:
			switch ev.Event {
			case "error":
				ln.Fatal(ln.F{"err": ev.Error, "action": "processing.event"})
			case "update":
				s := ev.Data.(madon.Status)

				for _, tag := range s.Tags {
					if tag.Name == cfg.Hashtag {
						err = c.ReblogStatus(s.ID)
						if err != nil {
							ln.Fatal(ln.F{"err": err, "action": "c.ReblogStatus", "id": s.ID})
						}

						ln.Log(ln.F{
							"action": "reblogged",
							"id":     s.ID,
						})
					}
				}
			}
		}
	}
}
