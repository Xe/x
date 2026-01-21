package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Marcel-ICMC/graw"
	"github.com/Marcel-ICMC/graw/reddit"
	"within.website/x/internal"
	"within.website/x/internal/yeet"
	"within.website/x/store"
	"within.website/x/web/discordwebhook"
)

var (
	discordWebhookURL = flag.String("discord-webhook-url", "", "The Discord webhook url sapientwindex will post to")
	redditUsername    = flag.String("reddit-username", "", "Your reddit username")
	subreddits        = flag.String("subreddits", "", "subreddits to scan (separate multiple by commas)")
	scanDuration      = flag.Duration("scan-duration", 30*time.Second, "scan frequency")
	stateDir          = flag.String("state-dir", "", "state directory")
)

func main() {
	internal.HandleStartup()

	if *discordWebhookURL == "" {
		slog.Error("you must set the discord webhook URL to use this bot")
		os.Exit(2)
	}

	if *redditUsername == "" {
		slog.Error("you must set your reddit username to use this bot")
		os.Exit(2)
	}

	if *subreddits == "" {
		slog.Error("you must set the subreddit list to use this bot")
		os.Exit(2)
	}

	st, err := store.NewCAS(*stateDir)
	if err != nil {
		slog.Error("can't open state directory", "dir", *stateDir, "err", err)
		os.Exit(2)
	}
	defer store.Close(st)

	redditUserAgent := fmt.Sprintf("graw:within.website/x/cmd/sapientwindex:%s by /u/%s", yeet.DateTag, *redditUsername)

	slog.Info("starting up", "subreddits", *subreddits, "scan_duration", (*scanDuration).String())

	handle, err := reddit.NewScript(redditUserAgent, *scanDuration)
	if err != nil {
		log.Fatal(err)
	}
	announce := &announcer{
		seenPosts: &store.JSON[void]{
			Underlying: st,
			Prefix:     "posts",
		},
	}

	scriptCfg := graw.Config{
		Subreddits: strings.Split(*subreddits, ","),
		Logger:     slog.NewLogLogger(slog.Default().Handler(), slog.LevelInfo),
	}

	for {
		stop, wait, err := graw.Scan(announce, handle, scriptCfg)
		if err != nil {
			log.Fatal(err)
		}

		wait()
		stop()

		slog.Info("connection lost, sleeping and retrying")
		time.Sleep(5 * time.Second)
	}
}

type void struct{}
type announcer struct {
	seenPosts *store.JSON[void]
}

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
	if a.seenPosts.Exists(context.Background(), post.ID) == nil {
		slog.Debug("already seen post, ignoring", "id", post.ID)
		return nil
	}

	if len(post.SelfText) > 1000 {
		post.SelfText = post.SelfText[:1000] + "... [truncated]"
	}

	wh := discordwebhook.Webhook{
		Username:  post.Author,
		Content:   fmt.Sprintf("## %s\n> %s\n<https://reddit.com%s>", post.Title, addMemeArrow(post.SelfText), post.Permalink),
		AvatarURL: fmt.Sprintf("https://cdn.xeiaso.net/avatar/%s", internal.Hash(post.Author, *redditUsername)),
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

	if err := a.seenPosts.Set(context.Background(), post.ID, void{}); err != nil {
		slog.Error("seen post error", "err", err)
		return nil
	}

	return nil
}
