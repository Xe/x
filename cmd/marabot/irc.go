package main

import (
	"context"
	"database/sql"
	"flag"

	irc "github.com/thoj/go-ircevent"
	"within.website/ln"
	"within.website/ln/opname"
)

var (
	ircNick         = flag.String("irc-nick", "[Mara]", "IRC nickname")
	ircUser         = flag.String("irc-user", "sh0rk", "IRC username")
	ircReal         = flag.String("irc-real", "Friendly sh0rk Mara", "IRC realname")
	ircServer       = flag.String("irc-server", "chrysalis:6667", "IRC server to connect to")
	ircSASLUsername = flag.String("irc-sasl-username", "", "SASL username")
	ircSASLPassword = flag.String("irc-sasl-password", "", "SASL password")
)

func NewIRCBot(ctx context.Context, db *sql.DB, messages chan string) {
	ctx = opname.With(ctx, "ircbot")
	ctx = ln.WithF(ctx, ln.F{
		"irc_server": *ircServer,
	})
	for {
		irccon := irc.IRC(*ircNick, *ircUser)
		go func() {
			<-ctx.Done()
			irccon.Disconnect()
		}()

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-messages:
					irccon.Privmsg("#xeserv", msg)
				}
			}
		}()

		if *ircSASLUsername != "" && *ircSASLPassword != "" {
			irccon.UseSASL = true
			irccon.SASLLogin = *ircSASLUsername
			irccon.SASLPassword = *ircSASLPassword
			irccon.SASLMech = "plain"
		}

		irccon.AddCallback("001", func(e *irc.Event) { irccon.Join("#xeserv") })
		irccon.AddCallback("PRIVMSG", func(e *irc.Event) {
			if _, err := db.ExecContext(ctx, `INSERT INTO irc_messages(nick, user, host, channel, content, tags) VALUES (?, ?, ?, ?, ?, ?)`, e.Nick, e.User, e.Host, e.Arguments[0], e.Message(), ""); err != nil {
				ln.Error(ctx, err)
			}
		})
		err := irccon.Connect(*ircServer)
		if err != nil {
			ln.Error(ctx, err)
			return
		}
		irccon.Loop()
	}
}
