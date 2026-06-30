package twirpslog

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/twitchtv/twirp"
	"within.website/x/web/middleware/sigv4"
	"within.website/x/web/middleware/sigv4/iamsts"
)

var (
	invocations = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "within_website_x",
		Subsystem: "twirp",
		Name:      "invocations",
	}, []string{"package", "service", "method"})

	errorHits = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "within_website_x",
		Subsystem: "twirp",
		Name:      "errors",
	}, []string{"package", "service", "method"})

	latency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "within_website_x",
		Subsystem: "twirp",
		Name:      "latency_microseconds",
		Buckets:   prometheus.ExponentialBuckets(64, 2, 16),
	}, []string{"package", "service", "method"})

	usage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "within_website_x",
		Subsystem: "twirp",
		Name:      "api_calls",
	}, []string{"method", "user_id"})
)

func Interceptor(lg *slog.Logger) twirp.Interceptor {
	return func(next twirp.Method) twirp.Method {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			pkg, _ := twirp.PackageName(ctx)
			svc, _ := twirp.ServiceName(ctx)
			meth, _ := twirp.MethodName(ctx)

			// Attribute the call to the verified user, not the access key they
			// signed with. Local sigv4 verification resolves the key to its IAM
			// user (sigv4.User); services that authenticate centrally via STS
			// carry the caller on iamsts.Caller instead.
			var userID string
			if u, ok := sigv4.User(ctx); ok {
				userID = u.GetId()
			} else if caller, ok := iamsts.Caller(ctx); ok {
				userID = caller.User.GetId()
			}

			lg := lg.With("package", pkg, "service", svc, "method", meth)
			if userID != "" {
				lg = lg.With("user_id", userID)
			}

			lg.Debug("started request", "req", req)
			t0 := time.Now()
			resp, err := next(ctx, req)
			taken := time.Since(t0)

			if err != nil {
				errorHits.WithLabelValues(pkg, svc, meth).Inc()
			} else {
				invocations.WithLabelValues(pkg, svc, meth).Inc()

				if userID != "" {
					usage.WithLabelValues(fmt.Sprintf("%s/%s/%s", pkg, svc, meth), userID).Inc()
				}
			}

			latency.WithLabelValues(pkg, svc, meth).Observe(float64(taken.Microseconds()))

			lg.Debug("ended request", "err", err, "taken", taken.String())
			return resp, err
		}
	}
}
