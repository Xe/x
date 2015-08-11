package main

import (
	"bufio"
	"flag"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/thoj/go-ircevent"
)

var (
	brainInput = flag.String("brain", "", "brain file")
	lastSpoken time.Time
)

func main() {
	flag.Parse()

	chain := NewChain(3)

	if *brainInput != "" {
		log.Printf("Opening %s...", *brainInput)

		fin, err := os.Open(*brainInput)
		if err != nil {
			panic(err)
		}

		s := bufio.NewScanner(fin)
		for s.Scan() {
			t := s.Text()

			_, err := chain.Write(t)
			if err != nil {
				panic(err)
			}
		}

		err = chain.Save("mybrain-pc.gob")
		if err != nil {
			panic(err)
		}
	} else {
		err := chain.Load("mybrain-pc.gob")
		if err != nil {
			panic(err)
		}
	}

	rand.Seed(time.Now().Unix())

	conn := irc.IRC("BeefSupreme", "Doritos")

	err := conn.Connect("irc.ponychat.net:6667")
	if err != nil {
		panic(err)
	}

	conn.AddCallback("001", func(e *irc.Event) {
		conn.Join("#geek")
	})

	conn.AddCallback("PRIVMSG", func(e *irc.Event) {
		log.Printf("writing brain with %s", e.Arguments[1])
		chain.Write(e.Arguments[1])
		chain.Save("mybrain-pc.gob")
	})

	conn.AddCallback("PRIVMSG", func(e *irc.Event) {
		if lastSpoken.Add(5 * time.Minute).Before(time.Now()) {
			log.Println("It's been long enough that I can speak!")

			if rand.Int()%4 == 2 {
				log.Printf("About to say something...")
				time.Sleep(time.Duration((rand.Int()%15)+4) * time.Second)
				conn.Privmsg(e.Arguments[0], chain.Generate(15))

				lastSpoken = time.Now()
			}
		}
	})

	conn.Loop()
}
