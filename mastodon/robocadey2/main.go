package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	madon "github.com/McKael/madon/v2"
	"github.com/jaytaylor/html2text"
	"within.website/ln"
	"within.website/ln/opname"
	"within.website/x/internal"
)

var (
	instance  = flag.String("instance", "", "mastodon instance")
	appID     = flag.String("app-id", "", "oauth2 app id")
	appSecret = flag.String("app-secret", "", "oauth2 app secret")
	token     = flag.String("token", "", "oauth2 token")
)

func main() {
	internal.HandleStartup()

	ctx := opname.With(context.Background(), "main")
	rand.Seed(time.Now().Unix())

	c, err := madon.RestoreApp("Robocadey2", *instance, *appID, *appSecret, &madon.UserToken{AccessToken: *token})
	if err != nil {
		ln.FatalErr(opname.With(ctx, "restore-app"), err)
	}

	ln.Log(ctx, ln.Info("waiting for messages"))

	for {
		evChan := make(chan madon.StreamEvent, 10)
		stop := make(chan bool)
		done := make(chan bool)
		ctx = opname.With(context.Background(), "notifications-stream")

		err = c.StreamListener("user", "", evChan, stop, done)
		if err != nil {
			ln.FatalErr(ctx, err)
		}

		for {
			select {
			case _, _ = <-done:
				goto redo
			case ev := <-evChan:
				ln.Log(ctx, ln.F{"event": ev.Event})
				switch ev.Event {
				case "error":
					ln.Error(opname.With(ctx, "event-parse"), err)
					break
				case "notification":
					n := ev.Data.(madon.Notification)

					if n.Type != "mention" {
						continue
					}

					if err := handleNotification(c, n); err != nil {
						ln.Error(ctx, err, ln.F{"content": n.Status.Content})
						continue
					}
				}
			}
		}
	}

redo:
}

type DiffusionMetadata struct {
	Prompt     string  `json:"prompt"`
	Outdir     string  `json:"outdir"`
	SkipGrid   bool    `json:"skip_grid"`
	SkipSave   bool    `json:"skip_save"`
	DDIMSteps  int     `json:"ddim_steps"`
	FixedCode  bool    `json:"fixed_code"`
	DDIMEta    float64 `json:"ddim_eta"`
	NIter      int     `json:"n_iter"`
	H          int     `json:"H"`
	W          int     `json:"W"`
	C          int     `json:"C"`
	F          int     `json:"f"`
	NSamples   int     `json:"n_samples"`
	NRows      int     `json:"n_rows"`
	Scale      float64 `json:"scale"`
	Device     string  `json:"device"`
	Seed       int     `json:"seed"`
	UNETBs     int     `json:"unet_bs"`
	Turbo      bool    `json:"turbo"`
	Precision  string  `json:"precision"`
	Format     string  `json:"format"`
	Sampler    string  `json:"sampler"`
	Checkpoint string  `json:"ckpt"`
}

func handleNotification(c *madon.Client, n madon.Notification) error {
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

	metadata, err := makeImage(ctx, text, dir)
	if err != nil {
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

	att, err := c.UploadMedia(filepath.Join(dir, "00002.png"), "Waifu Diffusion v1.3 with the prompt: "+text, "")
	if err != nil {
		c.PostStatus(madon.PostStatusParams{
			Text:       response.String() + " @cadey please help: " + err.Error(),
			Visibility: n.Status.Visibility,
			InReplyTo:  n.Status.ID,
		})
		return err
	}

	response.WriteString("here is your image:\n\n")
	fmt.Fprintf(response, "prompt: %s\n", text)
	fmt.Fprintf(response, "seed: %d\n", metadata.Seed)
	response.WriteString("Generated with Waifu Diffusion v1.3 (float16)")

	c.PostStatus(madon.PostStatusParams{
		Text:        response.String(),
		MediaIDs:    []int64{att.ID},
		Sensitive:   true,
		SpoilerText: "AI generated image (can be NSFW)",
		Visibility:  n.Status.Visibility,
		InReplyTo:   n.Status.ID,
	})

	return nil
}

func makeImage(ctx context.Context, prompt, dir string) (*DiffusionMetadata, error) {
	err := os.WriteFile(filepath.Join(dir, "prompt.txt"), []byte(prompt), 0666)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "./do_image_gen.sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("OUTDIR=%s", dir), fmt.Sprintf("SEED=%d", rand.Int31()))
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		return nil, err
	}

	var result DiffusionMetadata
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
