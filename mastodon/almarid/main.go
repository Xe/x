package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/McKael/madon/v2"
	"github.com/Xe/x/internal"
	"within.website/ln"
)

var (
	instance  = flag.String("instance", "", "mastodon instance")
	appID     = flag.String("app-id", "", "oauth2 app id")
	appSecret = flag.String("app-secret", "", "oauth2 app secret")
	token     = flag.String("token", "", "oauth2 token")
	wordFile  = flag.String("word-file", "./words.txt", "wordlist file")
	every     = flag.Duration("every", 12*time.Hour, "duration between utterances")
)

var ctx = context.Background()

func main() {
	internal.HandleStartup()

	fin, err := os.Open(*wordFile)
	if err != nil {
		ln.Fatal(ctx, ln.F{"err": err, "action": "os.Open(cfg.WordFile)"})
	}

	data, err := ioutil.ReadAll(fin)
	if err != nil {
		ln.Fatal(ctx, ln.F{"err": err, "action": "ioutil.ReadAll(fin)"})
	}

	c, err := madon.RestoreApp("almarid:", *instance, *appID, *appSecret, &madon.UserToken{AccessToken: *token})
	if err != nil {
		ln.Fatal(ctx, ln.F{"err": err, "action": "madon.RestoreApp"})
	}

	ctx = ln.WithF(ctx, ln.F{"every": *every, "instance": *instance, "iam": c.Name})

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
			time.Sleep(*every)
		}

		txt := fmt.Sprintf("%s is not doing, allah is doing", words[i])

		st, err := c.PostStatus(madon.PostStatusParams{
			Text:       txt,
			Visibility: "private",
		})
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
