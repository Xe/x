package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/a-h/templ"
	"within.website/x/htmx"
	"within.website/x/internal"
	"within.website/x/web"
	"within.website/x/xess"
)

//go:generate go tool templ generate

var (
	apiURL = flag.String("api-url", "https://waifuwave.fly.dev/generate", "API backend URL")
	bind   = flag.String("bind", ":3924", "TCP address to bind to")
)

func main() {
	internal.HandleStartup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = ctx

	mux := http.NewServeMux()
	htmx.Mount(mux)
	xess.Mount(mux)

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		prompt := r.FormValue("prompt")
		negPrompt := r.FormValue("negative_prompt")

		var imageURL string
		var howLong time.Duration

		if prompt != "" && negPrompt != "" {
			t0 := time.Now()
			output, err := getImage(r.Context(), *apiURL, prompt, negPrompt)
			if err != nil {
				slog.Error("can't make image", "err", err)
				templ.Handler(
					xess.Simple("It broke!", ohNoes(err.Error())),
					templ.WithStatus(http.StatusInternalServerError),
				).ServeHTTP(w, r)
				return
			}
			howLong = time.Now().Sub(t0)

			imageURL = output.URL
		}

		if prompt == "" {
			prompt = "1girl, solo, flower, long hair, outdoors, green hair, running, smiling, parted lips, blue eyes, cute, sakura blossoms, spring, sun, blue sky, depth of field, cat ears, kimono, bow, onsen, pool, warm lighting, safe"
		}

		if negPrompt == "" {
			negPrompt = "crop, bad hands, worst hands, worst quality"
		}

		templ.Handler(
			xess.Simple(
				"Nomadic Infra Demo",
				index(prompt, negPrompt, imageURL, howLong),
			),
		).ServeHTTP(w, r)
	})

	slog.Info("listening", "bind", *bind)
	log.Fatal(http.ListenAndServe(*bind, mux))
}

func getImage(ctx context.Context, apiURL, prompt, negPrompt string) (*Output, error) {
	buf := bytes.Buffer{}

	if err := json.NewEncoder(&buf).Encode(Input{negPrompt, prompt}); err != nil {
		return nil, fmt.Errorf("can't encode: %w", err)
	}

	resp, err := http.Post(apiURL, "application/json", &buf)
	if err != nil {
		return nil, fmt.Errorf("can't request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result Output
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("can't read result: %w", err)
	}

	return &result, nil
}

type Input struct {
	NegativePrompt string `json:"negative_prompt"`
	Prompt         string `json:"prompt"`
}

type Output struct {
	Fname []string `json:"fname"`
	URL   string   `json:"url"`
}
