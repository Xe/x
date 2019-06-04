package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"within.website/x/internal"
	"github.com/bwmarrin/discordgo"
	"github.com/garyburd/redigo/redis"
)

// Variables used for command line parameters
var (
	token    = flag.String("discord-token", "", "Discord bot token")
	redisurl = flag.String("redis-url", "", "Redis server URL")
)

func init() {
	internal.HandleStartup()
}

func main() {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + *token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	p, err := NewPool(*redisurl)
	if err != nil {
		log.Fatal(err)
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate(p))

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(p *redis.Pool) func(*discordgo.Session, *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore all messages created by the bot itself
		// This isn't required in this specific example but it's a good practice.
		if m.Author.ID == s.State.User.ID {
			return
		}

		if strings.Contains(m.Content, "🅱️") {
			go s.MessageReactionAdd(m.ChannelID, m.Message.ID, "🅱️")
		}
		if strings.Contains(m.Content, "shit") {
			go s.MessageReactionAdd(m.ChannelID, m.Message.ID, "💩")
		}
		if strings.Contains(m.Content, "🔥") {
			go s.MessageReactionAdd(m.ChannelID, m.Message.ID, "🔥")
		}
		if strings.Contains(strings.ToLower(m.Content), "lit") {
			go s.MessageReactionAdd(m.ChannelID, m.Message.ID, "🔥")
		}

		conn := p.Get()
		defer conn.Close()

		_, err := conn.Do("INCR", "counts:"+m.Author.ID)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(m.Author.ID, m.ChannelID, m.ContentWithMentionsReplaced())
	}
}

// NewPool creates a new redis pool with default "sane" settings.
func NewPool(uri string) (*redis.Pool, error) {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 4 * time.Minute,
		Dial:        func() (redis.Conn, error) { return redis.DialURL(uri) },
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}, nil
}
