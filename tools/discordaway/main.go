package main

import (
	"bytes"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/namsral/flag"
)

var (
	cfg      = flag.String("config", "/home/xena/.local/share/within/discordaway.cfg", "configuration file")
	username = flag.String("username", "", "Discord username to use")
	password = flag.String("password", "", "Discord password to use")
)

func main() {
	flag.Parse()

	dg, err := discordgo.New(*username, *password)
	if err != nil {
		log.Fatal(err)
	}

	dg.AddHandler(messageCreate)

	err = dg.Open()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("monitoring tmux status...")
	t := time.NewTicker(300 * time.Second)

	ok, err := isTmuxAttached()
	log.Println(ok, err)

	for {
		select {
		case <-t.C:
			at, err := isTmuxAttached()
			if err != nil {
				log.Println(err)
				return
			}

			if at {
				log.Println("Cadey is away, marking as away on Discord")
				dg.UpdateStatus(600, "around with reality for some reason")
			} else {
				log.Println("Cadey is back!!!")
				dg.UpdateStatus(0, "")
			}
		}
	}
}

func isTmuxAttached() (bool, error) {
	cmd := exec.Command("/usr/bin/tmux", "ls", "-F", "#{?session_attached,attached,not attached}")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return bytes.HasPrefix(output, []byte("attached")), nil
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == "72838115944828928" {
		content := m.ContentWithMentionsReplaced()

		if strings.HasPrefix(content, "TODO: ") {
			todoBody := strings.SplitN(content, "TODO: ", 2)[1]

			log.Printf("todo added: %s", todoBody)
			todoFields := strings.Fields(todoBody)

			cmd := exec.Command("/home/xena/go/bin/todo", append([]string{"add"}, todoFields...)...)
			err := cmd.Start()
			if err != nil {
				log.Println(err)
				return
			}

			err = cmd.Wait()
			if err != nil {
				log.Println(err)
			}
		}
	}
}
