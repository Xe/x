// Package sigv4any authenticates HTTP requests signed with either classic
// AWS Signature Version 4 (web/middleware/sigv4) or Signature Version 4A
// (web/middleware/sigv4a), dispatching per request on the Authorization
// header's algorithm token. Both verifiers can resolve the same credentials:
// a SigV4A keypair is a pure function of the same access-key-id and secret
// the classic HMAC ladder uses.
//
// Verified callers land in web/middleware/authctx regardless of which
// algorithm authenticated, so downstream code never needs to know.
package sigv4any

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/twitchtv/twirp"

	"within.website/x/web/middleware/sigv4"
	"within.website/x/web/middleware/sigv4a"
)

// Algorithm tokens beginning an Authorization header, the dispatch key.
const (
	algoV4  = "AWS4-HMAC-SHA256"
	algoV4A = "AWS4-ECDSA-P256-SHA256"
)

// Verifier authenticates requests signed with either classic SigV4 or
// SigV4A. A nil verifier disables its algorithm: matching requests are
// rejected as unauthenticated rather than dispatched, so a service can run
// SigV4A-only (or classic-only) through the same middleware while it drains
// old traffic.
type Verifier struct {
	// V4 verifies classic AWS4-HMAC-SHA256 requests.
	V4 *sigv4.Verifier

	// V4A verifies AWS4-ECDSA-P256-SHA256 requests.
	V4A *sigv4a.Verifier

	// Observe, when set, is called exactly once per request with "sigv4",
	// "sigv4a", or "none" (no recognized algorithm) before verification
	// runs. It exists so services can hook a metric without this package
	// depending on any metrics library; it must not block.
	Observe func(algorithm string)
}

// Middleware returns net/http middleware that rejects unsigned or invalid
// requests with 401/403 and otherwise passes them through, dispatching to
// the verifier matching the request's signature algorithm. On success the
// verified caller is available via authctx (each underlying verifier stores
// it there).
func (v *Verifier) Middleware(next http.Handler) http.Handler {
	var v4h, v4ah http.Handler
	if v.V4 != nil {
		v4h = v.V4.Middleware(next)
	}
	if v.V4A != nil {
		v4ah = v.V4A.Middleware(next)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		switch {
		case v4ah != nil && strings.HasPrefix(auth, algoV4A):
			v.observe("sigv4a")
			v4ah.ServeHTTP(w, r)
		case v4h != nil && strings.HasPrefix(auth, algoV4):
			v.observe("sigv4")
			v4h.ServeHTTP(w, r)
		default:
			v.observe("none")
			slog.DebugContext(r.Context(), "cannot serve request", "err", "no recognized signature algorithm", "method", r.Method, "path", r.URL.Path)
			twirp.WriteError(w, sigv4a.TwirpError(r.Context(), sigv4a.ErrMissingAuth))
		}
	})
}

func (v *Verifier) observe(algorithm string) {
	if v.Observe != nil {
		v.Observe(algorithm)
	}
}
