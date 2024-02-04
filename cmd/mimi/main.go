package main

import (
	"log"
	"log/slog"
	"net/http"

	"within.website/x/cmd/mimi/internal"
	"within.website/x/cmd/mimi/modules/discord"
	"within.website/x/cmd/mimi/modules/discord/flyio"
	"within.website/x/cmd/mimi/modules/scheduling"
)

func main() {
	ctx, cancel := internal.HandleStartup()
	defer cancel()

	d, err := discord.New(ctx)
	if err != nil {
		log.Fatalf("error creating discord module: %v", err)
	}

	b := flyio.New()

	d.Register(b)

	d.Open()

	slog.Info("bot started")

	http.Handle("/scheduling", scheduling.New())

	go func() {
		slog.Info("listening on :8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	<-ctx.Done()
}
