package commands

import (
	"math/rand"
	"strings"
	"time"

	"github.com/justinian/dice"
	"github.com/syfaro/finch"
	"gopkg.in/telegram-bot-api.v4"
)

func init() {
	finch.RegisterCommand(&diceCommand{})
	rand.Seed(time.Now().Unix())
}

type diceCommand struct {
	finch.CommandBase
}

func (cmd *diceCommand) Help() finch.Help {
	return finch.Help{
		Name:        "Dice",
		Description: "Standard: xdy[[k|d][h|l]z][+/-c] - rolls and sums x y-sided dice, keeping or dropping the lowest or highest z dice and optionally adding or subtracting c.",
		Example:     "/dice@@ 4d6kh3+4",
		Botfather: [][]string{
			[]string{"dice", "4d20, etc"},
		},
	}
}

func (cmd *diceCommand) ShouldExecute(message tgbotapi.Message) bool {
	return finch.SimpleCommand("dice", message.Text)
}

func (cmd *diceCommand) Execute(message tgbotapi.Message) error {
	parv := strings.Fields(message.CommandArguments())

	if len(parv) != 1 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Try something like 4d20")
		return cmd.Finch.SendMessage(msg)
	}

	text, _, err := dice.Roll(parv[0])
	if err != nil {
		return err
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, text.String())

	return cmd.Finch.SendMessage(msg)
}
