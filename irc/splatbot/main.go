package main

import (
	"flag"
	"fmt"

	"github.com/belak/irc"
)

var (
	nick     = flag.String("nick", "Inkling", "Nickname to use")
	user     = flag.String("user", "xena", "Username to use")
	channels = flag.String("channels", "#niichan", "Comma-separated list of channels to join")
	server   = flag.String("server", "irc.ponychat.net", "Server to connect to")
	port     = flag.Int("port", 6697, "port to connect to")
	ssl      = flag.Bool("ssl", true, "Use ssl?")
)

func init() {
	flag.Parse()
}

func main() {
	handler := irc.NewBasicMux()

	client := irc.NewClient(irc.HandlerFunc(handler.HandleEvent), *nick, *user, "SplatBot by Xena", "")

	handler.Event("001", func(c *irc.Client, e *irc.Event) {
		c.Writef("MODE %s +B", c.CurrentNick())
		c.Writef("JOIN %s", *channels)
	})

	handler.Event("PRIVMSG", handlePrivmsg)

	var err error

	if *ssl {
		err = client.DialTLS(fmt.Sprintf("%s:%d", *server, *port), nil)
	} else {
		err = client.Dial(fmt.Sprintf("%s:%d", *server, *port))
	}
	if err != nil {
		panic(err)
	}
}
