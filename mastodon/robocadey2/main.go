package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jaytaylor/html2text"
	"within.website/ln"
	"within.website/ln/opname"
	"within.website/x/internal"
	"within.website/x/internal/stablediffusion"
	"within.website/x/web/mastodon"
)

var (
	instance = flag.String("instance", "", "mastodon instance")
	token    = flag.String("token", "", "oauth2 token")
)

func main() {
	internal.HandleStartup()

	ctx := opname.With(context.Background(), "main")
	rand.Seed(time.Now().Unix())

	cli, err := mastodon.Authenticated("robocadey2", "https://within.website/.x.botinfo", *instance, *token)
	if err != nil {
		ln.FatalErr(ctx, err)
	}

	ln.Log(ctx, ln.Info("waiting for messages"))

	for {
		ctx, cancel := context.WithCancel(ctx)
		ch, err := cli.StreamMessages(ctx, mastodon.WSSubscribeRequest{Type: "subscribe", Stream: "user"})
		if err != nil {
			ln.FatalErr(ctx, err)
		}

		for msg := range ch {
			switch msg.Event {
			case "notification":
				var n mastodon.Notification
				if err := json.Unmarshal([]byte(msg.Payload), &n); err != nil {
					ln.Error(ctx, err, ln.Info("can't parse notification"))
					continue
				}

				if err := handleNotification(cli, n); err != nil {
					ln.Error(ctx, err, ln.F{"content": n.Status.Content})
					continue
				}
			}
		}
		cancel()
	}
}

func handleNotification(c *mastodon.Client, n mastodon.Notification) error {
	text, err := html2text.FromString(n.Status.Content, html2text.Options{OmitLinks: true})
	if err != nil {
		return nil
	}
	text = strings.ReplaceAll(text, "@ ", "")

	for _, m := range n.Status.Mentions {
		text = strings.ReplaceAll(text, m.Username, "")
	}

	text = strings.TrimSpace(text)

	fmt.Printf("text: %q\n", text)

	dir, err := os.MkdirTemp("", "robocadey2")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	seed := rand.Int()

	var extra string

	if rand.Intn(128) == 69 {
		extra = ", <lora:cdi:1>"
	}

	imgs, err := stablediffusion.Generate(ctx, stablediffusion.SimpleImageRequest{
		Prompt:         "masterpiece, best quality, " + text + extra,
		NegativePrompt: "person in distance, worst quality, low quality, medium quality, deleted, lowres, comic, bad anatomy, bad hands, text, error, missing fingers, extra digit, fewer digits, cropped, jpeg artifacts, signature, watermark, username, blurry",
		Seed:           seed,
		SamplerName:    "DPM++ 2M Karras",
		BatchSize:      1,
		NIter:          1,
		Steps:          40,
		CfgScale:       7,
		Width:          512,
		Height:         512,
		SNoise:         1,

		OverrideSettingsRestoreAfterwards: true,
	})
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(dir, "result.png"), imgs.Images[0], 0600); err != nil {
		return err
	}

	response := &strings.Builder{}

	response.WriteString("@")
	response.WriteString(n.Status.Account.Acct)
	response.WriteString(" ")

	for _, m := range n.Status.Mentions {
		if m.Acct == "robocadey" {
			continue
		}

		response.WriteString("@")
		response.WriteString(m.Acct)
		response.WriteString(" ")
	}

	var att *mastodon.Attachment
	tries := 4

	for tries != 0 {
		att, err = c.UploadMedia(ctx, bytes.NewBuffer(imgs.Images[0]), "result.png", "prompt: "+text, "")
		if err != nil {
			ln.Error(ctx, err, ln.F{"tries": tries})
			time.Sleep(time.Second)
			continue
		}
		break
	}

	if tries == 0 {
		c.CreateStatus(ctx, mastodon.CreateStatusParams{
			Status:     response.String() + " @cadey please help: " + err.Error(),
			Visibility: n.Status.Visibility,
			InReplyTo:  n.Status.ID,
		})
	}

	response.WriteString("here is your image:\n\n")
	fmt.Fprintf(response, "prompt: %s\n", text)
	fmt.Fprintf(response, "seed: %d\n", seed)
	fmt.Fprintln(response, "Generated with #xediffusion early alpha")

	if _, err := c.CreateStatus(ctx, mastodon.CreateStatusParams{
		Status:      response.String(),
		MediaIDs:    []string{att.ID},
		SpoilerText: "AI generated image (can be NSFW)",
		Visibility:  n.Status.Visibility,
		InReplyTo:   n.Status.ID,
	}); err != nil {
		return err
	}

	return nil
}
