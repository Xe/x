package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"text/template"
	"time"

	"github.com/Marcel-ICMC/graw"
	"github.com/Marcel-ICMC/graw/reddit"
	"within.website/x/internal"
)

var (
	redditUsername  = flag.String("reddit-username", "", "reddit username")
	redditPassword  = flag.String("reddit-password", "", "reddit password")
	redditAppID     = flag.String("reddit-app-id", "", "reddit app id")
	redditAppSecret = flag.String("reddit-app-secret", "", "reddit app secret")
	subreddit       = flag.String("subreddit", "shadowh511", "subreddit to post to")
	scanDuration    = flag.Duration("scan-duration", 30*time.Second, "how long to scan for")

	//go:embed prompts/*.txt
	prompts embed.FS
)

func main() {
	internal.HandleStartup()

	slog.Info("starting up", "username", *redditUsername, "subreddit", *subreddit, "scan_duration", (*scanDuration).String())

	cfg := reddit.BotConfig{
		Agent: "graw:sapientwindex:0.0.1 by /u/shadowh511",
		App: reddit.App{
			ID:       *redditAppID,
			Secret:   *redditAppSecret,
			Username: *redditUsername,
			Password: *redditPassword,
		},
	}

	bot, err := reddit.NewBot(cfg)
	if err != nil {
		log.Fatal(err)
	}

	handle, err := reddit.NewScript(cfg.Agent, *scanDuration)
	if err != nil {
		log.Fatal(err)
	}
	announce := &announcer{bot: bot}

	scriptCfg := graw.Config{Subreddits: []string{*subreddit, "shadowh511"}}

	stop, wait, err := graw.Scan(announce, handle, scriptCfg)
	if err != nil {
		log.Fatal(err)
	}

	defer stop()

	wait()
}

type announcer struct {
	bot reddit.Bot
}

func makePrompt(kind, title, body string) (string, error) {
	data, err := prompts.ReadFile("prompts/" + kind + ".txt")
	if err != nil {
		return "", fmt.Errorf("read prompt: %w", err)
	}

	tmpl, err := template.New("prompt").Parse(string(data))
	if err != nil {
		return "", fmt.Errorf("parse prompts: %w", err)
	}

	var prompt bytes.Buffer
	err = tmpl.Execute(&prompt, struct {
		Title string
		Body  string
	}{
		Title: title,
		Body:  body,
	})
	if err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return prompt.String(), nil
}

func (a *announcer) Post(post *reddit.Post) error {
	if post.LinkFlairText == "Personal" {
		return nil
	}

	slog.Info("got post", "title", post.Title, "body", post.SelfText)

	prompt, err := makePrompt("moderation", post.Title, post.SelfText)
	if err != nil {
		slog.Error("make prompt", "err", err)
		return nil
	}

	opts := &LLAMAOpts{
		Temperature:   0.8,
		TopK:          40,
		TopP:          0.9,
		Stream:        false,
		Prompt:        prompt,
		RepeatPenalty: 1.15,
		RepeatLastN:   512,
		Mirostat:      2,
		NPredict:      2048,
	}

	resp, err := Predict(opts)
	if err != nil {
		slog.Error("predict", "err", err)
		return nil
	}

	if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(resp.Content)), "YES") {
		slog.Info("not a question, skipping", "title", post.Title, "body", post.SelfText, "response", resp.Content)
		return nil
	}

	prompt, err = makePrompt("helper", post.Title, post.SelfText)
	if err != nil {
		slog.Error("make prompt", "err", err)
		return nil
	}

	opts.Prompt = prompt

	resp, err = Predict(opts)
	if err != nil {
		slog.Error("predict", "err", err)
		return nil
	}

	body := massageAnswer(resp.Content)

	if err := a.bot.Reply(post.Name, body); err != nil {
		slog.Error("reply", "err", err)
		return nil
	}

	return nil
}

func massageAnswer(answer string) string {
	answer = strings.TrimSpace(answer)
	answer = strings.TrimPrefix(answer, "ANSWER: ")
	return answer
}
