package main

import (
	"context"
	"errors"
	"net/http"

	"gorm.io/gorm"
	"within.website/x/cmd/iamd/models"
	"within.website/x/web/middleware/sigv4a"
)

// newVerifier builds the local SigV4A verifier used by the route middleware to
// authenticate callers to iamd. It resolves access key ids to their secrets
// from the DAO.
//
// A missing key maps to sigv4a.ErrUnknownKey (a 403/401 verification failure),
// but any other backing-store error propagates unwrapped so the middleware
// surfaces it as a 500 rather than masking an outage as an auth denial.
func newVerifier(dao *models.DAO, region, service string, maxBodySize int64) *sigv4a.Verifier {
	return &sigv4a.Verifier{
		Region:      region,
		Service:     service,
		MaxBodySize: maxBodySize,
		Lookup: sigv4a.LookuperFunc(func(accessKeyID string) (string, error) {
			secret, err := dao.SecretFor(context.Background(), accessKeyID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return "", sigv4a.ErrUnknownKey
				}
				return "", err
			}
			return secret, nil
		}),
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

// UserMiddleware resolves the sigv4a-authenticated caller (access key id) to
// its owning DAO user and stores it in the request context via
// sigv4a.WithUser, so shared downstream code can read the caller's user
// through sigv4a.User. It must run after sigv4a.Verifier.Middleware, which
// populates sigv4a.KeyID. A key whose owning user is gone or disabled is
// rejected as 403.
func UserMiddleware(dao *models.DAO) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			keyID, ok := sigv4a.KeyID(r.Context())
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
