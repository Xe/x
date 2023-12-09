package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"within.website/x/cmd/mimi/ollama"
	"within.website/x/internal"
	"within.website/x/llm"
	"within.website/x/llm/llamaguard"
)

var (
	model = flag.String("model", "xe/llamaguard", "model to use")
)

func main() {
	internal.HandleStartup()

	var messages []llm.Message

	if err := json.NewDecoder(os.Stdin).Decode(&messages); err != nil {
		panic(err)
	}

	slog.Info("got messages", "num", len(messages))

	out, err := llamaguard.Prompt(messages)
	if err != nil {
		panic(err)
	}

	fmt.Println(out)

	oc, err := ollama.ClientFromEnvironment()
	if err != nil {
		panic(err)
	}

	var result strings.Builder
	if err := oc.Generate(context.Background(), &ollama.GenerateRequest{
		Model:  *model,
		Prompt: out,
		Raw:    true,
	}, func(gr ollama.GenerateResponse) error {
		result.WriteString(gr.Response)
		return nil
	}); err != nil {
		panic(err)
	}

	fmt.Println(strings.TrimSpace(result.String()))
}
