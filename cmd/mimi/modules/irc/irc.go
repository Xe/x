package irc

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log/slog"

	"github.com/bwmarrin/discordgo"
	ircevent "github.com/thoj/go-ircevent"
)

var (
	discordAnnounceChannel = flag.String("discord-announce-channel", "", "Discord channel to announce in")

	ircServer   = flag.String("irc-server", "irc.libera.chat:6697", "IRC server to connect to")
	ircUsername = flag.String("irc-username", "[Mara]", "IRC username")
	ircIdent    = flag.String("irc-ident", "mara", "IRC ident")
	ircPassword = flag.String("irc-password", "", "IRC password")
	ircChannel  = flag.String("irc-channel", "#mimi", "IRC channel to join")
)

type Module struct {
	conn *ircevent.Connection
	dg   *discordgo.Session
}

func New(ctx context.Context, dg *discordgo.Session) (*Module, error) {
	conn := ircevent.IRC(*ircUsername, *ircIdent)
	conn.UseTLS = true
	conn.UseSASL = false
	conn.SASLLogin = *ircUsername
	conn.SASLPassword = *ircPassword
	conn.SASLMech = "PLAIN"

	conn.TLSConfig = &tls.Config{
		ServerName: "irc.libera.chat",
	}

	conn.AddCallback("001", func(e *ircevent.Event) {
		conn.Privmsgf("NickServ", "IDENTIFY %s %s", *ircUsername, *ircPassword)
	})

	conn.AddCallback("900", func(e *ircevent.Event) {
		conn.Join(*ircChannel)
		slog.Debug("joined channel", "channel", *ircChannel)
	})
	if err := conn.Connect(*ircServer); err != nil {
		return nil, fmt.Errorf("irc: error connecting to IRC server: %w", err)
	}

	go func() {
		<-ctx.Done()
		conn.Quit()
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-conn.Error:
				slog.Error("error from IRC server", "err", err)
			}
		}
	}()

	return &Module{
		conn: conn,
		dg:   dg,
	}, nil
}
