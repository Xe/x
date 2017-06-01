package main

import (
	"crypto/tls"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Xe/ln"
	_ "github.com/joho/godotenv/autoload"
	irc "gopkg.in/irc.v1"
)

var (
	addr     = os.Getenv("SERVER")
	password = os.Getenv("PASSWORD")

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

	ln.Log(ln.F{
		"action": "connected",
		"where":  addr,
	})

	cli := irc.NewClient(conn, irc.ClientConfig{
		Handler: irc.HandlerFunc(scoreCleveland),
		Nick:    "Xena",
		User:    "xena",
		Name:    "cleveland brown termination bot",
		Pass:    password,
	})

	ff := ln.FilterFunc(func(e ln.Event) bool {
		if val, ok := e.Data["svclog"]; ok && val.(bool) {
			delete(e.Data, "svclog")

			line, err := ln.DefaultFormatter.Format(e)
			if err != nil {
				ln.Fatal(ln.F{"err": err})
			}

			err = cli.Writef("PRIVMSG #services :%s", string(line))
			if err != nil {
				log.Fatal(err)
			}
		}

		return true
	})
	ln.DefaultLogger.Filters = append(ln.DefaultLogger.Filters, ff)

	go func() {
		for {
			time.Sleep(30 * time.Second)

			sclock.Lock()
			defer sclock.Unlock()

			changed := 0
			ignored := 0

			for key, sc := range scores {
				if sc >= notifyThreshold {
					ignored++
					continue
				}

				scores[key] = sc / 100
				changed++
			}

			sclock.Unlock()

			ln.Log(ln.F{
				"action":  "nerfed_scores",
				"changed": changed,
				"ignored": ignored,
			})
		}
	}()

	go func() {
		for {
			time.Sleep(5 * time.Minute)

			sclock.Lock()
			defer sclock.Unlock()

			nsc := map[string]float64{}

			halved := 0
			rem := 0

			for key, score := range scores {
				if score > 0.01 {
					if score > 3 {
						score = score / 2
						halved++
					}

					nsc[key] = score
				} else {
					rem++
				}
			}

			scores = nsc

			ln.Log(ln.F{
				"action":  "reaped_scores",
				"removed": rem,
				"halved":  halved,
				"svclog":  true,
			})

			sclock.Unlock()
		}
	}()

	ln.Log(ln.F{
		"action": "accepting_input",
		"svclog": true,
	})

	cli.Run()
}

const (
	notifyThreshold  = 3
	autobanThreshold = 10
)

