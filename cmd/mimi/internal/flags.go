package internal

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"within.website/x/internal"
	"within.website/x/web/ollama"
)

var (
	dataDir     = flag.String("data-dir", "./var", "data directory for the bot")
	ollamaModel = flag.String("ollama-model", "llama3", "ollama model tag")
	ollamaHost  = flag.String("ollama-host", "http://xe-inference.flycast:80", "ollama host")
)

func DataDir() string {
	os.MkdirAll(*dataDir, 0755)
	return *dataDir
}

func OllamaClient() *ollama.Client {
	return ollama.NewClient(*ollamaHost)
}

func OllamaModel() string {
	return *ollamaModel
}

func HandleStartup() (context.Context, context.CancelFunc) {
	internal.HandleStartup()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
		<-sc
		cancel()
	}()

	os.Setenv("OLLAMA_HOST", *ollamaHost)

	return ctx, cancel
}
