package commands

import (
	"fmt"
	"github.com/syfaro/finch"
	"gopkg.in/telegram-bot-api.v4"
)

func init() {
	finch.RegisterCommand(&infoCommand{})
}

type infoCommand struct {
	finch.CommandBase
}

func (cmd *infoCommand) Help() finch.Help {
	return finch.Help{
		Name:        "Info",
		Description: "Displays information about the currently requesting user",
		Example:     "/info@@",
		Botfather: [][]string{
			[]string{"info", "Information about the current user"},
		},
	}
}

func (cmd *infoCommand) ShouldExecute(message tgbotapi.Message) bool {
	return finch.SimpleCommand("info", message.Text)
}

func (cmd *infoCommand) Execute(message tgbotapi.Message) error {
	text := fmt.Sprintf("Your ID: %d\nChat ID: %d", message.From.ID, message.Chat.ID)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ReplyToMessageID = message.MessageID

	return cmd.Finch.SendMessage(msg)
}
