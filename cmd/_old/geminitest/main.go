package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"within.website/x/internal"
)

var (
	geminiApiKey = flag.String("gemini-api-key", "", "The Gemini API key")
	geminiModel  = flag.String("gemini-model", "gemini-1.5-flash", "The model to use for generating text")
)

func main() {
	internal.HandleStartup()
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(*geminiApiKey))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	iter := client.ListModels(ctx)
	for {
		m, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			panic(err)
		}
		fmt.Println(m.Name, m.Description)
	}
}
