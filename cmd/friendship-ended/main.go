package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
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

	ctx := context.Background()

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

	srv := newServer(newS3Store(s3c, *bucket, *presignExpiry), rend, *baseURL)

	hs := &http.Server{
		Addr:              *bind,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	slog.Info("listening", "bind", *bind, "bucket", *bucket, "base-url", *baseURL)
	if err := hs.ListenAndServe(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
