package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

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
	"within.website/x/web/middleware/sigv4"
)

var (
	bind          = flag.String("bind", ":9080", "HTTP bind address")
	dbLoc         = flag.String("db-loc", "./var/iamd.db", "SQLite database location")
	metricsBind   = flag.String("metrics-bind", ":9081", "Prometheus bind address")
	region        = flag.String("region", "us-east-1", "SigV4 credential-scope region all clients must sign with")
	service       = flag.String("service", "iam", "SigV4 credential-scope service all clients must sign with")
	maxBodySize   = flag.Int64("max-body-size", 1<<20, "max request body bytes hashed for SigV4 verification")
	bootstrapUser = flag.String("bootstrap-username", "", "if set and the DB has no users, create an admin user and signing key with this name at startup and log the credentials")
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

// newMux builds the HTTP mux serving the SigV4-protected IAM and STS Twirp
// services. Every route runs the same pipeline — verify the SigV4 signature,
// then resolve the caller to its DAO user (available downstream via sigv4.User)
// — including the STS route, whose callers are downstream verifiers
// authenticating with their own IAM credential.
func newMux(lg *slog.Logger, dao *models.DAO, verifier *sigv4.Verifier) *http.ServeMux {
	mux := http.NewServeMux()

	// Verify the signature, then annotate the request with the caller's user
	// (read downstream via sigv4.User).
	stack := chain(verifier.Middleware, UserMiddleware(dao))

	us := users.New(dao)
	mux.Handle(iamv1.UserServicePathPrefix, stack(iamv1.NewUserServiceServer(us, twirp.WithServerInterceptors(twirpslog.Interceptor(lg)))))

	ks := keys.New(dao)
	mux.Handle(iamv1.KeyServicePathPrefix, stack(iamv1.NewKeyServiceServer(ks, twirp.WithServerInterceptors(twirpslog.Interceptor(lg)))))

	stsSvc := sts.New(dao, verifier)
	mux.Handle(stsv1.STSServicePathPrefix, stack(stsv1.NewSTSServiceServer(stsSvc, twirp.WithServerInterceptors(twirpslog.Interceptor(lg)))))

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

	// One verifier backs the route middleware (authenticating every caller to
	// iamd, including STS) and the STS handler's bodyless end-user checks.
	verifier := newVerifier(dao, *region, *service, *maxBodySize)
	mux := newMux(lg, dao, verifier)

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
// empty, so the SigV4-protected routes can be reached after a fresh start. It is
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
