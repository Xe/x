package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"time"

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
	WordFile  string `env:"WORD_FILE,required"`
}{}

func main() {
	err := env.Parse(cfg)
	if err != nil {
		ln.Fatal(ln.F{"err": err, "action": "env.Parse"})
	}

	fin, err := os.Open(cfg.WordFile)
	if err != nil {
		ln.Fatal(ln.F{"err": err, "action": "os.Open(cfg.WordFile)"})
	}

	data, err := ioutil.ReadAll(fin)
	if err != nil {
		ln.Fatal(ln.F{"err": err, "action": "ioutil.ReadAll(fin)"})
	}

	c, err := madon.RestoreApp("furry boostbot", cfg.Instance, cfg.AppID, cfg.AppSecret, &madon.UserToken{AccessToken: cfg.Token})
	if err != nil {
		ln.Fatal(ln.F{"err": err, "action": "madon.RestoreApp"})
	}
	_ = c

	n := time.Now().UnixNano()
	rand.Seed(n * n)

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

	ln.Log(ln.F{"action": "words.loaded", "count": len(words)})
	perms := rand.Perm(len(words))

	for i := range perms {
		//time.Sleep(5 * time.Minute)

		txt := fmt.Sprintf("%s is not doing, allah is doing", words[i])

		//st, err := c.PostStatus(txt, 0, nil, false, "", "public")
		//if err != nil {
		//	ln.Log(ln.F{
		//		"err":    err,
		//		"action": "c.PostStatus",
		//		"text":   txt,
		//	})
		//
		//	continue
		//}

		ln.Log(ln.F{
			"action": "tooted",
			"text":   txt,
			//"id": st.ID,
			//"url": st.URL,
		})
	}
}
