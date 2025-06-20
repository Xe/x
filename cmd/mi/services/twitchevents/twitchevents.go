package twitchevents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	bsky "github.com/danrusei/gobot-bsky"
	"github.com/nicklaw5/helix/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/sync/errgroup"
	"within.website/x/cmd/mi/models"
	announcev1 "within.website/x/gen/within/website/x/mimi/announce/v1"
	"within.website/x/web/mastodon"
)

const (
	streamAnnouncement = "Xe is live on Twitch! https://twitch.tv/princessxen"
)

var (
	twitchEventsCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "mi_twitch_events_count",
		Help: "Number of twitch events ever processed",
	}, []string{"messageType"})
)

type Config struct {
	BlueskyAuthkey      string
	BlueskyHandle       string
	BlueskyPDS          string
	MastodonToken       string
	MastodonURL         string
	MimiAnnounceURL     string
	TwitchClientID      string
	TwitchClientSecret  string
	TwitchUserID        int
	TwitchWebhookSecret string
	TwitchWebhookURL    string
}

func (c Config) BlueskyAgent(ctx context.Context) (*bsky.BskyAgent, error) {
	bluesky := bsky.NewAgent(ctx, c.BlueskyPDS, c.BlueskyHandle, c.BlueskyAuthkey)
	if err := bluesky.Connect(ctx); err != nil {
		slog.Error("failed to connect to bluesky", "err", err)
		return nil, err
	}

	if err := bluesky.Connect(ctx); err != nil {
		slog.Error("failed to connect to bluesky", "err", err)
		return nil, err
	}

	return &bluesky, nil
}

type Server struct {
	dao      *models.DAO
	mimi     announcev1.Post
	mastodon *mastodon.Client
	cfg      Config
	twitch   *helix.Client
}

func New(ctx context.Context, dao *models.DAO, cfg Config) (*Server, error) {
	twitch, err := helix.NewClient(&helix.Options{
		ClientID:     cfg.TwitchClientID,
		ClientSecret: cfg.TwitchClientSecret,
	})

	if err != nil {
		slog.Error("can't create twitch client", "err", err)
		return nil, err
	}

	resp, err := twitch.RequestAppAccessToken([]string{"user:read:email", "user:read:broadcast"})
	if err != nil {
		slog.Error("can't request app access token", "err", err)
		return nil, err
	}

	twitch.SetAppAccessToken(resp.Data.AccessToken)

	mas, err := mastodon.Authenticated("mi_irl", "https://xeiaso.net", cfg.MastodonURL, cfg.MastodonToken)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate to mastodon: %w", err)
	}

	s := &Server{
		dao:      dao,
		mimi:     announcev1.NewPostProtobufClient(cfg.MimiAnnounceURL, &http.Client{}),
		mastodon: mas,
		cfg:      cfg,
		twitch:   twitch,
	}

	if err := s.maybeCreateWebhookSubscription(); err != nil {
		slog.Error("cant' create subscription", "err", err)
	}

	return s, nil
}

const sixteenMegs = 16777216

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body = http.MaxBytesReader(w, r.Body, sixteenMegs)
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		slog.Error("can't read from body", "err", err)
		http.Error(w, "can't read", http.StatusBadRequest)
		return
	}

	if !helix.VerifyEventSubNotification(s.cfg.TwitchWebhookSecret, r.Header, string(data)) {
		slog.Error("can't verify event", "err", "invalid secret")
		http.Error(w, "no auth", http.StatusUnauthorized)
		return
	}

	messageType := r.Header.Get("Twitch-Eventsub-Message-Type")
	twitchEventsCount.WithLabelValues(messageType).Inc()

	lg := slog.With("message_type", messageType)
	lg.Debug("got message")

	body = io.NopCloser(bytes.NewBuffer(data))

	switch messageType {
	case "webhook_callback_verification":
		err = s.handleWebhookVerification(w, body)
	case "revocation":
		go func() {
			time.Sleep(5 * time.Minute)
			if err := s.maybeCreateWebhookSubscription(); err != nil {
				slog.Error("can't create new webhook subscription after the current one was revoked", "err", err)
			}
		}()
	case "notification":
		err = s.handleNotification(r.Context(), lg, w, data)
	default:
		lg.Error("unknown event", "type", messageType, "body", json.RawMessage(data))
		http.Error(w, "unknown event", http.StatusOK)
	}

	if err != nil {
		twitchEventsCount.WithLabelValues(messageType).Inc()
		lg.Error("can't handle message", "err", err)
		http.Error(w, "can't deal with this input", http.StatusInternalServerError)
		return
	}
}

