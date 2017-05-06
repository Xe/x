package main

import (
	"log"

	"github.com/Xe/ln"
	"github.com/caarlos0/env"
	_ "github.com/joho/godotenv/autoload"
	"gopkg.in/telegram-bot-api.v4"
)

type config struct {
	RedditUsername string `env:"REDDIT_USERNAME,required"`
	RedditPassword string `env:"REDDIT_PASSWORD.required"`

	TelegramToken string `env:"TELEGRAM_TOKEN,required"`
	TelegramAdmin string `env:"TELEGRAM_ADMIN,required"`
}

func main() {
	var cfg config
	err := env.Parse(&cfg)
	if err != nil {
		ln.Fatal(ln.F{"err": err, "action": "env.Parse"})
	}

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		ln.Fatal(ln.F{"err": err, "action": "tgbotapi.NewBotAPI"})
	}

	bot.Debug = true

	ln.Log(ln.F{"action": "telegram_active", "username": bot.Self.UserName})

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		msg.ReplyToMessageID = update.Message.MessageID

		bot.Send(msg)
	}
}
