package main

import (
	"bytes"
	"context"
	"encoding/json"
	"expvar"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jaytaylor/html2text"
	"golang.org/x/exp/slog"
	"tailscale.com/metrics"
	"tailscale.com/tsnet"
	"tailscale.com/tsweb"
	"within.website/x/internal"
	"within.website/x/web/mastodon"
	"within.website/x/web/stablediffusion"
)

var (
	hostname = flag.String("hostname", "robocadey2", "hostname to use on tailnet")
	dataDir  = flag.String("dir", dataLocation(), "stateful data directory")
	instance = flag.String("instance", "", "mastodon instance")
	token    = flag.String("token", "", "oauth2 token")

	uploads = expvar.NewInt("gauge_robocadey2_uploads")
	retries = expvar.NewInt("gauge_robocadey2_retries")

	usageCount = metrics.LabelMap{Label: "user"}
)

func envOr(key, defaultVal string) string {
	if result, ok := os.LookupEnv(key); ok {
		return result
	}
	return defaultVal
}

func dataLocation() string {
	if dir, ok := os.LookupEnv("STATE"); ok {
		return dir
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return os.Getenv("STATE")
	}
	return filepath.Join(dir, "within.website", "x", "robocadey2")
}

func main() {
	internal.HandleStartup()

	os.MkdirAll(*dataDir, 0777)

	rand.Seed(time.Now().Unix())

	cli, err := mastodon.Authenticated("robocadey2", "https://within.website/.x.botinfo", *instance, *token)
	if err != nil {
		log.Fatal(err)
	}

	expvar.Publish("gauge_robocadey_usage_by_user", &usageCount)
	os.MkdirAll(filepath.Join(*dataDir, "tsnet"), 0777)
	srv := &tsnet.Server{
		Hostname: *hostname,
		Logf:     log.New(io.Discard, "", 0).Printf,
		AuthKey:  os.Getenv("TS_AUTHKEY"),
		Dir:      filepath.Join(*dataDir, "tsnet"),
	}

	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}

	httpCli := srv.HTTPClient()
	if err != nil {
		log.Fatal(err)
	}

	slog.Info("waiting for messages")

	b := &Bot{
		cli: cli,
		sd:  &stablediffusion.Client{HTTP: httpCli},
	}

	go func() {
		lis, err := srv.Listen("tcp", ":80")
		if err != nil {
			log.Fatalf("tsnet can't listen: %v", err)
		}

		http.DefaultServeMux.HandleFunc("/debug/varz", tsweb.VarzHandler)

		defer srv.Close()
		defer lis.Close()
		log.Fatal(http.Serve(lis, http.DefaultServeMux))
	}()

	for {
		ctx, cancel := context.WithCancel(context.Background())
		ch, err := cli.StreamMessages(ctx, mastodon.WSSubscribeRequest{Type: "subscribe", Stream: "user"})
		if err != nil {
			log.Fatal(err)
		}

		for msg := range ch {
			switch msg.Event {
			case "notification":
				var n mastodon.Notification
				if err := json.Unmarshal([]byte(msg.Payload), &n); err != nil {
					slog.Error("can't parse notification", "err", err)
					continue
				}

				if n.Type != "mention" {
					continue
				}

				if err := b.handleNotification(n); err != nil {
					slog.Error("can't handle notification", "err", err, "content", n.Status.Content)
					continue
				}
			}
		}
		cancel()
	}
}

type Bot struct {
	cli *mastodon.Client
	sd  *stablediffusion.Client
}

func (b *Bot) handleNotification(n mastodon.Notification) error {
	text, err := html2text.FromString(n.Status.Content, html2text.Options{OmitLinks: true})
	if err != nil {
		return nil
	}
	text = strings.ReplaceAll(text, "@ ", "")

	for _, m := range n.Status.Mentions {
		text = strings.ReplaceAll(text, m.Username, "")
	}

	text = strings.TrimSpace(text)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	seed := rand.Int()

	var extra string

	if rand.Intn(128) == 69 {
		extra = ", <lora:cdi:1>"
	}

	imgs, err := b.sd.Generate(ctx, stablediffusion.SimpleImageRequest{
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
	usageCount.Add(n.Status.Account.Acct, 1)

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
		att, err = b.cli.UploadMedia(ctx, bytes.NewBuffer(imgs.Images[0]), "result.png", "prompt: "+text, "")
		if err != nil {
			slog.Error("retrying", "err", err, "tries", tries)
			time.Sleep(time.Second)
			tries--
			retries.Add(1)
			continue
		}
		uploads.Add(1)
		break
	}

	if tries == 0 {
		b.cli.CreateStatus(ctx, mastodon.CreateStatusParams{
			Status:     response.String() + " @cadey please help: " + err.Error() + " (tried 4 times)",
			Visibility: n.Status.Visibility,
			InReplyTo:  n.Status.ID,
		})
	}

	response.WriteString("here is your image:\n\n")
	fmt.Fprintf(response, "prompt: %s\n", text)
	fmt.Fprintf(response, "seed: %d\n", seed)
	fmt.Fprintln(response, "Generated with #xediffusion early alpha")

	if st, err := b.cli.CreateStatus(ctx, mastodon.CreateStatusParams{
		Status:      response.String(),
		MediaIDs:    []string{att.ID},
		SpoilerText: "AI generated image (can be NSFW)",
		Visibility:  n.Status.Visibility,
		InReplyTo:   n.Status.ID,
	}); err != nil {
		return err
	} else {
		slog.Info("status created", "url", st.URL, "responsible_party", n.Status.Account.Acct, "visibility", n.Status.Visibility)
	}

	return nil
}
