package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/go-shiori/go-readability"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"within.website/x/internal"
	"within.website/x/web/marginalia"
	"within.website/x/web/openai/chatgpt"
)

var (
	marginaliaToken = flag.String("marginalia-token", "", "Token for Marginalia internet search")
	openAIToken     = flag.String("openai-token", "", "OpenAI token")
	openAIModel     = flag.String("openai-model", "gpt-3.5-turbo-16k-0613", "OpenAI model to use")
	telegramAdmin   = flag.Int64("telegram-admin", 0, "Telegram bot admin")
	telegramToken   = flag.String("telegram-token", "", "Telegram bot token")
)

func main() {
	internal.HandleStartup()

	mc := marginalia.New(*marginaliaToken, nil)

	cGPT := chatgpt.NewClient(*openAIToken)

	// Note: Please keep in mind that default logger may expose sensitive information,
	// use in development only
	bot, err := telego.NewBot(*telegramToken)
	if err != nil {
		log.Fatal(err)
	}

	// Get updates channel
	updates, err := bot.UpdatesViaLongPolling(nil)
	if err != nil {
		log.Fatal(err)
	}

	// Create bot handler and specify from where to get updates
	bh, err := th.NewBotHandler(bot, updates)
	if err != nil {
		log.Fatal(err)
	}

	// Stop handling updates
	defer bh.Stop()

	// Stop getting updates
	defer bot.StopLongPolling()

	// Register new handler with match on command `/start`
	bh.Handle(func(bot *telego.Bot, update telego.Update) {
		// Send message
		if _, err := bot.SendMessage(tu.Message(
			tu.ID(update.Message.Chat.ID),
			fmt.Sprintf("Hello %s!", update.Message.From.FirstName),
		)); err != nil {
			slog.Error("can't send message", "err", err)
		}
	}, th.CommandEqual("start"))

	bh.Handle(func(bot *telego.Bot, update telego.Update) {
		if update.Message.From.ID != *telegramAdmin {
			bot.SendMessage(tu.Message(
				tu.ID(update.Message.Chat.ID),
				"unknown command",
			))
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		q := strings.Join(strings.Split(update.Message.Text, " ")[1:], " ")

		lg := slog.Default().With(
			"telegram_requestor", update.Message.From.ID,
			"telegram_requestor_name", fmt.Sprintf("%s %s", update.Message.From.FirstName, update.Message.From.LastName),
			"search_query", q,
		)
		results, err := mc.Search(ctx, &marginalia.Request{
			Query: q,
			Count: aws.Int(5),
		})
		if err != nil {
			lg.Error("can't search", "err", err)
			bot.SendMessage(tu.Message(
				tu.ID(update.Message.Chat.ID),
				fmt.Sprintf("Error: %v", err),
			))
			return
		}

		var sb strings.Builder

		fmt.Fprintf(&sb, "License: %s\n\n", results.License)

		for _, result := range results.Results {
			fmt.Fprintf(&sb, "**%s** (%s):\n", result.Title, result.URL)

			lg.Info("resolving article", "result_title", result.Title, "result_url", result.URL)

			article, err := readability.FromURL(result.URL, 30*time.Second)
			if err != nil {
				fmt.Fprintf(&sb, "Can't parse article: %v", err)
				continue
			}

			resp, err := cGPT.Complete(ctx, chatgpt.Request{
				Model: *openAIModel,
				Messages: []chatgpt.Message{
					{
						Role:    "system",
						Content: "You are a programmer's research assistant, engaging users in thoughtful discussions on a wide range of topics, from ethics and metaphysics to programming and architectural design. Offer insights into the works of various philosophers, well-known programmers, their theories, and ideas. Encourage users to think critically and reflect on the nature of existence, knowledge, and values.",
					},
					{
						Role:    "user",
						Content: "Can you summarize this article by " + article.Byline + " in 3 sentences or less?\n\n" + article.TextContent,
					},
				},
			})
			if err != nil {
				lg.Error("can't summarize article", "err", err)
			}

			fmt.Fprintf(&sb, "%s\n\n", resp.Choices[0].Message.Content)
		}

		msg := tu.Message(tu.ID(update.Message.Chat.ID), sb.String())
		msg.ParseMode = telego.ModeMarkdown

		if _, err := bot.SendMessage(msg); err != nil {
			lg.Error("can't send final message", "err", err)
			return
		}

		lg.Info("query successful")
	}, th.CommandEqual("search"))

	// Register new handler with match on any command
	// Handlers will match only once and in order of registration,
	// so this handler will be called on any command except `/start` command
	bh.Handle(func(bot *telego.Bot, update telego.Update) {
		// Send message
		_, _ = bot.SendMessage(tu.Message(
			tu.ID(update.Message.Chat.ID),
			"Unknown command, use /start",
		))
	}, th.AnyCommand())

	// Start handling updates
	bh.Start()
}
