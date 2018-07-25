package main

import (
	"context"
	"crypto/tls"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Xe/ln"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/joho/godotenv/autoload"
	irc "gopkg.in/irc.v1"
)

var (
	addr      = os.Getenv("SERVER")
	password  = os.Getenv("PASSWORD")
	tgToken   = os.Getenv("TELEGRAM_TOKEN")
	pokeIDStr = os.Getenv("POKE_ID")

	sclock sync.Mutex
	scores map[string]float64
)

var ctx context.Context

func main() {
	scores = map[string]float64{}

	pokeID, err := strconv.Atoi(pokeIDStr)
	if err != nil {
		log.Fatal(err)
	}

	ba, err := tgbotapi.NewBotAPI(tgToken)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := tls.Dial("tcp", addr, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx = context.Background()
	ctx = ln.WithF(ctx, ln.F{"app": "clevelandbrown"})

	ln.Log(ctx, ln.F{
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

	ff := ln.FilterFunc(func(ctx context.Context, e ln.Event) bool {
		if val, ok := e.Data["svclog"]; ok && val.(bool) {
			delete(e.Data, "svclog")

			line, err := ln.DefaultFormatter.Format(ctx, e)
			if err != nil {
				ln.FatalErr(ctx, err)
			}

			err = cli.Writef("PRIVMSG #services :%s", string(line))
			if err != nil {
				log.Fatal(err)
			}

			msg := tgbotapi.NewMessage(int64(pokeID), string(line))

			_, err = ba.Send(msg)
			if err != nil {
				log.Printf("can't send telegram message: %v", err)
				return true
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

			ln.Log(ctx, ln.F{
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

			ln.Log(ctx, ln.F{
				"action":  "reaped_scores",
				"removed": rem,
				"halved":  halved,
			})

			sclock.Unlock()
		}
	}()

	ln.Log(ctx, ln.F{
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
		ln.Fatal(ctx, ln.F{
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

	// channels to ignore
	switch m.Params[0] {
	case "#services", "#/dev/syslog":
		return
	}

	//opt-out list
	switch m.Prefix.Name {
	case "Taz", "cadance-syslog", "FromDiscord", "Sonata_Dusk", "CQ_Discord", "Onion":
		return
	case "Sparkler", "Aeyris", "Ryunosuke", "WaterStar", "Pegasique":
		return
	}

	sc, ok := scores[m.Prefix.Host]
	if !ok {
		sc = 0
	}

	for _, line := range lines {
		if strings.Contains(strings.ToLower(m.Trailing()), line) {
			sc += 1

			ln.Log(ctx, ln.F{
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
			const delta = 5
			sc += delta
			ln.Log(ctx, ln.F{
				"action":  "efknockr_detected",
				"score":   sc,
				"user":    m.Prefix.String(),
				"channel": m.Params[0],
				"delta":   delta,
				"svclog":  true,
			})
		}
	}

	scores[m.Prefix.Host] = sc

	if sc >= notifyThreshold {
		ln.Log(ctx, ln.F{
			"action":  "warn",
			"channel": m.Params[0],
			"user":    m.Prefix.String(),
			"score":   sc,
			"svclog":  true,
			"ping":    "Xena",
		})
	}

	if sc > autobanThreshold {
		c.Writef("PRIVMSG OperServ :AKILL ADD %s spamming | Cleveland show spammer", m.Prefix.Name)

		ln.Log(ctx, ln.F{
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
	"ARE YOU MAD THOSE PORCH *ONKEYS ARE ALWAYS BITCHING ABOUT RACISM??",
	"DO YOU THINK THEY BELONG IN A ZOO WITH OBAMA EATING BANANA'S??",
	"PLEASE JOIN #/JOIN ON irc.freenode.net OR MESSAGE VAP0R ON FREENODE",
	"FOR INFORMATION ON A SOON MEETING OF SIMILARLY MINDED INVIDIDUALS",
	"URGENT!! URGENT!! URGENT!! URGENT!! URGENT!!",
	"THE DARKIES WHO LOOK LIKE GUERRILLAS WANT OUT OF THE ZOO!!",
	"IF YOU WANT THEM TO STAY THERE IS A MEETING IN 30 MINS!!!",
	"TRUMP TRUMP TRUMP TRUMP TRUMP TRUMP TRUMP TRUMP TRUMP TRUMP",
	"ARE YOU TIRED OF PEOPLE WHO LOOK LIKE MONKEYS",
	"TRYING TO TAKE MONUMENTS DOWN AND TAKING FREESPEECH AWAY??",
	"HAVE YOU COME TO REALIZE THAT THEY HAVE A IQ OF 10 AND",
	"CAN ONLY EAT BANANAS? IF SO PLEASE CHECK OUT",
	"freedomeu4y6vlqu.onion/6667 (FREESPEECH IRC)",
	"freedomeu4y6vlqu.onion",
	"HAVE YOU EVER WANTED TO USE A TERMINAL IRC PROGRAM?",
	"BUT DID NOT HAVE THE NECESSARY COMPUTER SKILLS?",
	"WEECHAT IS NOW OFFERING TECH SUPPORT FOR ONLY",
	"$10 DOLLARS A YEAR!! PLEASE VISIT #WEECHAT",
	"IRC.FREENODE.NET FOR MORE INFORMATION!!",
	"TECHPONIES IS NOW OFFERING TECH SUPPORT FOR ONLY",
	"$10 DOLLARS A YEAR!! PLEASE VISIT #TECHPONIES",
	"IRC.CANTERNET.ORG FOR MORE INFORMATION!!",
	"ITS FUND-RAISER WEEK!!",
	"DONATIONS ARE NEEDED FOR THIS GREAT WORK!!",
	"PLEASE VISIT #WEECHAT",
	"IRC.FREENODE.NET FOR MORE INFORMATION!!",
	"type !donation in channel for donation info",
	"DON'T YOU THINK ITS TIME FOR A HONEST CONVERSATION",
	"ON HOW DUMB NIGGAS ARE?? PLEASE JOIN THE CONVO AT",
	"dumniggaff3fjhrw.onion or with port 6667 (IRC)",
	"LOOK AT ALL THE NIGGERS LOOTING IN FLORIDA ROFL!!",
	"JOIN THE DISCUSSION torniggaiaoxhlcl.onion/6667",
	"When disaster strikes...niggers go shopping",
	"https://www.youtube.com/watch?v=uVee9A2IK0I",
	"https://www.youtube.com/watch?v=AZkJzWFT6rA <== all niggers rofl",
	"JOIN THE DISCUSSION dumniggaff3fjhrw.onion/6667",
	"ARE YOU TIRED OF THOSE NIGGERS DISRESPECTING THE AMERICAN FLAG??",
	"DO YOU AGREE WITH TRUMP THAT THOSE NIGGERS SHOULD BE FIRED",
	"KEKISTAN IS HELPING ORGANIZING FREE SPEECH WEEK!!",
	"HE CAN BE CONTACTED AT IRC.MADIRC.NET #1337",
	"gv4z277ijqyn7uenr5pdvsovoawibwcjtlqgkxkfifdcs7csshpq.b32.i2p",
	"(accessible with tunnel)",
	"http://lvb6wabr3fuv7l2lmmaj33jwh7ntb7uuhmfmluc7hwtf6rm36k6q.b32.i2p/",
	"(kiwi client on webpage)",
	"PLEASE CALL L0DE RIGHT NOW!!!",
	"415-349-5666",
	"his live show @",
	"https://www.youtube.com/watch?v=rXWx3lPlwgE",
	"WE ARE TRYING TO INCREASE PARTICIPATION IN THIS SHOW",
	"PLEASE CALL AND PARTICIPATE.",
	"LOOK BITCHES CALL THE SHOW",
	"LIVE DISCUSSION ON WHY NIGGERS ARE A CURSE FOR THE USA",
	"or go to #lrh efnet irc for more information",
	"THOSE STUPID MUSLIM SAND NIGGERS HAVE KILLED INNOCENT AMERICANS AGAIN",
	"THE NAZI ORGANIZATION OF AMERICA IS PLANNING AN EMERGENCY MEETING",
	"TODAY @ #/JOIN ON IRC.FREENODE.NET. DO NOT COMPLAIN IN #FREENODE",
	"THIS MEETING IS INTENDED TO BE FOR MORE LIKEMINDED INDIVIDUALS",
	"IF YOU HAVE QUESTIONS PLEASE DONT HESISTATE SENDING A MESSAGE TO",
	"VAP0R ON IRC.FREENODE.NET.",
	"The l0de Radio Hour is LIVE ! Call in now /join #LRH on irc.efnet.org !!! JOIN ! NOW !",
	"https://www.youtube.com/watch?v=DIZqYgaOchY",
	"JOIN #LRH EFNET TO JOIN THE DISCUSSION",
	"ARE YOU INTERESTED IN THE WHITE NATIONALIST MOVEMENT?",
	"DO YOU WONDER WHY WHITE PEOPLE ARE SMARTER THAN NIGGERS?",
	"PLEASE JOIN #/JOIN ON FREENODE IRC NETWORK AND MESSAGE",
	"VAP0R ON HOW TO JOIN THIS GROWING MOVEMENT!!",
	"IRC.SUPERNETS.ORG #SUPERBOWL FEB 4. SUPERBOWL PARTY!!",
	"ASK CHRONO FOR DETAILS!!",
	"Hey, I thought you guys might be interested in this blog by freenode staff member Bryan 'kloeri' Ostergaard https://bryanostergaard.com/",
	"or maybe this blog by freenode staff member Matthew 'mst' Trout https://MattSTrout.com",
	"Read what IRC investigative journalists have uncovered on the freenode pedophilia scandal https://encyclopediadramatica.rs/Freenodegate",
	"Voice your opinions at https://webchat.freenode.net/?channels=#freenode",
	"This message was brought to you by Private Internet Access. Voice your opinions at https://webchat.freenode.net/?channels=%23freenode",
	"<script type=\"text/javascript\" src=\"http://web.nba1001.net:8888/tj/tongji.js\"></script>",
}
