package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func hipster(s *discordgo.Session, m *discordgo.Message, parv []string) error {
	msg, err := getHipsterText()
	if err != nil {
		return err
	}

	s.ChannelMessageSend(m.ChannelID, msg)
	return nil
}

func getHipsterText() (string, error) {
	resp, err := http.Get("http://hipsterjesus.com/api/?type=hipster-centric&html=false") // paras parameter no longer respected, need to re-implement locally
	if err != nil {
		return "", err
	}

	textStruct := &struct {
		Text string `json:"text"`
	}{}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	json.Unmarshal(body, textStruct)

	text := strings.Split(textStruct.Text, ". ")[0]
	textSlice := strings.Split(text, " ") // Separate each word into an array
	truncatedText := ""

	wordCount := 5 // change this to adjust word count

	for i := 0; i < wordCount; i++ {
		truncatedText += textSlice[i] + " "
	}

	return truncatedText, nil
}
