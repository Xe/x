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
		main()
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
			for {
				b.lookingForUserNetworks = true
				c.Writef("PRIVMSG *status ListAllUserNetworks")
				time.Sleep(2 * time.Second) // always sleep 2
				b.lookingForUserNetworks = false

				time.Sleep(1 * time.Hour)
			}
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
	split := strings.Fields(m.Trailing())
	username := m.Prefix.Name[1:]

	if len(split) == 0 {
		return
	}

	switch strings.ToLower(split[0]) {
	case "help":
		c.Writef("PRIVMSG ?%s :Commands available:", username)
		c.Writef("PRIVMSG ?%s :- ChangeName <new desired \"real name\">", username)
		c.Writef("PRIVMSG ?%s :  Changes your IRC \"real name\" to a new value instead of the default", username)
		c.Writef("PRIVMSG ?%s :- Reconnect", username)
		c.Writef("PRIVMSG ?%s :  Disconnects from PonyChat and connects to PonyChat again", username)
		c.Writef("PRIVMSG ?%s :- Help", username)
		c.Writef("PRIVMSG ?%s :  Shows this Message", username)
	case "changename":
		if len(split) < 1 {
			c.Writef("NOTICE %s :Usage: ChangeName <new desired \"real name\">")
			return
		}

		gecos := strings.Join(split[1:], " ")
		c.Writef("PRIVMSG *controlpanel :Set RealName %s %s", username, gecos)
		c.Writef("PRIVMSG ?%s :Please reply %q to confirm changing your \"real name\" to: %s", username, "Reconnect", gecos)
	case "reconnect":
		c.Writef("PRIVMSG ?%s :Reconnecting...", username)
		c.Writef("PRIVMSG *controlpanel Reconnect %s PonyChat", username)
	}
}

func (b *Bot) HandleCommand(c *irc.Client, m *irc.Message) {
	split := strings.Fields(m.Trailing())
	if split[0][0] == ';' {
		switch strings.ToLower(split[0][1:]) {
		case "request":
			c.Write("PRIVMSG #bnc :In order to request a BNC account, please connect to the bouncer server (bnc.ponychat.net, ssl port 6697, allow untrusted certs) with your nickserv username and passsword in the server password field (example: AzureDiamond:hunter2)")
		case "help":
			c.Write("PRIVMSG #bnc :PonyChat bouncer help is available here: https://ponychat.net/help/bnc/")
		case "rules":
			c.Write("PRIVMSG #bnc :Terms of the BNC")
			c.Write("PRIVMSG #bnc :- Do not use the BNC to evade channel bans")
			c.Write("PRIVMSG #bnc :- Do not use the BNC to violate any network rules")
			c.Write("PRIVMSG #bnc :- Do not use the BNC to connect to any other IRC network than PonyChat")
		}
	}
}

func (b *Bot) RemoveNetwork(c *irc.Client, username, network string) {
	c.Writef("PRIVMSG *controlpanel :DelNetwork %s %s", username, network)
}
