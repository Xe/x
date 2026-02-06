package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	jetstreamClient "github.com/bluesky-social/jetstream/pkg/client"
	"github.com/bluesky-social/jetstream/pkg/client/schedulers/parallel"
	jsModels "github.com/bluesky-social/jetstream/pkg/models"
	"github.com/goccy/go-json"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/s2-streamstore/s2-sdk-go/s2"
	"within.website/x/bundler"
	"within.website/x/internal"
)

var (
	jetStreamURL = flag.String("jetstream-url", "wss://jetstream2.us-east.bsky.network/subscribe", "Jetstream server to subscribe to")

	s2AccessToken     = flag.String("s2-access-token", "hunter2", "S2 access token")
	s2AccountEndpoint = flag.String("s2-account-endpoint", "", "S2 account endpoint")
	s2BasinEndpoint   = flag.String("s2-basin-endpoint", "", "S2 basin endpoint")
	s2BasinName       = flag.String("s2-basin-name", "bluesky-data", "Basin name for all Bluesky data types")

	bundleCountThreshold = flag.Int("bundle-count-threshold", 100, "Number of items to batch before flushing")
	bundleDelayThreshold = flag.Duration("bundle-delay-threshold", time.Second, "Maximum time to wait before flushing a bundle")
	metricsAddr          = flag.String("metrics-addr", ":9090", "Metrics server address")
)

var (
	batchesSuccessful = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "amano_batches_successful_total",
		Help: "Total number of successful batches written to S2",
	})
	batchesFailed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "amano_batches_failed_total",
		Help: "Total number of failed batch writes to S2",
	})
	recordsProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "amano_records_processed_total",
		Help: "Total number of records processed",
	})
	reconnects = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "amano_reconnects_total",
		Help: "Total number of Jetstream reconnections",
	})
)

// streamRecord holds a record and its target stream for batching.
type streamRecord struct {
	stream s2.StreamName
	record s2.AppendRecord
}

func main() {
	internal.HandleStartup()

	slog.Info("starting up",
		"jetstream-url", *jetStreamURL,
		"has-s2-access-token", *s2AccessToken != "",
		"s2-account-endpoint", *s2AccountEndpoint,
		"s2-basin-endpoint", *s2BasinEndpoint,
		"s2-basin-name", *s2BasinName,
		"bundle-count-threshold", *bundleCountThreshold,
		"bundle-delay-threshold", *bundleDelayThreshold,
		"metrics-addr", *metricsAddr,
	)

	// Register metrics.
	prometheus.MustRegister(batchesSuccessful, batchesFailed, recordsProcessed, reconnects)

	// Start metrics server.
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		slog.Info("metrics server listening", "addr", *metricsAddr)
		if err := http.ListenAndServe(*metricsAddr, nil); err != nil {
			slog.Error("metrics server failed", "err", err)
		}
	}()

	s2Cli := s2.NewFromEnvironment(nil)
	basin := s2Cli.Basin(*s2BasinName)

	// Create a bundler for batching S2 writes.
	b := bundler.New(func(ctx context.Context, items []streamRecord) {
		// Group records by stream.
		byStream := make(map[s2.StreamName][]s2.AppendRecord)
		for _, item := range items {
			byStream[item.stream] = append(byStream[item.stream], item.record)
		}

		// Append each batch to its stream.
		for stream, records := range byStream {
			if _, err := basin.Stream(stream).Append(ctx, &s2.AppendInput{
				Records: records,
			}); err != nil {
				slog.Error("can't publish batch", "stream", stream, "num_records", len(records), "err", err)
				batchesFailed.Inc()
			} else {
				slog.Debug("published batch", "stream", stream, "num_records", len(records))
				batchesSuccessful.Inc()
			}
			recordsProcessed.Add(float64(len(records)))
		}
	})
	b.BundleCountThreshold = *bundleCountThreshold
	b.DelayThreshold = *bundleDelayThreshold
	b.HandlerLimit = 10

	jsCfg := jetstreamClient.DefaultClientConfig()
	jsCfg.WebsocketURL = *jetStreamURL

	jscli, err := jetstreamClient.NewClient(
		jsCfg,
		slog.With("aspect", "jetstream"),
		parallel.NewScheduler(
			1,
			"amano",
			slog.With("aspect", "fan-out"),
			handleEvent(b),
		),
	)
	if err != nil {
		slog.Error("can't set up jetstream client", "err", err)
		os.Exit(1)
	}
	defer jscli.Scheduler.Shutdown()

	// Track the cursor for reconnection.
	var cursor int64 = time.Now().UnixNano()

	for {
		slog.Info("connecting to jetstream", "cursor", cursor)
		if err := jscli.ConnectAndRead(context.Background(), &cursor); err != nil {
			slog.Error("jetstream connection failed, will retry", "err", err)
			reconnects.Inc()
		}

		// Exponential backoff before reconnecting.
		for i := 0; i < 5; i++ {
			time.Sleep(time.Second << uint(i))
		}
	}
}

func handleEvent(b *bundler.Bundler[streamRecord]) func(ctx context.Context, ev *jsModels.Event) error {
	return func(ctx context.Context, ev *jsModels.Event) error {
		var rec streamRecord
		var size int

		switch ev.Kind {
		case "account":
			data, err := json.Marshal(ev.Account)
			if err != nil {
				return fmt.Errorf("can't marshal account event: %w", err)
			}
			rec = streamRecord{
				stream: "account",
				record: s2.AppendRecord{Body: data},
			}
			size = len(data)

		case "identity":
			data, err := json.Marshal(ev.Identity)
			if err != nil {
				return fmt.Errorf("can't marshal identity event: %w", err)
			}
			rec = streamRecord{
				stream: "identity",
				record: s2.AppendRecord{Body: data},
			}
			size = len(data)

		case "commit":
			stream := s2.StreamName(fmt.Sprintf("commit/%s", ev.Commit.Collection))
			data, err := json.Marshal(ev.Commit)
			if err != nil {
				return fmt.Errorf("can't marshal commit event: %w", err)
			}
			rec = streamRecord{
				stream: stream,
				record: s2.AppendRecord{
					Headers: []s2.Header{
						{Name: []byte("bsky-actor-did"), Value: []byte(ev.Did)},
						{Name: []byte("bsky-commit-collection"), Value: []byte(ev.Commit.Collection)},
						{Name: []byte("bsky-commit-operation"), Value: []byte(ev.Commit.OpType)},
					},
					Body: data,
				},
			}
			size = len(data) + len(ev.Did) + len(ev.Commit.Collection) + len(ev.Commit.OpType)
		}

		// Add to bundler (non-blocking).
		if err := b.Add(rec, size); err != nil {
			return fmt.Errorf("can't add to bundler: %w", err)
		}

		return nil
	}
}
