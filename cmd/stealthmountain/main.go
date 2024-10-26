package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"time"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	bskyData "github.com/bluesky-social/indigo/api/bsky"
	jsModels "github.com/bluesky-social/jetstream/pkg/models"
	"github.com/goccy/go-json"
	"github.com/nats-io/nats.go"
	"within.website/x/internal"
	bsky "within.website/x/web/bskybot"
)

const (
	PostTopic = "amano.commit.app.bsky.feed.post"
)

var (
	blueskyAuthkey = flag.String("bsky-authkey", "", "Bluesky authkey")
	blueskyHandle  = flag.String("bsky-handle", "", "Bluesky handle")
	blueskyPDS     = flag.String("bsky-pds", "https://bsky.social", "Bluesky PDS")
	natsURL        = flag.String("nats-url", "nats://localhost:4222", "nats url")

	sneakPeakRegex = regexp.MustCompile("(?i)sneak peak")
)

func main() {
	internal.HandleStartup()

	slog.Info("starting up",
		"have-bsky-authkey", *blueskyAuthkey != "",
		"bsky-handle", *blueskyHandle,
		"bsky-pds", *blueskyPDS,
		"nats-url", *natsURL,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bsAgent, err := bskyAuth(ctx, *blueskyPDS, *blueskyHandle, *blueskyAuthkey)
	if err != nil {
		slog.Error("can't auth to bluesky", "err", err)
		os.Exit(1)
	}

	nc, err := nats.Connect(*natsURL)
	if err != nil {
		slog.Error("can't connect to NATS", "err", err)
		os.Exit(1)
	}
	defer nc.Close()
	slog.Info("connected to NATS")

	sub, err := nc.SubscribeSync(PostTopic)
	if err != nil {
		slog.Error("can't subscribe to post feed", "err", err)
		os.Exit(1)
	}
	defer sub.Drain()

	for {
		m, err := sub.NextMsg(time.Second)
		if err != nil {
			slog.Error("can't read message", "err", err)
			continue
		}

		var commit jsModels.Commit
		if err := json.Unmarshal(m.Data, &commit); err != nil {
			slog.Error("can't unmarshal commit", "err", err)
			continue
		}

		if commit.Operation == "delete" {
			continue
		}

		var post bskyData.FeedPost
		if err := json.Unmarshal(commit.Record, &post); err != nil {
			slog.Error("can't unmarshal post", "err", err)
			continue
		}

		if !sneakPeakRegex.MatchString(post.Text) {
			continue
		}

		actorID := m.Header.Get("bsky-actor-did")
		slog.Info("found a stealth mountain!", "id", commit.Rev, "actor", actorID)
		reply, err := bsky.NewPostBuilder(`I think you mean "sneak peek"`).Build()
		if err != nil {
			slog.Error("can't build reply post", "err", err)
		}
		parent := comatproto.RepoStrongRef{
			LexiconTypeID: "app.bsky.feed.post",
			Uri:           fmt.Sprintf("at://%s/app.bsky.feed.post/%s", actorID, commit.RKey),
			Cid:           commit.CID,
		}
		root := parent

		if post.Reply != nil {
			root = *post.Reply.Root
		}

		reply.Reply = &bskyData.FeedPost_ReplyRef{
			Parent: &parent,
			Root:   &root,
		}

		reply.CreatedAt = time.Now().UTC().Format(time.RFC3339)

		cid, uri, err := bsAgent.PostToFeed(ctx, reply)
		if err != nil {
			slog.Error("cannot post to feed", "err", err)
			continue
		}

		slog.Info("posted to bluesky", "bluesky_cid", cid, "bluesky_uri", uri)
	}
}

func bskyAuth(ctx context.Context, pds, handle, authkey string) (*bsky.BskyAgent, error) {
	bluesky := bsky.NewAgent(ctx, pds, handle, authkey)

	slog.Debug("connecting to bluesky server", "pds", pds, "handle", handle)

	if err := bluesky.Connect(ctx); err != nil {
		slog.Error("failed to connect to bluesky", "err", err)
		return nil, err
	}

	return &bluesky, nil
}
