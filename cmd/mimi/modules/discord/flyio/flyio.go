package flyio

import (
	"github.com/bwmarrin/discordgo"
)

type Module struct {
	sess *discordgo.Session
}

func New() *Module {
	return &Module{}
}

func (m *Module) Register(s *discordgo.Session) {
	m.sess = s
}
