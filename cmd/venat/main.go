package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/philippgille/chromem-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"
	"golang.org/x/sync/errgroup"
	"within.website/x/cmd/venat/internal/agentloop"
	"within.website/x/cmd/venat/internal/models"
	"within.website/x/cmd/venat/internal/tools/memory"
	"within.website/x/cmd/venat/internal/tools/python"
	"within.website/x/internal"

	_ "net/http/pprof"
)

var (
	dataDir     = flag.String("data-dir", "./var", "data directory for Venat data")
	metricsBind = flag.String("metrics-bind", ":9095", "metrics bind address")

	ErrMainExited = errors.New("venat: main exited")
)

func main() {
	internal.HandleStartup()
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(ErrMainExited)

	if err := run(ctx); err != nil {
		slog.Error("error running venat", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	vectorDB, err := chromem.NewPersistentDB(filepath.Join(*dataDir, "vectordb"), true)
	if err != nil {
		return fmt.Errorf("can't create vector database: %w", err)
	}

	_ = vectorDB

	dao, err := models.New(filepath.Join(*dataDir, "venat.db"), filepath.Join(*dataDir, "venat-backup.db"))
	if err != nil {
		return fmt.Errorf("can't create SQLite database: %w", err)
	}

	if err := dao.Ping(ctx); err != nil {
		return fmt.Errorf("can't ping database: %w", err)
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		http.DefaultServeMux.Handle("/metrics", promhttp.Handler())
		slog.Info("starting metrics server", "bind", *metricsBind)
		return http.ListenAndServe(*metricsBind, nil)
	})

	g.Go(func() error {
		c := cron.New()
		if _, err := c.AddFunc("@every 1h", dao.Backup); err != nil {
			return fmt.Errorf("failed to add cron job: %w", err)
		}
		c.Start()
		<-ctx.Done()
		c.Stop()
		return nil
	})

	g.Go(func() error {
		model := "glm-4.7-flash:latest"
		apiEndpoint := "http://localhost:11434/v1"
		apiKey := "ollama"

		cli := openai.NewClient(
			option.WithAPIKey(apiKey),
			option.WithBaseURL(apiEndpoint),
		)

		coll, err := vectorDB.GetOrCreateCollection("memory", map[string]string{"date": time.Now().Format(time.DateOnly)}, chromem.NewEmbeddingFuncOllama("qwen3-embedding:latest", ""))
		if err != nil {
			return fmt.Errorf("can't create memory collection: %w", err)
		}

		al := agentloop.New(
			"test-agent",
			"",
			"You are a helpful Python assistant that writes and executes Python code according to user questions. Use the python tool to execute Python code in WebAssembly. You MUST use the python tool before answering questions. Do not comment on using the python tool, just use it. When asked to remember things, use the memory_add tool.",
			model,
			[]agentloop.Tool{
				python.Impl{},
				&memory.MemoryAdd{Coll: coll},
				&memory.MemorySearch{Coll: coll},
			},
			cli,
			slog.Default(),
		)

		result, err := al.Run(ctx, "How many r's are in the word raspberry? Remember that I like my pizza cooked well done.")
		if err != nil {
			return err
		}

		fmt.Println(result.Response)

		return nil
	})

	if err := g.Wait(); err != nil {
		slog.Error("error in one of the grouped workers", "err", err)
		return fmt.Errorf("error in one of the grouped workers: %w", err)
	}

	return nil
}
