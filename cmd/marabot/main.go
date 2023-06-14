package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"within.website/ln"
	"within.website/ln/opname"
	"within.website/x/internal"
	"within.website/x/web/revolt"
)

var (
	revoltToken           = flag.String("revolt-token", "", "Revolt bot token")
	revoltAPIServer       = flag.String("revolt-api-server", "https://api.revolt.chat", "API server for Revolt")
	revoltWebsocketServer = flag.String("revolt-ws-server", "wss://ws.revolt.chat", "Websocket server for Revolt")
)

func main() {
	internal.HandleStartup()

	ctx, cancel := context.WithCancel(opname.With(context.Background(), "marabot"))
	defer cancel()

	ln.Log(ctx, ln.Action("starting up"))

	// Init a new client.
	client := revolt.NewWithEndpoint(*revoltToken, *revoltAPIServer, *revoltWebsocketServer)

	mr := &MaraRevolt{
		cli: client,
	}

	client.Connect(ctx, mr)

	// Wait for close.
	sc := make(chan os.Signal, 1)

	signal.Notify(
		sc,
		syscall.SIGINT,
		syscall.SIGTERM,
		os.Interrupt,
	)
	for {
		select {
		case <-ctx.Done():
			return
		case <-sc:
			ln.Log(ctx, ln.Info("shutting down"))
			cancel()
			time.Sleep(150 * time.Millisecond)
			return
		}
	}
}

type MaraRevolt struct {
	cli *revolt.Client
	revolt.NullHandler
}

func (m *MaraRevolt) MessageCreate(ctx context.Context, msg *revolt.Message) error {
	if msg.Content == "!ping" {
		sendMsg := &revolt.SendMessage{
			Masquerade: &revolt.Masquerade{
				Name:      "cipra",
				AvatarURL: "https://cdn.xeiaso.net/avatar/" + internal.Hash("cipra", "yasomi"),
			},
		}
		sendMsg.SetContent("🏓 Pong!")

		if _, err := msg.Reply(true, sendMsg); err != nil {
			return err
		}
	}
	return nil
}
