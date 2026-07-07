// Package twirpslog instruments Twirp services with slog request logging and
// Prometheus RPC metrics.
package twirpslog

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/twitchtv/twirp"
	xslog "within.website/x/internal/slog"
	"within.website/x/web/middleware/sigv4"
	"within.website/x/web/middleware/sigv4/iamsts"
	"within.website/x/web/middleware/sigv4a"
	sigv4aiamsts "within.website/x/web/middleware/sigv4a/iamsts"
)

var (
	invocations = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "within_website_x",
		Subsystem: "twirp",
		Name:      "invocations_total",
		Help:      "Twirp method calls, successful or not.",
	}, []string{"package", "service", "method"})

	errorHits = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "within_website_x",
		Subsystem: "twirp",
		Name:      "errors_total",
		Help:      "Twirp method calls that returned an error.",
	}, []string{"package", "service", "method"})

	latency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "within_website_x",
		Subsystem: "twirp",
		Name:      "latency_seconds",
		Help:      "Twirp method call duration.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"package", "service", "method"})

	// usage attributes calls to the verified caller for billing. Only
	// successful calls are billed, by design. The user_id label is bounded by
	// the IAM user population.
	usage = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "within_website_x",
		Subsystem: "twirp",
		Name:      "api_calls_total",
		Help:      "Successful Twirp method calls per verified user, for billing.",
	}, []string{"method", "user_id"})
)

// Interceptor returns a Twirp server interceptor that debug-logs every call to
// lg and records the RPC metrics above. Request and response payloads are
// never logged: on IAM services they carry forwarded Authorization material
// that would let a log reader replay a signed request.
func Interceptor(lg *slog.Logger) twirp.Interceptor {
	return func(next twirp.Method) twirp.Method {
		return func(ctx context.Context, req any) (any, error) {
			pkg, _ := twirp.PackageName(ctx)
			svc, _ := twirp.ServiceName(ctx)
			meth, _ := twirp.MethodName(ctx)

			// Attribute the call to the verified user, not the access key they
			// signed with. Local verification resolves the key to its IAM user
			// (User); services that authenticate centrally via STS carry the
			// caller on iamsts.Caller instead. Both the sigv4a and classic
			// middleware families populate one of these two shapes, so check
			// each family in turn: sigv4a first, since it's what iamd and new
			// services authenticate with, then the classic sigv4/iamsts chain,
			// kept as a working illustration of the derived-key deployment.
			var userID string
			if u, ok := sigv4a.User(ctx); ok {
				userID = u.GetId()
			} else if caller, ok := sigv4aiamsts.Caller(ctx); ok {
				userID = caller.PrincipalID
			} else if u, ok := sigv4.User(ctx); ok {
				userID = u.GetId()
			} else if caller, ok := iamsts.Caller(ctx); ok {
				userID = caller.PrincipalID
			}

			attrs := []slog.Attr{
				slog.String("package", pkg),
				slog.String("service", svc),
				slog.String("method", meth),
			}
			if userID != "" {
				attrs = append(attrs, slog.String("user_id", userID))
			}

			// The attributes live only on the context: the interceptor's own
			// log lines below and anything downstream logging via slog.*Context
			// get them from the ContextHandler that internal/slog.Init installs.
			// Attaching them to lg as well would emit every key twice.
			ctx = xslog.ContextWithAttrs(ctx, attrs...)

			lg.DebugContext(ctx, "started request")
			t0 := time.Now()
			resp, err := next(ctx, req)
			taken := time.Since(t0)

			invocations.WithLabelValues(pkg, svc, meth).Inc()
			if err != nil {
				errorHits.WithLabelValues(pkg, svc, meth).Inc()
			} else if userID != "" {
				usage.WithLabelValues(fmt.Sprintf("%s/%s/%s", pkg, svc, meth), userID).Inc()
			}

			latency.WithLabelValues(pkg, svc, meth).Observe(taken.Seconds())

			lg.DebugContext(ctx, "ended request", "err", err, "taken", taken.String())
			return resp, err
		}
	}
}
