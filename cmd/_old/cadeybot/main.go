package main

import (
	"bufio"
	"flag"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/joho/godotenv"
	"within.website/x/internal"
	"within.website/x/internal/flagenv"
)

var (
	token      = flag.String("token", "", "discord token")
	brainInput = flag.String("brain", "", "brain file")
)

func main() {
	flagenv.Parse()
	flag.Parse()
	internal.HandleStartup()

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

		err = chain.Save("cadey.gob")
		if err != nil {
			panic(err)
		}
	} else {
		err := chain.Load("cadey.gob")
		if err != nil {
			panic(err)
		}
	}

	rand.Seed(time.Now().Unix())

	mc := func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore all messages created by the bot itself
		// This isn't required in this specific example but it's a good practice.
		if m.Author.ID == s.State.User.ID {
			return
		}

		mentionsMe := false
		for _, us := range m.Mentions {
			if us.ID == s.State.User.ID {
				mentionsMe = true
				break
			}
		}

		if !mentionsMe {
			return
		}

		s.ChannelMessageSend(m.ChannelID, chain.Generate(15))
	}

	if *token == "" {
		log.Fatal("set -token or TOKEN")
	}

	dg, err := discordgo.New("Bot " + *token)
	if err != nil {
		log.Fatal(err)
	}

	dg.AddHandler(mc)

	err = dg.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer dg.Close()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}
