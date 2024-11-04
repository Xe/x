package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	jetstreamClient "github.com/bluesky-social/jetstream/pkg/client"
	"github.com/bluesky-social/jetstream/pkg/client/schedulers/parallel"
	jsModels "github.com/bluesky-social/jetstream/pkg/models"
	"github.com/goccy/go-json"
	"github.com/nats-io/nats.go"
	"within.website/x/internal"
)

var (
	jetStreamURL = flag.String("jetstream-url", "wss://jetstream2.us-east.bsky.network/subscribe", "Jetstream server to subscribe to")
	natsURL      = flag.String("nats-url", "nats://localhost:4222", "nats url")
)

func main() {
	internal.HandleStartup()

	slog.Info("starting up",
		"jetstream-url", jetStreamURL,
		"nats-url", natsURL,
	)

	nc, err := nats.Connect(*natsURL)
	if err != nil {
		slog.Error("can't connect to NATS", "err", err)
		os.Exit(1)
	}
	defer nc.Close()
	slog.Info("connected to NATS")

	jsCfg := jetstreamClient.DefaultClientConfig()
	jsCfg.WebsocketURL = *jetStreamURL

	jscli, err := jetstreamClient.NewClient(
		jsCfg,
		slog.With("aspect", "jetstream"),
		parallel.NewScheduler(
			1,
			"amano",
			slog.With("aspect", "fan-out"),
			handleEvent(nc),
		),
	)
	if err != nil {
		slog.Error("can't set up jetstream client", "err", err)
		os.Exit(1)
	}
	defer jscli.Scheduler.Shutdown()

	now := time.Now().UnixNano()
	if err := jscli.ConnectAndRead(context.Background(), &now); err != nil {
		slog.Error("can't connect to jetstream", "err", err)
		os.Exit(1)
	}

	slog.Info("connected to jetstream")

	mu := sync.Mutex{}
	mu.Lock()
	mu.Lock()
}

func handleEvent(nc *nats.Conn) func(ctx context.Context, ev *jsModels.Event) error {
	return func(ctx context.Context, ev *jsModels.Event) error {
		switch ev.Kind {
		case "account":
			data, err := json.Marshal(ev.Account)
			if err != nil {
				return fmt.Errorf("can't marshal account event: %w", err)
			}

			if err := nc.Publish("amano.account", data); err != nil {
				return fmt.Errorf("can't publish account event: %w", err)
			}
		case "identity":
			data, err := json.Marshal(ev.Identity)
			if err != nil {
				return fmt.Errorf("can't marshal identity event: %w", err)
			}

			if err := nc.Publish("amano.identity", data); err != nil {
				return fmt.Errorf("can't publish identity event: %w", err)
			}
		case "commit":
			subject := fmt.Sprintf("amano.commit.%s", ev.Commit.Collection)
			data, err := json.Marshal(ev.Commit)
			if err != nil {
				return fmt.Errorf("can't marshal commit event: %w", err)
			}

			m := nats.NewMsg(subject)
			m.Data = data
			m.Header.Set("bsky-actor-did", ev.Did)
			m.Header.Set("bsky-commit-collection", ev.Commit.Collection)

			if err := nc.PublishMsg(m); err != nil {
				return fmt.Errorf("can't publish %q event: %w", subject, err)
			}
		}

		return nil
	}
}
