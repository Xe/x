package main

import (
	"context"
	"flag"
	"time"

	irc "github.com/thoj/go-ircevent"
	"within.website/ln"
	"within.website/ln/opname"
	"within.website/x/internal"
	"within.website/x/web/revolt"
)

var (
	ircNick          = flag.String("irc-nick", "[Mara]", "IRC nickname")
	ircUser          = flag.String("irc-user", "sh0rk", "IRC username")
	ircReal          = flag.String("irc-real", "Friendly sh0rk Mara", "IRC realname")
	ircServer        = flag.String("irc-server", "chrysalis:6667", "IRC server to connect to")
	ircSASLUsername  = flag.String("irc-sasl-username", "", "SASL username")
	ircSASLPassword  = flag.String("irc-sasl-password", "", "SASL password")
	ircRevoltChannel = flag.String("irc-revolt-channel", "", "channel to copy #xeserv messages to")
)

func (mr *MaraRevolt) IRCBot(ctx context.Context) {
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
			t := time.NewTicker(250 * time.Millisecond)
			defer t.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-mr.ircmsgs:
					<-t.C
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
			if _, err := mr.db.ExecContext(ctx, `INSERT INTO irc_messages(nick, user, host, channel, content, tags) VALUES (?, ?, ?, ?, ?, ?)`, e.Nick, e.User, e.Host, e.Arguments[0], e.Message(), ""); err != nil {
				ln.Error(ctx, err)
			}

			if e.Arguments[0] == "#xeserv" {
				sendMsg := &revolt.SendMessage{
					Masquerade: &revolt.Masquerade{
						Name:      e.Nick,
						AvatarURL: "https://cdn.xeiaso.net/avatar/" + internal.Hash(e.User, e.Host),
					},
					Content: e.Message(),
				}

				if _, err := mr.cli.ChannelSendMessage(ctx, *ircRevoltChannel, sendMsg); err != nil {
					ln.Error(ctx, err)
					return
				}
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
