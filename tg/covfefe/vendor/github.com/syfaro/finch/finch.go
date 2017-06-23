// Package finch is a framework for Telegram Bots.
package finch

import (
	"encoding/json"
	"github.com/getsentry/raven-go"
	"gopkg.in/telegram-bot-api.v4"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

// Config is a type used for storing configuration information.
type Config map[string]interface{}

var sentryEnabled bool = false

// LoadConfig loads the saved config, if it exists.
//
// It looks for a FINCH_CONFIG environmental variable,
// before falling back to a file name config.json.
func LoadConfig() (*Config, error) {
	fileName := os.Getenv("FINCH_CONFIG")
	if fileName == "" {
		fileName = "config.json"
	}

	f, err := ioutil.ReadFile(fileName)
	if err != nil {
		return &Config{}, nil
	}

	var cfg Config
	json.Unmarshal(f, &cfg)

	return &cfg, nil
}

// Save saves the current Config struct.
//
// It uses the same file as LoadConfig.
func (c *Config) Save() error {
	b, err := json.Marshal(c)
	if err != nil {
		if sentryEnabled {
			raven.CaptureErrorAndWait(err, nil)
		}

		return err
	}

	fileName := os.Getenv("FINCH_CONFIG")
	if fileName == "" {
		fileName = "config.json"
	}

	return ioutil.WriteFile(fileName, b, 0600)
}

// Finch is a Telegram Bot, including API, Config, and Commands.
type Finch struct {
	API      *tgbotapi.BotAPI
	Config   Config
	Commands []*CommandState
	Inline   InlineCommand
}

// NewFinch returns a new Finch instance, with Telegram API setup.
func NewFinch(token string) *Finch {
	return NewFinchWithClient(token, &http.Client{})
}

// NewFinchWithClient returns a new Finch instance,
// using a different net/http Client.
func NewFinchWithClient(token string, client *http.Client) *Finch {
	bot := &Finch{}

	api, err := tgbotapi.NewBotAPIWithClient(token, client)
	if err != nil {
		panic(err)
	}

	bot.API = api
	bot.Commands = commands
	bot.Inline = inline

	c, _ := LoadConfig()
	bot.Config = *c

	return bot
}

// Start initializes commands, and starts listening for messages.
func (f *Finch) Start() {
	if v, ok := f.Config["sentry_dsn"]; ok {
		sentryEnabled = true
		raven.SetDSN(v.(string))
	}

	f.commandInit()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 86400

	updates, err := f.API.GetUpdatesChan(u)
	if err != nil {
		if sentryEnabled {
			raven.CaptureErrorAndWait(err, nil)
		}

		log.Fatal(err)
	}

	for update := range updates {
		go f.commandRouter(update)
	}
}

// StartWebhook initializes commands,
// then registers a webhook for the bot to listen on
func (f *Finch) StartWebhook(endpoint string) {
	f.commandInit()

	f.API.ListenForWebhook(endpoint)
}

// SendMessage sends a message with various changes, and does not return the Message.
//
// At some point, this may do more handling as needed.
func (f *Finch) SendMessage(message tgbotapi.MessageConfig) error {
	message.Text = strings.Replace(message.Text, "@@", "@"+f.API.Self.UserName, -1)

	_, err := f.API.Send(message)
	if err != nil && sentryEnabled {
		raven.CaptureError(err, nil)
	}
	return err
}

// QuickReply quickly sends a message as a reply.
func (f *Finch) QuickReply(message tgbotapi.Message, text string) error {
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ReplyToMessageID = message.MessageID

	return f.SendMessage(msg)
}
