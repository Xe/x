package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"within.website/x/internal"
	"within.website/x/tigris"
)

var (
	bind          = flag.String("bind", ":3000", "HTTP address to bind to")
	bucket        = flag.String("bucket", "xe-friendship-ended", "Tigris bucket to store friendships in")
	baseURL       = flag.String("base-url", "http://localhost:3000", "public URL of this service, used in OpenGraph tags")
	presignExpiry = flag.Duration("presign-expiry", time.Hour, "how long presigned image URLs stay valid")
)

func main() {
	internal.HandleStartup()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	s3c, err := tigris.Client(ctx)
	if err != nil {
		slog.Error("can't create Tigris client", "err", err)
		os.Exit(1)
	}

	rend, err := newRenderer()
	if err != nil {
		slog.Error("can't load renderer", "err", err)
		os.Exit(1)
	}

	srv := newServer(newS3Store(s3c, *bucket, *presignExpiry), rend, *baseURL, *presignExpiry)

	hs := &http.Server{
		Addr:              *bind,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("listening", "bind", *bind, "bucket", *bucket, "base-url", *baseURL)
		serveErr <- hs.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server stopped", "err", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		stop()
		slog.Info("shutting down")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := hs.Shutdown(shutdownCtx); err != nil {
			slog.Error("can't shut down cleanly", "err", err)
			os.Exit(1)
		}
	}
}
