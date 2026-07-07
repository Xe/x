package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/twitchtv/twirp"
	"gorm.io/gorm"
	"within.website/x/cmd/iamd/models"
	"within.website/x/web/middleware/sigv4"
	"within.website/x/web/middleware/sigv4a"
)

// Algorithm prefixes of the Authorization header, the dispatch key for
// dual-algorithm verification.
const (
	algoV4  = "AWS4-HMAC-SHA256"
	algoV4A = "AWS4-ECDSA-P256-SHA256"
)

var authRequests = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "iamd_auth_requests_total",
	Help: "Authenticated requests by signature algorithm, so classic sigv4 traffic is visible while it drains.",
}, []string{"algorithm"})

// newDualVerifier builds the route middleware that authenticates callers
// signed with either classic SigV4 or SigV4A. Both verifiers resolve secrets
// through the same DAO lookup, and the same credential works under both
// algorithms: the SigV4A keypair is a pure function of the stored secret.
// Requests dispatch on the Authorization header's algorithm token; anything
// else is rejected as unauthenticated.
func newDualVerifier(dao *models.DAO, region, service string, maxBodySize int64) func(http.Handler) http.Handler {
	// fetch is the one DAO lookup shared by both verifiers. It returns the
	// raw secret or gorm.ErrRecordNotFound unwrapped; each verifier's
	// Lookuper below maps that to its own package's ErrUnknownKey sentinel,
	// since no single sentinel type can satisfy both package APIs.
	fetch := func(accessKeyID string) (string, error) {
		return dao.SecretFor(context.Background(), accessKeyID)
	}

	v4 := &sigv4.Verifier{
		Region:      region,
		Service:     service,
		MaxBodySize: maxBodySize,
		Lookup: sigv4.LookuperFunc(func(accessKeyID string) (string, error) {
			secret, err := fetch(accessKeyID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return "", sigv4.ErrUnknownKey
				}
				return "", err
			}
			return secret, nil
		}),
	}
	v4a := &sigv4a.Verifier{
		Region:      region,
		Service:     service,
		MaxBodySize: maxBodySize,
		Lookup: sigv4a.LookuperFunc(func(accessKeyID string) (string, error) {
			secret, err := fetch(accessKeyID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return "", sigv4a.ErrUnknownKey
				}
				return "", err
			}
			return secret, nil
		}),
	}

	return func(next http.Handler) http.Handler {
		v4h := v4.Middleware(next)
		v4ah := v4a.Middleware(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			switch {
			case strings.HasPrefix(auth, algoV4A):
				authRequests.WithLabelValues("sigv4a").Inc()
				v4ah.ServeHTTP(w, r)
			case strings.HasPrefix(auth, algoV4):
				authRequests.WithLabelValues("sigv4").Inc()
				v4h.ServeHTTP(w, r)
			default:
				authRequests.WithLabelValues("none").Inc()
				slog.DebugContext(r.Context(), "cannot serve request", "err", "no recognized signature algorithm", "method", r.Method, "path", r.URL.Path)
				twirp.WriteError(w, sigv4a.TwirpError(r.Context(), sigv4a.ErrMissingAuth))
			}
		})
	}
}

// chain composes middlewares so the first listed runs first (outermost):
// chain(a, b)(h) is equivalent to a(b(h)). It keeps the request pipeline
// readable as a top-to-bottom list instead of a right-nested call.
func chain(mws ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			h = mws[i](h)
		}
		return h
	}
}

// UserMiddleware resolves the authenticated caller (access key id) to its
// owning DAO user and stores it in the request context via sigv4a.WithUser —
// the canonical key all iamd services read via sigv4a.User, regardless of
// which algorithm verified the request. It reads sigv4a.KeyID first, falling
// back to sigv4.KeyID, since the caller may have been verified by either the
// SigV4A or the classic SigV4 leg of the dual verifier. It must run after
// that verifier's middleware, which populates one of the two. A key whose
// owning user is gone or disabled is rejected as 403.
func UserMiddleware(dao *models.DAO) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			keyID, ok := sigv4a.KeyID(r.Context())
			if !ok {
				keyID, ok = sigv4.KeyID(r.Context())
			}
			if !ok {
				http.Error(w, "sigv4a: no authenticated caller", http.StatusUnauthorized)
				return
			}
			u, err := dao.GetUserByAccessKeyID(r.Context(), keyID)
			if err != nil {
				http.Error(w, "sigv4a: caller has no enabled user", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r.WithContext(sigv4a.WithUser(r.Context(), u.AsProto())))
		})
	}
}
