package main

import (
	"crypto/tls"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/AllenDang/simhash"

	irc "gopkg.in/irc.v1"

	_ "github.com/joho/godotenv/autoload"
)

var (
	addr        = os.Getenv("SERVER")
	password    = os.Getenv("PASSWORD")
	monitorChan = os.Getenv("MONITOR_CHAN")

	sclock sync.Mutex
	scores map[string]float64
)

func main() {
	scores = map[string]float64{}

	conn, err := tls.Dial("tcp", addr, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	cli := irc.NewClient(conn, irc.ClientConfig{
		Handler: irc.HandlerFunc(scoreCleveland),
		Nick:    "Xena",
		User:    "xena",
		Name:    "cleveland brown termination bot",
		Pass:    password,
	})

	go func() {
		for {
			time.Sleep(30 * time.Second)

			sclock.Lock()
			defer sclock.Unlock()

			scores = map[string]float64{}

			sclock.Unlock()
		}
	}()

	cli.Run()
}

func scoreCleveland(c *irc.Client, m *irc.Message) {
	sclock.Lock()
	defer sclock.Unlock()

	if m.Command != "PRIVMSG" {
		return
	}

	/*
		if !strings.HasSuffix(m.Params[0], monitorChan) {
				return
			}
	*/

	log.Printf("%#v", m)

	sc, ok := scores[m.Prefix.Name]
	if !ok {
		sc = 0
	}

	lv := simhash.GetLikenessValue(strings.ToLower(m.Params[1]), showLyrics)
	log.Printf("%s: %v", m.Params[1], lv)

	sc += lv

	scores[m.Prefix.Name] = sc

	log.Printf("%s: %v", m.Prefix.Name, sc)

	if sc >= 8.0 {
		c.Writef("PRIVMSG #opers :%s has cleveland themeshow similarity score %v", m.Prefix.Name, sc)
	}
}

const showLyrics = `my name is cleveland brown and I am proud to be
right back in my hometown with my new family.
there's old friends and new friends and even a bear.
through good times and bad times it's true love we share.
and so I found a place
where everyone will know
my happy mustache face
this is the cleveland show! haha!`
