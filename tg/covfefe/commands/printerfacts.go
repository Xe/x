package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/syfaro/finch"
	"gopkg.in/telegram-bot-api.v4"
)

func init() {
	finch.RegisterCommand(&printerfactCommand{})
}

type printerfactCommand struct {
	finch.CommandBase
}

func (cmd *printerfactCommand) Help() finch.Help {
	return finch.Help{
		Name:        "Printerfact",
		Description: "Displays printerfactrmation about the currently requesting user",
		Example:     "/printerfact@@",
		Botfather: [][]string{
			[]string{"printerfact", "Printerfactrmation about the current user"},
		},
	}
}

func (cmd *printerfactCommand) ShouldExecute(message tgbotapi.Message) bool {
	return finch.SimpleCommand("printerfact", message.Text)
}

func (cmd *printerfactCommand) Execute(message tgbotapi.Message) error {
	resp, err := http.Get("http://xena.stdlib.com/printerfacts")
	if err != nil {
		panic(err)
	}

	factStruct := &struct {
		Facts []string `json:"facts"`
	}{}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	json.Unmarshal(body, factStruct)

	text := fmt.Sprintf("%s", factStruct.Facts[0])

	msg := tgbotapi.NewMessage(message.Chat.ID, text)

	return cmd.Finch.SendMessage(msg)
}
