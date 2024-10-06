package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	gpt3encoder "github.com/samber/go-gpt-3-encoder"
	"jaytaylor.com/html2text"
	"within.website/x/internal"
	"within.website/x/llm"
)

var (
	cacheFolder = flag.String("cache-folder", "./var/hn", "Folder to cache items in")
	hnUser      = flag.String("hn-user", "xena", "Hacker News user to scrape")
	scrapeDelay = flag.Duration("scrape-delay", 50*time.Millisecond, "Delay between scraping items")
)

const systemMessage = `You are a commenter on the website "Hacker News". If asked for your name, you will respond with "Mimi". You should be friendly unless people are being mean to you, then you can be mean back.`

func main() {
	internal.HandleStartup()
	ctx, cancel := ControlCContext()
	defer cancel()

	slog.Debug("starting hnscrape", "scrapeDelay", scrapeDelay.String(), "hnUser", *hnUser)

	hn := NewHNClient(*scrapeDelay)

	if *cacheFolder != "" {
		slog.Debug("caching items to", "cacheFolder", *cacheFolder)
		os.MkdirAll(*cacheFolder, 0755)
		os.MkdirAll(filepath.Join(*cacheFolder, "items"), 0755)
		os.MkdirAll(filepath.Join(*cacheFolder, "indices"), 0755)
		os.MkdirAll(filepath.Join(*cacheFolder, "conversations"), 0755)
		hn = hn.WithCacheFolder(*cacheFolder)
	}

	u, err := hn.GetUser(ctx, *hnUser)
	if err != nil {
		slog.Error("failed to get user", "err", err, "user", *hnUser)
		os.Exit(1)
	}

	slog.Debug("got user", "user", u.Created.String(), "karma", u.Karma, "submitted", len(u.Submitted))

	reverseIntSlice(u.Submitted)

	conversations := map[string][]int{}

	for _, itemID := range u.Submitted {
		item, err := hn.GetItem(ctx, itemID)
		if err != nil {
			slog.Error("failed to get item", "err", err, "itemID", itemID)
			os.Exit(1)
		}

		if item.Type != "comment" {
			continue
		}

		if item.Parent == nil {
			continue
		}

		parent, err := hn.GetItem(ctx, *item.Parent)
		if err != nil {
			slog.Error("failed to get parent", "err", err, "itemID", item.ID)
			continue
		}
		_ = parent

		pathToRoot, err := hn.PathToRoot(ctx, item.ID)
		if err != nil {
			slog.Error("failed to get path to root", "err", err, "itemID", item.ID)
			continue
		}

		conversationID, err := getConversationIDName(pathToRoot)
		if err != nil {
			slog.Error("failed to get conversation ID", "err", err, "itemID", item.ID)
			continue
		}

		slog.Info("got conversation ID", "itemID", item.ID, "conversationID", conversationID)

		conversations[conversationID] = pathToRoot
	}

	fout, err := os.Create(filepath.Join(*cacheFolder, *hnUser+".jsonl"))
	if err != nil {
		slog.Error("failed to create train file", "err", err)
		os.Exit(1)
	}
	defer fout.Close()

	for conversationID, path := range conversations {
		items := []*HNItem{}

		for _, itemID := range path {
			item, err := hn.GetItem(ctx, itemID)
			if err != nil {
				slog.Error("failed to get item", "err", err, "itemID", itemID)
				os.Exit(1)
			}

			items = append(items, item)
		}

		messages := []llm.Message{}

		for i, item := range items {
			_ = i
			text := item.Text
			role := "user"

			if item.Type == "story" {
				role = "system"
				text = systemMessage
			}

			if item.By == *hnUser {
				role = "assistant"
			}

			if role == "user" && len(items) > i+1 && items[i+1].By != *hnUser {
				next := items[i+1]
				next.Text = text + "\n\n" + next.Text
				continue
			}

			plainText, err := html2text.FromString(text, html2text.Options{OmitLinks: true})
			if err != nil {
				slog.Error("failed to convert HTML to text", "err", err, "itemID", item.ID)
				os.Exit(1)
			}

			messages = append(messages, llm.Message{
				Role:    role,
				Content: plainText,
			})
		}

		if err := json.NewEncoder(fout).Encode(Conversation{Messages: messages}); err != nil {
			slog.Error("failed to write conversation", "err", err, "conversationID", conversationID)
			os.Exit(1)
		}
	}

	if err := json.NewEncoder(fout).Encode(Conversation{
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: systemMessage,
			},
			{
				Role:    "user",
				Content: "What is your name?",
			},
			{
				Role:    "assistant",
				Content: "My name is Mimi, duh!",
			},
		}}); err != nil {
		slog.Error("failed to write conversation", "err", err)
		os.Exit(1)
	}

	slog.Info("wrote training data to", "file", fout.Name())

	// rewind fout
	fout.Seek(0, 0)

	tokenEncoder, err := gpt3encoder.NewEncoder()
	if err != nil {
		slog.Error("failed to create token encoder", "err", err)
		os.Exit(1)
	}

	mostTokens := 0

	// read it back
	dec := json.NewDecoder(fout)
	for {
		var c Conversation
		if err := dec.Decode(&c); err != nil {
			break
		}

		sess := llm.Session{
			Messages: []llm.ChatMLer{},
		}

		for _, m := range c.Messages {
			sess.Messages = append(sess.Messages, m)
		}

		chatml := sess.ChatML()
		tokens, err := tokenEncoder.Encode(chatml)
		if err != nil {
			slog.Error("failed to encode tokens", "err", err)
			os.Exit(1)
		}

		if len(tokens) > mostTokens {
			mostTokens = len(tokens)
		}
	}

	slog.Info("most tokens", "tokens", mostTokens)
}

type Conversation struct {
	Messages []llm.Message `json:"messages"`
}

func getConversationIDName(path []int) (string, error) {
	if len(path) == 0 {
		return "", fmt.Errorf("path is empty you goofus")
	}

	h := sha256.New()
	if err := json.NewEncoder(h).Encode(path); err != nil {
		return "", fmt.Errorf("failed to encode path: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func ControlCContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
		<-sc
		cancel()
		<-sc
		os.Exit(1)
	}()

	return ctx, cancel
}

func reverseIntSlice(s []int) {
	for i := len(s)/2 - 1; i >= 0; i-- {
		opp := len(s) - 1 - i
		s[i], s[opp] = s[opp], s[i]
	}
}