func (s *Server) maybeCreateWebhookSubscription() error {
	subs, err := s.twitch.GetEventSubSubscriptions(&helix.EventSubSubscriptionsParams{
		Status: "enabled",
	})
	if err != nil {
		return fmt.Errorf("can't get eventsub subscriptions: %w", err)
	}

	found := false
	for _, sub := range subs.Data.EventSubSubscriptions {
		if sub.Transport.Callback == s.cfg.TwitchWebhookURL {
			slog.Info("no need to resubscribe, webhook URL was found")
			found = true
			break
		}
	}

	if found {
		return nil
	}

	if _, err := s.twitch.CreateEventSubSubscription(&helix.EventSubSubscription{
		Type:    "stream.online",
		Version: "1",
		Condition: helix.EventSubCondition{
			BroadcasterUserID: strconv.Itoa(s.cfg.TwitchUserID),
		},
		Transport: helix.EventSubTransport{
			Method:   "webhook",
			Callback: s.cfg.TwitchWebhookURL,
			Secret:   s.cfg.TwitchWebhookSecret,
		},
	}); err != nil {
		return fmt.Errorf("can't create subscription: %w", err)
	}
	slog.Info("created webhook subscription")

	return nil
}

func (s *Server) handleWebhookVerification(w http.ResponseWriter, body io.Reader) error {
	var data struct {
		Challenge string `json:"challenge"`
	}

	if err := json.NewDecoder(body).Decode(&data); err != nil {
		return fmt.Errorf("can't decode challenge: %w", err)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, data.Challenge)

	return nil
}

func (s *Server) handleNotification(ctx context.Context, lg *slog.Logger, w http.ResponseWriter, bodyData []byte) error {
	var data Event

	if err := json.NewDecoder(bytes.NewBuffer(bodyData)).Decode(&data); err != nil {
		return fmt.Errorf("can't decode notification: %w", err)
	}

	lg = lg.With("event", data.Subscription.Type)

	var err error
	switch data.Subscription.Type {
	case "stream.online":
		var ev helix.EventSubStreamOnlineEvent
		if err := json.Unmarshal(data.Event, &ev); err != nil {
			return fmt.Errorf("can't unmarshal event: %w", err)
		}
		err = s.handleStreamUp(ctx, lg, &ev)
	default:
		lg.Error("unknown event", "event", data.Subscription.Type, "data", data.Event)
	}

	if err != nil {
		lg.Error("can't handle message", "err", err)
		http.Error(w, "can't deal with this event", http.StatusInternalServerError)
		return nil
	}

	return nil
}

func (s *Server) handleStreamUp(ctx context.Context, lg *slog.Logger, ev *helix.EventSubStreamOnlineEvent) error {
	lg.Info("broadcaster went online!", "username", ev.BroadcasterUserLogin)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		post, err := s.mastodon.CreateStatus(gCtx, mastodon.CreateStatusParams{
			Status: streamAnnouncement,
		})
		if err != nil {
			slog.Error("failed to announce to mastodon", "err", err)
			return err
		}
		slog.Info("posted to mastodon", "mastodon_url", post.URL)
		return nil
	})

	g.Go(func() error {
		bs, err := s.cfg.BlueskyAgent(gCtx)
		if err != nil {
			return fmt.Errorf("can't create bluesky agent: %w", err)
		}

		u, err := url.Parse("https://twitch.tv/princessxen")
		if err != nil {
			return fmt.Errorf("[unexpected] can't create twitch URL: %w", err)
		}

		q := u.Query()
		q.Set("utm_campaign", "mi_irl")
		q.Set("utm_medium", "social")
		q.Set("utm_source", "bluesky")
		u.RawQuery = q.Encode()

		var sb strings.Builder
		fmt.Fprintln(&sb, "Xe is live on stream!")
		fmt.Fprintln(&sb)
		fmt.Fprint(&sb, u.String())

		post, err := bsky.NewPostBuilder(sb.String()).
			WithExternalLink("twitch.tv - princessxen", *u, "Xe on Twitch!").
			WithFacet(bsky.Facet_Link, u.String(), u.String()).
			Build()
		if err != nil {
			return fmt.Errorf("can't build post: %w", err)
		}

		cid, uri, err := bs.PostToFeed(gCtx, post)
		if err != nil {
			return fmt.Errorf("can't post to feed: %w", err)
		}

		lg.Info("posted to bluesky", "bluesky_cid", cid, "bluesky_uri", uri, "body", sb.String())

		return nil
	})

	g.Go(func() error {
		if _, err := s.mimi.Post(gCtx, &announcev1.StatusUpdate{
			Body: streamAnnouncement,
		}); err != nil {
			slog.Error("can't announce to Mimi", "err", err)
			return nil
		}

		lg.Info("posted to Mimi")

		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

type Event struct {
	Subscription Subscription    `json:"subscription"`
	Event        json.RawMessage `json:"event"`
}

type Subscription struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	Type      string    `json:"type"`
	Version   string    `json:"version"`
	Condition Condition `json:"condition"`
	Transport Transport `json:"transport"`
	CreatedAt time.Time `json:"created_at"`
	Cost      int       `json:"cost"`
}

type Condition struct {
	BroadcasterUserID string `json:"broadcaster_user_id"`
}

type Transport struct {
	Method   string `json:"method"`
	Callback string `json:"callback"`
}
