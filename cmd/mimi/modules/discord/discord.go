package discord

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

var (
	discordToken = flag.String("discord-token", "", "discord token")
)

type Module struct {
	dg *discordgo.Session
}

func New(ctx context.Context) (*Module, error) {
	dg, err := discordgo.New("Bot " + *discordToken)
	if err != nil {
		return nil, fmt.Errorf("discord: error creating discord session: %w", err)
	}

	go func() {
		<-ctx.Done()
		dg.Close()
	}()

	return &Module{
		dg: dg,
	}, nil
}

func (m *Module) Open() {
	go func() {
		if err := m.dg.Open(); err != nil {
			log.Fatal(err)
		}
	}()
}

func (m *Module) Register(dm DiscordModule) {
	dm.Register(m.dg)
}

type DiscordModule interface {
	Register(s *discordgo.Session)
}
