package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	bsky "github.com/danrusei/gobot-bsky"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/emptypb"
	"within.website/x/cmd/mi/models"
	"within.website/x/proto/external/jsonfeed"
	"within.website/x/proto/mimi/announce"
	"within.website/x/web/mastodon"
)

type Announcer struct {
	dao      *models.DAO
	mastodon *mastodon.Client
	bluesky  *bsky.BskyAgent
	mimi     announce.Announce
}

func NewAnnouncer(ctx context.Context, dao *models.DAO) (*Announcer, error) {
	mas, err := mastodon.Authenticated("mi_irl", "https://xeiaso.net", *mastodonURL, *mastodonToken)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate to mastodon: %w", err)
	}

	blueAgent := bsky.NewAgent(ctx, *blueskyPDS, *blueskyHandle, *blueskyAuthkey)
	if err := blueAgent.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to bluesky: %w", err)
	}

	return &Announcer{
		dao:      dao,
		mastodon: mas,
		bluesky:  &blueAgent,
		mimi:     announce.NewAnnounceProtobufClient(*mimiAnnounceURL, &http.Client{}),
	}, nil
}

func (a *Announcer) Announce(ctx context.Context, it *jsonfeed.Item) (*emptypb.Empty, error) {
	if has, err := a.dao.HasBlogpost(ctx, it.GetUrl()); err != nil {
		return nil, err
	} else if has {
		return &emptypb.Empty{}, nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "%s\n\n%s", it.GetTitle(), it.GetUrl())

	// announce to bluesky and mastodon
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		post, err := a.mastodon.CreateStatus(gCtx, mastodon.CreateStatusParams{
			Status: sb.String(),
		})
		if err != nil {
			slog.Error("failed to announce to mastodon", "err", err)
			return err
		}
		slog.Info("posted to mastodon", "blogpost_url", it.GetUrl(), "mastodon_url", post.URL)
		return nil
	})

	g.Go(func() error {
		if err := a.bluesky.Connect(gCtx); err != nil {
			slog.Error("failed to connect to bluesky", "err", err)
			return err
		}

		u, err := url.Parse(it.GetUrl())
		if err != nil {
			return err
		}
		post, err := bsky.NewPostBuilder(sb.String()).
			WithExternalLink(it.GetTitle(), *u, "The newest post on Xe Iaso's blog").
			WithFacet(bsky.Facet_Link, it.GetUrl(), it.GetUrl()).
			Build()
		if err != nil {
			slog.Error("failed to build bluesky post", "err", err)
			return err
		}

		cid, uri, err := a.bluesky.PostToFeed(ctx, post)
		if err != nil {
			slog.Error("failed to post to bluesky", "err", err)
			return err
		}

		slog.Info("posted to bluesky", "blogpost_url", it.GetUrl(), "bluesky_cid", cid, "bluesky_uri", uri)
		return nil
	})

	g.Go(func() error {
		if _, err := a.mimi.Announce(gCtx, it); err != nil {
			slog.Error("failed to announce to mimi", "err", err)
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	_, err := a.dao.InsertBlogpost(ctx, it)
	return &emptypb.Empty{}, err
}
