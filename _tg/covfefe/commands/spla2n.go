package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/syfaro/finch"
	"gopkg.in/telegram-bot-api.v4"
)

func init() {
	finch.RegisterCommand(&splattusCommand{})
}

type splattusCommand struct {
	finch.CommandBase
}

func (cmd *splattusCommand) Help() finch.Help {
	return finch.Help{
		Name:        "Splattus",
		Description: "Displays splatoon 2 status",
		Example:     "/splattus@@",
		Botfather: [][]string{
			[]string{"splattus", "Splatoon 2 map rotations"},
		},
	}
}

func (cmd *splattusCommand) ShouldExecute(message tgbotapi.Message) bool {
	return finch.SimpleCommand("splattus", message.Text)
}

func (cmd *splattusCommand) Execute(message tgbotapi.Message) error {
	resp, err := http.Get("https://splatoon.ink/schedule2")
	if err != nil {
		panic(err)
	}

	st := &splattus{}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	json.Unmarshal(body, st)

	modeInfo := []string{st.Modes.Regular[1].String(), st.Modes.Gachi[1].String(), st.Modes.League[1].String()}
	text := strings.Join(modeInfo, "\n")

	msg := tgbotapi.NewMessage(message.Chat.ID, text)

	return cmd.Finch.SendMessage(msg)
}

type splatoonMode struct {
	StartTime int64            `json:"startTime"`
	EndTime   int64            `json:"endTime"`
	Maps      []string         `json:"maps"`
	Rule      splatoonRule     `json:"rule"`
	Mode      splatoonGameMode `json:"mode"`
}

func (sm splatoonMode) String() string {
	maps := strings.Join(sm.Maps, ", ")
	end := time.Unix(sm.EndTime, 0)
	now := time.Now()
	diff := end.Sub(now)

	return fmt.Sprintf("%s:\nRotation ends at %s (in %s)\nMaps: %s\nRule: %s\n", sm.Mode, end.Format(time.RFC3339), diff, maps, sm.Rule)
}

type splatoonGameMode struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

func (sgm splatoonGameMode) String() string {
	return sgm.Name
}

type splatoonRule struct {
	Key           string `json:"key"`
	MultilineName string `json:"multiline_name"`
	Name          string `json:"name"`
}

func (sr splatoonRule) String() string {
	return sr.Name
}

type splattus struct {
	UpdateTime int64 `json:"updateTime"`
	Modes      struct {
		League  []splatoonMode `json:"league"`
		Regular []splatoonMode `json:"regular"`
		Gachi   []splatoonMode `json:"gachi"`
	} `json:"modes"`
}