func scoreCleveland(c *irc.Client, m *irc.Message) {
	if m.Trailing() == "!ohshitkillit" && m.Prefix.Host == "ponychat.net" {
		ln.Fatal(ln.F{
			"action":  "emergency_stop",
			"user":    m.Prefix.String(),
			"channel": m.Params[0],
			"svclog":  true,
		})
	}

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

	switch m.Params[0] {
	case "#services", "#/dev/syslog":
		return
	}

	switch m.Prefix.Name {
	case "Taz", "cadance-syslog", "FromDiscord", "Sonata_Dusk", "CQ_Discord", "Onion":
		return

	case "Sparkler":
		// (Sparkler) lol
		// (Sparkler) don't banzor me :(
		return

	case "Aeyris":
		// known shitposter, collison risk :(
		return

	case "Ryunosuke", "WaterStar":
		return
	}

	sc, ok := scores[m.Prefix.Host]
	if !ok {
		sc = 0
	}

	for _, line := range lines {
		if strings.Contains(strings.ToLower(m.Trailing()), line) {
			sc += 1

			ln.Log(ln.F{
				"action":     "siren_compare",
				"channel":    m.Params[0],
				"user":       m.Prefix.String(),
				"scoredelta": 1,
				"svclog":     true,
			})
		}
	}

	thisLine := strings.ToLower(m.Trailing())

	for _, efnLine := range efknockr {
		if strings.Contains(thisLine, strings.ToLower(efnLine)) {
			sc += 3
			ln.Log(ln.F{
				"action":  "efknockr_detected",
				"score":   sc,
				"user":    m.Prefix.String(),
				"channel": m.Params[0],
				"delta":   3,
				"svclog":  true,
			})
		}
	}

	scores[m.Prefix.Host] = sc

	if sc >= notifyThreshold {
		ln.Log(ln.F{
			"action":  "warn",
			"channel": m.Params[0],
			"user":    m.Prefix.String(),
			"score":   sc,
			"svclog":  true,
			"ping":    "Xena",
		})
	}

	if sc >= autobanThreshold {
		c.Writef("PRIVMSG OperServ :AKILL ADD %s spamming | Cleveland show spammer", m.Prefix.Name)
		c.Writef("PRIVMSG %s :Sorry for that, he's gone now.", m.Params[0])

		ln.Log(ln.F{
			"action":  "kline_added",
			"channel": m.Params[0],
			"user":    m.Prefix.String(),
			"score":   sc,
			"svclog":  true,
		})

		scores[m.Prefix.Host] = 0
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

var lines = []string{
	"my name is cleveland brown and I am proud to be",
	"my name is cl3v3land brown and i am proud to be",
	"right back in my hometown with my new family",
	"right back in my hometown with my n3w family",
	"there's old friends and new friends and even a bear",
	"through good times and bad times it's true love we share.",
	"and so I found a place",
	"where everyone will know",
	"my happy mustache face",
	"this is the cleveland show! haha!",
}

var efknockr = []string{
	"THIS NETWORK IS FUCKING BLOWJOBS LOL COME TO WORMSEC FOR SOME ICE COLD CHATS",
	"0 DAY BANANA BOMBS \"OK\"",
	"IRC.WORMSEC.US",
	"THE HOTTEST MOST EXCLUSIVE SEC ON THE NET",
	"THIS NETWORK IS BLOWJOBS! GET ON SUPERNETS FOR COLD HARD CHATS NOW",
	"IRC.SUPERNETS.ORG | PORT 6667/6697 (SSL) | #SUPERBOWL | IPV6 READY",
	"▓█▓▓▓▒▒▒▒▒▒▓▓▓▓▓▓▓▓▓▓▄░  ░▄▓▓▓▓▓▓▓▓▓█▓▓▓  IRC.WORMSEC.US  |  #SUPERBOWL",
	"THIS NETWORK IS BLOWJOBS! GET ON SUPERNETS FOR COLD HARD CHATS NOW",
	"▄",
	"███▓▓▒▒▒▒▒▒▒░░░               ░░░░▒▒▒▓▓▓▓",
	"▓█▓▓▓▒▒▒▒▒▒▓▓▓▓▓▓▓▓▓▓▄░   ░▄▓▓▓▓▓▓▓▓▓█▓▓▓",
	"▒▓▓▓▓▒▒░░▒█▓▓▓▓▓▓▓▓▓▓█░▒░░▒▓▓▓▓▓▓▓▓▓▓▓█▓▓",
	"░▒▓▓▒▒▒▒░░▒▒█▓▓▓▓▓▓▓▓▓█░▒░░░▒▓▓▓▓▓▓▓▓▓▓█▒▓░",
	"▒▒▒▒▒▒▒▒▒▒▒░░▀▀▀▀▀▀▀ ░▒░░   ░▒▒▒▀▀▀▀▀▀▒▓▓▓▒",
	"THE HOTTEST MOST EXCLUSIVE SEC ON THE NET",
	"â–‘â–’â–“â–“â–’â–’â–’â–’â–‘â–‘â–’",
	"Techman likes to fuck kids in the ass!!",
	"https://discord.gg/3b86TH7",
	"|     |\\\\",
	"/     \\ ||",
	"(  ,(   )=m=D~~~ LOL DONGS",
	"/  / |  |",
}
