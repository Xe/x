package main

import (
	"flag"
	"log"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"within.website/x/cmd/mimi/internal"
	"within.website/x/cmd/mimi/modules/discord"
	"within.website/x/cmd/mimi/modules/discord/flyio"
	"within.website/x/cmd/mimi/modules/scheduling"
)

var (
	grpcAddr = flag.String("grpc-addr", ":9001", "GRPC listen address")
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

	slog.Info("bot started")

	gs := grpc.NewServer()

	scheduling.RegisterSchedulingServer(gs, scheduling.New())

	go func() {
		log.Fatal(gs.Serve(lis))
	}()

	<-ctx.Done()
}
