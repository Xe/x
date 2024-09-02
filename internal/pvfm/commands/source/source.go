package source

import "github.com/bwmarrin/discordgo"

func Source(s *discordgo.Session, m *discordgo.Message, parv []string) error {
	s.ChannelMessageSend(m.ChannelID, "Source code: https://github.com/PonyvilleFM/aura")
	return nil
}
