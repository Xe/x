package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"within.website/x/cmd/mi/models"
	"within.website/x/cmd/mi/services/events"
	"within.website/x/cmd/mi/services/glance"
	"within.website/x/cmd/mi/services/homefrontshim"
	"within.website/x/cmd/mi/services/importer"
	"within.website/x/cmd/mi/services/posse"
	"within.website/x/cmd/mi/services/switchtracker"
	"within.website/x/cmd/mi/services/twitchevents"
	pb "within.website/x/gen/within/website/x/mi/v1"
	announcev1 "within.website/x/gen/within/website/x/mimi/announce/v1"
	"within.website/x/internal"
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
	blueskyAuthkey      = flag.String("bsky-authkey", "", "Bluesky authkey")
	blueskyHandle       = flag.String("bsky-handle", "", "Bluesky handle")
	blueskyPDS          = flag.String("bsky-pds", "https://bsky.social", "Bluesky PDS")
	mastodonToken       = flag.String("mastodon-token", "", "Mastodon token")
	mastodonURL         = flag.String("mastodon-url", "", "Mastodon URL")
	mastodonUsername    = flag.String("mastodon-username", "", "Mastodon username")
	mimiAnnounceURL     = flag.String("mimi-announce-url", "", "Mimi announce URL")
	twitchClientID      = flag.String("twitch-client-id", "", "twitch.tv client ID")
	twitchClientSecret  = flag.String("twitch-client-secret", "", "twitch.tv client secret")
	twitchUserID        = flag.Int("twitch-user-id", 105794391, "twitch.tv user ID")
	twitchWebhookSecret = flag.String("twitch-webhook-secret", "", "twitch.tv webhook secret")
	twitchWebhookURL    = flag.String("twitch-webhook-url", "", "URL for Twitch events to be pushed to")
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

	if *twitchClientID != "" {
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

		mux.Handle("/twitch", te)
	}

	st := switchtracker.New(dao)
	es := events.New(dao, *flyghtTrackerURL)

	gs := grpc.NewServer()

	reflection.Register(gs)

	announcev1.RegisterAnnounceServer(gs, ann)
	pb.RegisterSwitchTrackerServer(gs, st)
	pb.RegisterEventsServer(gs, es)

	mux.Handle(announcev1.AnnouncePathPrefix, announcev1.NewAnnounceServer(ann))
	mux.Handle(pb.SwitchTrackerPathPrefix, pb.NewSwitchTrackerServer(st))

	// XXX(Xe): shim through old paths
	mux.HandleFunc("/twirp/within.website.x.mi.SwitchTracker/", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("got here", "path", r.URL.Path)
		r.URL.Path = strings.Replace(r.URL.Path, "/twirp/within.website.x.mi.SwitchTracker/", "/twirp/within.website.x.mi.v1.SwitchTracker/", -1)
		slog.Info("got here", "path", r.URL.Path)
		pb.NewSwitchTrackerServer(st).ServeHTTP(w, r)
	})

	mux.Handle(pb.EventsPathPrefix, pb.NewEventsServer(es))

	// XXX(Xe): shim through old paths
	mux.HandleFunc("/twirp/within.website.x.mi.Events/", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("got here", "path", r.URL.Path)
		r.URL.Path = strings.Replace(r.URL.Path, "/twirp/within.website.x.mi.Events/", "/twirp/within.website.x.mi.v1.Events/", -1)
		slog.Info("got here", "path", r.URL.Path)
		pb.NewEventsServer(es).ServeHTTP(w, r)
	})

	mux.Handle("/front", homefrontshim.New(dao))
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
