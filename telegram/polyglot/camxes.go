package main

import (
	"bytes"
	"log"
	"strings"

	"github.com/Syfaro/finch"
	"github.com/Xe/johaus/parser"
	_ "github.com/Xe/johaus/parser/camxes"
	"github.com/Xe/johaus/pretty"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func init() {
	finch.RegisterCommand(&camxesCommand{})
}

type camxesCommand struct {
	finch.CommandBase
}

const dialect = "camxes"

func (c camxesCommand) Help() finch.Help {
	return finch.Help{
		Name:        "camxes",
		Description: "Use the camxes parser for lojban",
		Example:     "/camxes@@ .i lo mlatu cu pinxe lo ladru",
		Botfather: [][]string{
			[]string{"camxes", "parse lojban"},
		},
	}
}

func (c camxesCommand) ShouldExecute(message tgbotapi.Message) bool {
	return finch.SimpleCommand("camxes", message.Text)
}

func (c camxesCommand) Execute(message tgbotapi.Message) error {
	msg := strings.Join(strings.Split(message.Text, " ")[1:], " ")
	log.Printf("msg: %s", msg)
	tree, err := parser.Parse(dialect, msg)
	if err != nil {
		return err
	}

	parser.RemoveMorphology(tree)
	parser.AddElidedTerminators(tree)
	parser.RemoveSpace(tree)
	parser.CollapseLists(tree)

	buf := bytes.NewBuffer(nil)
	pretty.Braces(buf, tree)

	log.Println(buf.String())

	tmsg := tgbotapi.NewMessage(message.Chat.ID, buf.String())
	tmsg.ReplyToMessageID = message.MessageID
	return c.CommandBase.Finch.SendMessage(tmsg)
}
