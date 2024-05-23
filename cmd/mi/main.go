package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
	"within.website/x/cmd/mi/models"
	"within.website/x/internal"
	pb "within.website/x/proto/mi"
	"within.website/x/proto/mimi/announce"
)

var (
	bind         = flag.String("bind", ":8080", "HTTP bind address")
	dbLoc        = flag.String("db-loc", "./var/data.db", "")
	internalBind = flag.String("internal-bind", ":9195", "HTTP internal routes bind address")

	// POSSE flags
	blueskyAuthkey   = flag.String("bsky-authkey", "", "Bluesky authkey")
	blueskyHandle    = flag.String("bsky-handle", "", "Bluesky handle")
	blueskyPDS       = flag.String("bsky-pds", "https://bsky.social", "Bluesky PDS")
	mastodonToken    = flag.String("mastodon-token", "", "Mastodon token")
	mastodonURL      = flag.String("mastodon-url", "", "Mastodon URL")
	mastodonUsername = flag.String("mastodon-username", "", "Mastodon username")
	mimiAnnounceURL  = flag.String("mimi-announce-url", "", "Mimi announce URL")
)

func main() {
	internal.HandleStartup()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	slog.Info(
		"starting up",
		"bind", *bind,
		"db-loc", *dbLoc,
		"internal-bind", *internalBind,
		"bsky-handle", *blueskyHandle,
		"bsky-pds", *blueskyPDS,
		"mastodon-url", *mastodonURL,
		"mastodon-username", *mastodonUsername,
	)

	dao, err := models.New(*dbLoc)
	if err != nil {
		slog.Error("failed to create dao", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	ann, err := NewAnnouncer(ctx, dao)
	if err != nil {
		slog.Error("failed to create announcer", "err", err)
		os.Exit(1)
	}

	mux.Handle(announce.AnnouncePathPrefix, announce.NewAnnounceServer(ann))
	mux.Handle(pb.SwitchTrackerPathPrefix, pb.NewSwitchTrackerServer(NewSwitchTracker(dao)))
	mux.Handle("/front", &HomeFrontShim{dao: dao})

	i := &Importer{db: dao.DB()}
	i.Mount(http.DefaultServeMux)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := dao.Ping(r.Context()); err != nil {
			slog.Error("database not healthy", "err", err)
			http.Error(w, "database not healthy", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})
	http.Handle("/metrics", promhttp.Handler())

	g, _ := errgroup.WithContext(ctx)

	g.Go(func() error {
		slog.Info("starting internal server", "bind", *internalBind)
		return http.ListenAndServe(*internalBind, nil)
	})

	g.Go(func() error {
		slog.Info("starting server", "bind", *bind)
		return http.ListenAndServe(*bind, mux)
	})

	g.Go(func() error {
		<-ctx.Done()
		return ctx.Err()
	})

	if err := g.Wait(); err != nil {
		slog.Error("error doing work", "err", err)
		os.Exit(1)
	}
}
