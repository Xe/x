package posse

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	bsky "github.com/danrusei/gobot-bsky"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/twitchtv/twirp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/emptypb"
	"within.website/x/cmd/mi/models"
	jsonfeedv1 "within.website/x/gen/within/website/x/external/jsonfeed/v1"
	announcev1 "within.website/x/gen/within/website/x/mimi/announce/v1"
	"within.website/x/web/mastodon"
)

var (
	possePosts = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mi_posse_posts",
		Help: "Number of posts sent to social networks.",
	}, []string{"service"})

	posseErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mi_posse_errors",
		Help: "Number of errors encountered while sending posts to social networks.",
	}, []string{"service"})
)

type Announcer struct {
	dao      *models.DAO
	mastodon *mastodon.Client
	mimi     announcev1.Announce
	cfg      Config

	announcev1.UnimplementedAnnounceServer
}

type Config struct {
	BlueskyAuthkey  string
	BlueskyHandle   string
	BlueskyPDS      string
	MastodonToken   string
	MastodonURL     string
	MimiAnnounceURL string
}

func New(ctx context.Context, dao *models.DAO, cfg Config) (*Announcer, error) {
	mas, err := mastodon.Authenticated("mi_irl", "https://xeiaso.net", cfg.MastodonURL, cfg.MastodonToken)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate to mastodon: %w", err)
	}

	return &Announcer{
		dao:      dao,
		mastodon: mas,
		mimi:     announcev1.NewAnnounceProtobufClient(cfg.MimiAnnounceURL, &http.Client{}),
		cfg:      cfg,
	}, nil
}

func (a *Announcer) Announce(ctx context.Context, it *jsonfeedv1.Item) (*emptypb.Empty, error) {
	u, err := url.Parse(it.GetUrl())
	if err != nil {
		slog.Error("[unexpected] can't parse URL", "err", err, "url", it.GetUrl())
		return &emptypb.Empty{}, nil
	}

	if u.Host != "xeiaso.net" {
		slog.Info("skipping announcement", "url", it.GetUrl(), "reason", "non-prod URLs")
		return &emptypb.Empty{}, nil
	}
	if has, err := a.dao.HasBlogpost(ctx, it.GetUrl()); err != nil {
		return nil, err
	} else if has {
		return &emptypb.Empty{}, nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "%s\n\n%s", it.GetTitle(), it.GetUrl())

	if _, err := a.dao.InsertBlogpost(ctx, it); err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	// announce to bluesky and mastodon
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		post, err := a.mastodon.CreateStatus(gCtx, mastodon.CreateStatusParams{
			Status: sb.String() + "?utm_campaign=mi_irl&utm_medium=social&utm_source=mastodon",
		})
		if err != nil {
			posseErrors.WithLabelValues("mastodon").Inc()
			slog.Error("failed to announce to mastodon", "err", err)
			return err
		}
		possePosts.WithLabelValues("mastodon").Inc()
		slog.Info("posted to mastodon", "blogpost_url", it.GetUrl(), "mastodon_url", post.URL)
		return nil
	})

	g.Go(func() error {
		bluesky := bsky.NewAgent(gCtx, a.cfg.BlueskyPDS, a.cfg.BlueskyHandle, a.cfg.BlueskyAuthkey)
		if err := bluesky.Connect(gCtx); err != nil {
			posseErrors.WithLabelValues("bluesky").Inc()
			slog.Error("failed to connect to bluesky", "err", err)
			return err
		}

		if err := bluesky.Connect(gCtx); err != nil {
			posseErrors.WithLabelValues("bluesky").Inc()
			slog.Error("failed to connect to bluesky", "err", err)
			return err
		}

		u, err := url.Parse(it.GetUrl())
		if err != nil {
			posseErrors.WithLabelValues("bluesky").Inc()
			slog.Error("failed to parse url", "err", err)
			return err
		}

		q := u.Query()
		q.Set("utm_campaign", "mi_irl")
		q.Set("utm_medium", "social")
		q.Set("utm_source", "bluesky")
		u.RawQuery = q.Encode()

		post, err := bsky.NewPostBuilder(sb.String()).
			WithExternalLink(it.GetTitle(), *u, "The newest post on Xe Iaso's blog").
			WithFacet(bsky.Facet_Link, it.GetUrl(), it.GetUrl()).
			Build()
		if err != nil {
			posseErrors.WithLabelValues("bluesky").Inc()
			slog.Error("failed to build bluesky post", "err", err)
			return err
		}

		cid, uri, err := bluesky.PostToFeed(ctx, post)
		if err != nil {
			posseErrors.WithLabelValues("bluesky").Inc()
			slog.Error("failed to post to bluesky", "err", err)
			return err
		}

		possePosts.WithLabelValues("bluesky").Inc()
		slog.Info("posted to bluesky", "blogpost_url", it.GetUrl(), "bluesky_cid", cid, "bluesky_uri", uri)
		return nil
	})

	g.Go(func() error {
		if _, err := a.mimi.Announce(gCtx, it); err != nil {
			slog.Error("failed to announce to mimi", "err", err)
			return nil
		}
		possePosts.WithLabelValues("irc").Inc()
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
