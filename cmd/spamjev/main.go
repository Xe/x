// Command spamjev receives GitHub creation webhooks.
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
)

var (
	bind          = flag.String("bind", ":8080", "HTTP bind address")
	webhookSecret = flag.String("webhook-secret", "", "GitHub webhook secret (required)")
)

func main() {
	internal.HandleStartup()
	if *webhookSecret == "" {
		slog.Error("set --webhook-secret or WEBHOOK_SECRET")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	mux := http.NewServeMux()
	mux.Handle("POST /webhook", &receiver{secret: []byte(*webhookSecret), handlers: loggingHandlers{}})
	srv := &http.Server{
		Addr:              *bind,
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("HTTP shutdown failed", "err", err)
			srv.Close()
		}
	}()

	slog.Info("starting webhook receiver", "bind", *bind)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("HTTP server failed", "err", err)
		os.Exit(1)
	}
	<-shutdownDone
}
