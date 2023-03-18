package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/bwmarrin/discordgo"
	"within.website/x/internal"
	textgen "within.website/x/internal/textgeneration"
)

var (
	discordToken = flag.String("discord-token", "", "Discord bot token")
	guildID      = flag.String("discord-guild", "", "Test guild ID. If not passed - bot registers commands globally")
	channelID    = flag.String("discord-channel", "", "Channel to restrict responses to")

	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "textgen",
			Description: "Generate text with LLaMA",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "prompt",
					Description: "Prompt for LLaMA",
					Type:        discordgo.ApplicationCommandOptionString,
				},
			},
		},
	}
)

func main() {
	internal.HandleStartup()

	var s *discordgo.Session

	s, err := discordgo.New("Bot " + *discordToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.ApplicationCommandData().Name == "textgen" {
			handleTextgen(s, i)
		}
	})

	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	err = s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, *guildID, v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop

	log.Println("Removing commands...")
	for _, v := range registeredCommands {
		err := s.ApplicationCommandDelete(s.State.User.ID, *guildID, v.ID)
		if err != nil {
			log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
		}
	}

	log.Println("Gracefully shutting down.")
}

func handleTextgen(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if i.ChannelID != *channelID {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You must use this in <#" + *channelID + ">.",
			},
		})
		return
	}

	cr := new(textgen.ChatRequest)
	cr.ApplyPreset("Default")
	cr.MaxNewTokens = 64
	cr.DoSample = true
	cr.EarlyStopping = true

	prompt, ok := i.ApplicationCommandData().Options[0].Value.(string)
	if !ok {
		panic("what.")
	}

	cr.Input = prompt

	resp, err := textgen.Generate(ctx, cr)
	if err != nil {
		log.Printf("error generating response: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("error generating response: %v", err),
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: resp.Data[0],
		},
	})
}
