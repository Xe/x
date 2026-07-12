package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/twitchtv/twirp"
	"golang.org/x/sync/errgroup"
	"within.website/x"
	"within.website/x/cmd/iamd/models"
	"within.website/x/cmd/iamd/services/iam/keys"
	"within.website/x/cmd/iamd/services/iam/sts"
	"within.website/x/cmd/iamd/services/iam/users"
	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
	"within.website/x/internal"
	"within.website/x/twirp/twirpslog"
)

var (
	bind          = flag.String("bind", ":9080", "HTTP bind address")
	dbLoc         = flag.String("db-loc", "./var/iamd.db", "SQLite database location")
	metricsBind   = flag.String("metrics-bind", ":9081", "Prometheus bind address")
	region        = flag.String("region", "us-east-1", "region all clients must sign for: the classic SigV4 credential-scope region and the SigV4A X-Amz-Region-Set target")
	service       = flag.String("service", "iam", "credential-scope service all clients must sign with (both algorithms)")
	maxBodySize   = flag.Int64("max-body-size", 1<<20, "max request body bytes hashed for SigV4A verification")
	bootstrapUser = flag.String("bootstrap-username", "", "if set and the DB has no users, create an admin user and signing key with this name at startup and log the credentials")

	signingKeyCacheTTL = flag.Duration("signing-key-cache-ttl", 5*time.Minute, "how long downstream verifiers may cache a derived signing key before re-fetching; bounds revocation latency")
)

func main() {
	internal.HandleStartup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lg := slog.With("program", "iamd", "version", x.Version)
	slog.SetDefault(lg)

	if err := run(ctx, lg); err != nil {
		lg.ErrorContext(ctx, "can't run service", "err", err)
		os.Exit(1)
	}
}

// newMux builds the HTTP mux serving the dual-algorithm-protected IAM and
// signing-key Twirp services. Every route runs the same pipeline — verify the
// request's signature under either classic SigV4 or SigV4A (dispatched by
// authMW per request), then resolve the caller to its DAO user (available
// downstream via sigv4a.User). The SigningKeyService route's callers are
// downstream verifiers authenticating with their own IAM credential.
func newMux(lg *slog.Logger, dao *models.DAO, authMW func(http.Handler) http.Handler, signingKeyCacheTTL time.Duration) *http.ServeMux {
	mux := http.NewServeMux()

	// Verify the signature, then annotate the request with the caller's user
	// (read downstream via sigv4a.User).
	stack := chain(authMW, UserMiddleware(dao))

	us := users.New(dao)
	mux.Handle(iamv1.UserServicePathPrefix, stack(iamv1.NewUserServiceServer(us, twirp.WithServerInterceptors(twirpslog.Interceptor(lg)))))

	ks := keys.New(dao)
	mux.Handle(iamv1.KeyServicePathPrefix, stack(iamv1.NewKeyServiceServer(ks, twirp.WithServerInterceptors(twirpslog.Interceptor(lg)))))

	sk := sts.NewSigningKeys(dao, *region, *service, signingKeyCacheTTL)
	mux.Handle(stsv1.SigningKeyServicePathPrefix, stack(stsv1.NewSigningKeyServiceServer(sk, twirp.WithServerInterceptors(twirpslog.Interceptor(lg)))))

	return mux
}

func run(ctx context.Context, lg *slog.Logger) error {
	lg.InfoContext(ctx,
		"starting up",
		"bind", *bind,
		"db-loc", *dbLoc,
		"metrics-bind", *metricsBind,
		"region", *region,
		"service", *service,
		"max-body-size", *maxBodySize,
	)

	// A zero or negative TTL would make every GetSigningKey response
	// use-once (cache_until in the past or equal to now never keeps a key),
	// silently degrading every downstream verifier to an RPC per request
	// instead of the intended warm-cache hot path. Reject it at startup
	// rather than let it fail open into a latent load problem.
	if *signingKeyCacheTTL <= 0 {
		return fmt.Errorf("-signing-key-cache-ttl must be positive, got %s", *signingKeyCacheTTL)
	}

	dao, err := models.New(*dbLoc)
	if err != nil {
		return err
	}

	if *bootstrapUser != "" {
		if err := bootstrap(ctx, lg, dao, *bootstrapUser); err != nil {
			return fmt.Errorf("bootstrap: %w", err)
		}
	}

	http.Handle("/metrics", promhttp.Handler())

	// The route middleware is the dual verifier's only consumer: it
	// authenticates every caller to iamd, including SigningKeyService.
	authMW := newDualVerifier(dao, *region, *service, *maxBodySize)
	mux := newMux(lg, dao, authMW, *signingKeyCacheTTL)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		lg.InfoContext(ctx, "Listening for metrics", "metrics-bind", *metricsBind)
		return http.ListenAndServe(*metricsBind, nil)
	})

	g.Go(func() error {
		slog.InfoContext(ctx, "starting server", "bind", *bind)
		return http.ListenAndServe(*bind, mux)
	})

	g.Go(func() error {
		<-ctx.Done()
		return ctx.Err()
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("error running group: %w", err)
	}

	return nil
}

// bootstrap creates the first admin user and a signing key when the database is
// empty, so the SigV4A-protected routes can be reached after a fresh start. It is
// a no-op once any user exists. The generated secret is logged once at startup;
// it is never accepted via a flag or env var, so no signing secret sits in the
// environment.
func bootstrap(ctx context.Context, lg *slog.Logger, dao *models.DAO, name string) error {
	existing, err := dao.ListUsers(ctx, 1, 0)
	if err != nil {
		return fmt.Errorf("check for existing users: %w", err)
	}
	if len(existing) > 0 {
		lg.InfoContext(ctx, "bootstrap skipped: users already present")
		return nil
	}

	u, err := dao.CreateUser(ctx, name)
	if err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}
	if err := dao.DB().Model(&models.User{}).Where("id = ?", u.Model.ID).Update("is_admin", true).Error; err != nil {
		return fmt.Errorf("mark admin: %w", err)
	}

	k, err := dao.CreateKey(ctx, u, "bootstrap")
	if err != nil {
		return fmt.Errorf("create bootstrap key: %w", err)
	}

	lg.InfoContext(ctx, "bootstrap complete: created admin user and signing key",
		"user_id", u.UUID,
		"access_key_id", k.AccessKeyID,
		"secret_access_key", k.SecretAccessKey,
	)
	return nil
}
