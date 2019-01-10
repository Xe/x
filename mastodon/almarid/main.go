package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/McKael/madon"
	"github.com/caarlos0/env"
	_ "github.com/joho/godotenv/autoload"
	"within.website/ln"
)

var cfg = &struct {
	Instance  string `env:"INSTANCE,required"`
	AppID     string `env:"APP_ID,required"`
	AppSecret string `env:"APP_SECRET,required"`
	Token     string `env:"TOKEN,required"`
	WordFile  string `env:"WORD_FILE,required"`
}{}

var ctx = context.Background()

func main() {
	err := env.Parse(cfg)
	if err != nil {
		ln.Fatal(ctx, ln.F{"err": err, "action": "env.Parse"})
	}

	fin, err := os.Open(cfg.WordFile)
	if err != nil {
		ln.Fatal(ctx, ln.F{"err": err, "action": "os.Open(cfg.WordFile)"})
	}

	data, err := ioutil.ReadAll(fin)
	if err != nil {
		ln.Fatal(ctx, ln.F{"err": err, "action": "ioutil.ReadAll(fin)"})
	}

	c, err := madon.RestoreApp("almarid:", cfg.Instance, cfg.AppID, cfg.AppSecret, &madon.UserToken{AccessToken: cfg.Token})
	if err != nil {
		ln.Fatal(ctx, ln.F{"err": err, "action": "madon.RestoreApp"})
	}
	_ = c

	lines := bytes.Split(data, []byte("\n"))
	words := []string{}

	for _, line := range lines {
		if len(line) > 5 {
			word := string(line)

			if strings.HasPrefix(word, "'") {
				word = word[1:]
			}

			words = append(words, word)
		}
	}

	ln.Log(ctx, ln.F{"action": "words.loaded", "count": len(words)})

	lenBig := big.NewInt(int64(len(words)))

	first := true

	for {
		bi, err := rand.Int(rand.Reader, lenBig)
		if err != nil {
			ln.Log(ctx, ln.F{
				"action": "big.Rand",
				"err":    err,
			})

			continue
		}

		i := int(bi.Int64())

		if first {
			first = false
		} else {
			time.Sleep(600 * time.Minute)
		}

		txt := fmt.Sprintf("%s is not doing, allah is doing", words[i])

		st, err := c.PostStatus(txt, 0, nil, false, "", "private")
		if err != nil {
			ln.Log(ctx, ln.F{
				"err":    err,
				"action": "c.PostStatus",
				"text":   txt,
			})

			continue
		}

		ln.Log(ctx, ln.F{
			"action": "tooted",
			"text":   txt,
			"id":     st.ID,
			"url":    st.URL,
		})
	}
}
