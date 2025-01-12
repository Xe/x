package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"within.website/x/cmd/mi/models"
	"within.website/x/cmd/mi/services/events"
	"within.website/x/cmd/mi/services/glance"
	"within.website/x/cmd/mi/services/homefrontshim"
	"within.website/x/cmd/mi/services/importer"
	"within.website/x/cmd/mi/services/posse"
	"within.website/x/cmd/mi/services/switchtracker"
	"within.website/x/cmd/mi/services/twitchevents"
	"within.website/x/internal"
	pb "within.website/x/proto/mi"
	"within.website/x/proto/mimi/announce"
)

var (
	bind         = flag.String("bind", ":8080", "HTTP bind address")
	dbLoc        = flag.String("db-loc", "./var/data.db", "SQLite database location")
	backupDBLoc  = flag.String("backup-db-loc", "./var/data.db.backup", "backup SQLite database location")
	grpcBind     = flag.String("grpc-bind", ":8081", "GRPC bind address")
	internalBind = flag.String("internal-bind", ":9195", "HTTP internal routes bind address")

	// Events flags
	flyghtTrackerURL = flag.String("flyght-tracker-url", "", "Flyght Tracker URL")

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
		"backup-db-loc", *backupDBLoc,
		"internal-bind", *internalBind,
		"bsky-handle", *blueskyHandle,
		"bsky-pds", *blueskyPDS,
		"mastodon-url", *mastodonURL,
		"mastodon-username", *mastodonUsername,
		"have-mimi-announce-url", *mimiAnnounceURL != "",
		"have-flyght-tracker-url", *flyghtTrackerURL != "",
	)

	dao, err := models.New(*dbLoc, *backupDBLoc)
	if err != nil {
		slog.Error("failed to create dao", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	ann, err := posse.New(ctx, dao, posse.Config{
		BlueskyAuthkey:  *blueskyAuthkey,
		BlueskyHandle:   *blueskyHandle,
		BlueskyPDS:      *blueskyPDS,
		MastodonToken:   *mastodonToken,
		MastodonURL:     *mastodonURL,
		MimiAnnounceURL: *mimiAnnounceURL,
	})
	if err != nil {
		slog.Error("failed to create announcer", "err", err)
		os.Exit(1)
	}

	te, err := twitchevents.New(ctx, dao, twitchevents.Config{
		BlueskyAuthkey:  *blueskyAuthkey,
		BlueskyHandle:   *blueskyHandle,
		BlueskyPDS:      *blueskyPDS,
		MastodonToken:   *mastodonToken,
		MastodonURL:     *mastodonURL,
		MimiAnnounceURL: *mimiAnnounceURL,
	})
	if err != nil {
		slog.Error("failed to create twitch events", "err", err)
		os.Exit(1)
	}

	st := switchtracker.New(dao)
	es := events.New(dao, *flyghtTrackerURL)

	gs := grpc.NewServer()

	announce.RegisterAnnounceServer(gs, ann)
	pb.RegisterSwitchTrackerServer(gs, st)
	pb.RegisterEventsServer(gs, es)

	mux.Handle(announce.AnnouncePathPrefix, announce.NewAnnounceServer(ann))
	mux.Handle(pb.SwitchTrackerPathPrefix, pb.NewSwitchTrackerServer(st))
	mux.Handle(pb.EventsPathPrefix, pb.NewEventsServer(es))
	mux.Handle("/front", homefrontshim.New(dao))
	mux.Handle("/twitch", te)
	mux.Handle("/glance", glance.New(dao))

	i := importer.New(dao)
	i.Mount(http.DefaultServeMux)

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
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
		slog.Info("starting private grpc server", "bind", *grpcBind)
		lis, err := net.Listen("tcp", *grpcBind)
		if err != nil {
			return err
		}

		return gs.Serve(lis)
	})

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

	g.Go(func() error {
		c := cron.New()
		c.AddFunc("@every 1h", dao.Backup)
		c.Run()
		return nil
	})

	if err := g.Wait(); err != nil {
		slog.Error("error doing work", "err", err)
		os.Exit(1)
	}
}
