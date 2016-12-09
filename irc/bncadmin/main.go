package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/belak/irc"
	_ "github.com/joho/godotenv/autoload"
)

var (
	bncUsername          = needEnv("BNC_USERNAME")
	bncPassword          = needEnv("BNC_PASSWORD")
	bncServer            = needEnv("BNC_SERVER")
	serverSuffixExpected = needEnv("SERVER_SUFFIX")
)

func needEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatal("need value for " + key)
	}

	return v
}

func main() {
	log.Println("Bot connecting to " + bncServer)
	conn, err := tls.Dial("tcp", bncServer, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	c := irc.NewClient(conn, irc.ClientConfig{
		Nick: "admin",
		Pass: fmt.Sprintf("%s:%s", bncUsername, bncPassword),
		User: "BNCbot",
		Name: "BNC admin bot",

		Handler: NewBot(),
	})

	for _, cap := range []string{"userhost-in-names", "multi-prefix", "znc.in/server-time-iso"} {
		c.Writef("CAP REQ %s", cap)
	}

	err = c.Run()
	if err != nil {
		panic(err)
	}
}

type Bot struct {
	setupDaemon            sync.Once
	lookingForUserNetworks bool

	// i am sorry
	launUsername string
}

func NewBot() *Bot {
	return &Bot{}
}

func (b *Bot) Handle(c *irc.Client, m *irc.Message) {
	b.setupDaemon.Do(func() {
		go func() {
			b.lookingForUserNetworks = true
			c.Writef("PRIVMSG *status ListAllUserNetworks")
			time.Sleep(2 * time.Second) // always sleep 2
			b.lookingForUserNetworks = false

			time.Sleep(1 * time.Hour)
		}()
	})

	// log.Printf("in >> %s", m)

	switch m.Command {
	case "PRIVMSG":
		if m.Prefix.Name == "*status" {
			b.HandleStarStatus(c, m)
		}

		if strings.HasPrefix(m.Prefix.Name, "?") {
			b.HandlePartyLineCommand(c, m)
		}

		if m.Params[0] == "#bnc" {
			b.HandleCommand(c, m)
		}

	case "NOTICE":
		if m.Prefix.Name == "*status" {
			f := strings.Fields(m.Trailing())
			if f[0] == "***" {
				log.Println(m.Trailing())
				// look up geoip and log here
			}
		}
	}
}

func (b *Bot) HandleStarStatus(c *irc.Client, m *irc.Message) {
	if b.lookingForUserNetworks {
		if strings.HasPrefix(m.Trailing(), "| ") {
			f := strings.Fields(m.Trailing())

			switch len(f) {
			case 11: // user name line
				//  11: []string{"|", "AzureDiamond", "|", "N/A", "|", "0", "|", "|", "|", "|", "|"}
				username := f[1]
				b.launUsername = username

			case 15: // server and nick!user@host line
				// 15: []string{"|", "`-", "|", "PonyChat", "|", "0", "|", "Yes", "|", "amethyststar.ponychat.net", "|", "test!test@lypbmzxixk.ponychat.net", "|", "1", "|"}
				server := f[9]
				network := f[3]
				if !strings.HasSuffix(server, serverSuffixExpected) {
					log.Printf("%s is using the BNC to connect to unknown server %s, removing permissions", b.launUsername, server)
					b.RemoveNetwork(c, b.launUsername, network)
					c.Writef("PRIVMSG ?%s :You have violated the terms of the BNC service and your account has been disabled. Please contact PonyChat staff to appeal this.", b.launUsername)
					c.Writef("PRIVMSG *blockuser block %s", b.launUsername)
				}
			}
		}
	}
}

func (b *Bot) HandlePartyLineCommand(c *irc.Client, m *irc.Message) {

}

func (b *Bot) HandleCommand(c *irc.Client, m *irc.Message) {

}

func (b *Bot) RemoveNetwork(c *irc.Client, username, network string) {
	c.Writef("PRIVMSG *controlpanel :DelNetwork %s %s", username, network)
}
