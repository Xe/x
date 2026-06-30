package main

import (
	"context"
	"errors"
	"net/http"

	"gorm.io/gorm"
	"within.website/x/cmd/iamd/models"
	"within.website/x/web/middleware/sigv4"
)

// newVerifier builds the local SigV4 verifier shared by the route middleware
// (authenticating callers to iamd) and the STS handler (validating end-user
// signatures forwarded by downstream services). It resolves access key ids to
// their secrets from the DAO.
//
// A missing key maps to sigv4.ErrUnknownKey (a 403/401 verification failure),
// but any other backing-store error propagates unwrapped so the middleware
// surfaces it as a 500 rather than masking an outage as an auth denial.
func newVerifier(dao *models.DAO, region, service string, maxBodySize int64) *sigv4.Verifier {
	return &sigv4.Verifier{
		Region:      region,
		Service:     service,
		MaxBodySize: maxBodySize,
		Lookup: sigv4.LookuperFunc(func(accessKeyID string) (string, error) {
			secret, err := dao.SecretFor(context.Background(), accessKeyID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return "", sigv4.ErrUnknownKey
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

// UserMiddleware resolves the sigv4-authenticated caller (access key id) to its
// owning DAO user and stores it in the request context via sigv4.WithUser, so
// shared downstream code can read the caller's user through sigv4.User. It must
// run after sigv4.Verifier.Middleware, which populates sigv4.KeyID. A key whose
// owning user is gone or disabled is rejected as 403.
func UserMiddleware(dao *models.DAO) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			keyID, ok := sigv4.KeyID(r.Context())
			if !ok {
				http.Error(w, "sigv4: no authenticated caller", http.StatusUnauthorized)
				return
			}
			u, err := dao.GetUserByAccessKeyID(r.Context(), keyID)
			if err != nil {
				http.Error(w, "sigv4: caller has no enabled user", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r.WithContext(sigv4.WithUser(r.Context(), u.AsProto())))
		})
	}
}
