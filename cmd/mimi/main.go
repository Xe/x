package main

import (
	"flag"
	"log"
	"log/slog"
	"net"
	"net/http"

	"google.golang.org/grpc"
	"within.website/x/cmd/mimi/internal"
	"within.website/x/cmd/mimi/modules/discord"
	"within.website/x/cmd/mimi/modules/discord/flyio"
	"within.website/x/cmd/mimi/modules/irc"
	"within.website/x/cmd/mimi/modules/scheduling"
)

var (
	grpcAddr = flag.String("grpc-addr", ":9001", "GRPC listen address")
	httpAddr = flag.String("http-addr", ":9002", "HTTP listen address")
)

func main() {
	ctx, cancel := internal.HandleStartup()
	defer cancel()

	lis, err := net.Listen("tcp", *grpcAddr)
	if err != nil {
		log.Fatalf("can't open GRPC port %s: %v", *grpcAddr, err)
	}

	d, err := discord.New(ctx)
	if err != nil {
		log.Fatalf("error creating discord module: %v", err)
	}

	b := flyio.New()

	d.Register(b)

	d.Open()

	ircBot, err := irc.New(ctx)
	if err != nil {
		log.Fatalf("error creating irc module: %v", err)
	}

	slog.Info("bot started")

	gs := grpc.NewServer()

	scheduling.RegisterSchedulingServer(gs, scheduling.New())

	mux := http.NewServeMux()

	b.RegisterHTTP(mux)
	ircBot.RegisterHTTP(mux)

	go func() {
		log.Fatal(gs.Serve(lis))
	}()

	go func() {
		log.Fatal(http.ListenAndServe(*httpAddr, mux))
	}()

	<-ctx.Done()
}
