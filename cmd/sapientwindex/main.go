package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Marcel-ICMC/graw"
	"github.com/Marcel-ICMC/graw/reddit"
	"within.website/x/internal"
	"within.website/x/web/discordwebhook"
)

var (
	discordWebhookURL = flag.String("discord-webhook-url", "", "discord webhook url")
	redditUserAgent   = flag.String("reddit-user-agent", "graw:windex:0.0.1 by /u/shadowh511", "reddit user agent")
	subreddit         = flag.String("subreddit", "tulpas", "subreddit to post to")
	scanDuration      = flag.Duration("scan-duration", 30*time.Second, "scan frequency")
)

func main() {
	internal.HandleStartup()

	slog.Info("starting up", "subreddit", *subreddit, "scan_duration", (*scanDuration).String())

	handle, err := reddit.NewScript(*redditUserAgent, *scanDuration)
	if err != nil {
		log.Fatal(err)
	}
	announce := &announcer{}

	scriptCfg := graw.Config{
		Subreddits: []string{*subreddit},
		Logger:     slog.NewLogLogger(slog.Default().Handler(), slog.LevelInfo),
	}

	stop, wait, err := graw.Scan(announce, handle, scriptCfg)
	if err != nil {
		log.Fatal(err)
	}

	defer stop()

	wait()
}

type announcer struct{}

func addMemeArrow(str string) string {
	var result strings.Builder
	for _, char := range str {
		if char == '\n' {
			result.WriteString("\n> ")
		} else {
			result.WriteRune(char)
		}
	}
	return result.String()
}

func (a announcer) Post(post *reddit.Post) error {
	if len(post.SelfText) > 1000 {
		post.SelfText = post.SelfText[:1000] + " [truncated]"
	}

	wh := discordwebhook.Webhook{
		Username:  post.Author,
		Content:   fmt.Sprintf("## %s\n> %s\n<https://reddit.com%s>", post.Title, addMemeArrow(post.SelfText), post.Permalink),
		AvatarURL: fmt.Sprintf("https://cdn.xeiaso.net/avatar/%s", internal.Hash(post.Author, *redditUserAgent)),
		AllowedMentions: map[string][]string{
			"parse": {},
		},
	}

	if !post.IsSelf {
		wh.Content = fmt.Sprintf("## %s\n%s\n\n<https://reddit.com%s>", post.Title, post.URL, post.Permalink)
	}

	slog.Debug("got post", "title", post.Title)

	req := discordwebhook.Send(*discordWebhookURL, wh)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("discord webhook error", "err", err)
		return nil
	}

	if err := discordwebhook.Validate(resp); err != nil {
		slog.Error("discord webhook error", "err", err)
		return nil
	}

	return nil
}
