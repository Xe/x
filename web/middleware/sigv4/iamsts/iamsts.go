// Package iamsts authenticates HTTP requests signed with AWS Signature
// Version 4 by verifying them locally against a cached derived signing key
// fetched from IAM's SigningKeyService.
//
// The verifying service never holds raw secret access keys. It fetches the
// SigV4 derived key HMAC(HMAC(HMAC(HMAC("AWS4"+secret, date), region),
// service), "aws4_request") for a credential scope once, caches it for the
// server-advised TTL, and recomputes/compares signatures itself. When the
// cache is warm, verification is a pure function of the request bytes and
// the cached key — no IAM RPC on the hot path.
//
// Caching rules follow the SigV4 spec: the key is uniquely scoped to the
// exact (access_key_id, YYYYMMDD, region, service) tuple from the request's
// Credential= component, entries honor the response's cache_until and are
// never kept past not_valid_after, concurrent misses for one scope collapse
// into a single RPC, and refusals (unknown or disabled keys) are cached
// briefly so a flood of bad credentials cannot hammer IAM.
//
// Authenticate the SigningKeyService client the same way as any other iamd
// caller: give Config.HTTPClient a sigv4client transport signing with the
// verifier's own IAM credential.
package iamsts

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/twitchtv/twirp"
	"golang.org/x/sync/singleflight"

	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	"within.website/x/web/middleware/sigv4"
)

// amzTimeFormat is the AWS X-Amz-Date timestamp format.
const amzTimeFormat = "20060102T150405Z"

// defaultNegativeTTL bounds how long a refusal (unknown or disabled key) is
// remembered before IAM is asked again.
const defaultNegativeTTL = 30 * time.Second

// Config configures a Verifier. BaseURL, HTTPClient, Region, and Service are
// required.
type Config struct {
	// BaseURL is the iamd endpoint serving SigningKeyService.
	BaseURL string

	// HTTPClient carries the GetSigningKey RPCs. It must authenticate to
	// iamd — typically a sigv4client transport signing with the verifier's
	// own IAM credential.
	HTTPClient *http.Client

	// Region and Service pin the credential scope incoming requests must be
	// signed for, exactly as on sigv4.Verifier.
	Region  string
	Service string

	// MaxBodySize caps the bytes buffered to verify the payload hash. Zero
	// means unlimited.
	MaxBodySize int64

	// NegativeTTL is how long a refusal is cached. Defaults to 30s.
	NegativeTTL time.Duration

	// Now is overridable for tests. Defaults to time.Now.
	Now func() time.Time
}

// Verifier authenticates requests locally using cached derived signing keys.
// It implements sigv4.SigningKeyLookuper against a SigningKeyService client.
type Verifier struct {
	client stsv1.SigningKeyService
	inner  *sigv4.Verifier
	negTTL time.Duration
	now    func() time.Time

	sf    singleflight.Group
	mu    sync.Mutex
	cache map[scopeKey]*entry
}

// scopeKey is the exact tuple a derived key is scoped to: the literal strings
// from the request's Credential= component, unnormalized.
type scopeKey struct {
	accessKeyID string
	date        string
	region      string
	service     string
}

func (k scopeKey) String() string {
	return strings.Join([]string{k.accessKeyID, k.date, k.region, k.service}, "\x00")
}

// entry is one cache slot: either a derived key with its identity, or a
// remembered refusal (err set). expiresAt of zero means "use once, do not
// serve from cache".
type entry struct {
	signingKey []byte
	identity   *stsv1.TokenIdentity
	err        error
	expiresAt  time.Time
}

// New returns a Verifier fetching derived keys from cfg.BaseURL.
func New(cfg Config) *Verifier {
	v := &Verifier{
		client: stsv1.NewSigningKeyServiceProtobufClient(cfg.BaseURL, cfg.HTTPClient),
		negTTL: cfg.NegativeTTL,
		now:    cfg.Now,
		cache:  make(map[scopeKey]*entry),
	}
	if v.negTTL == 0 {
		v.negTTL = defaultNegativeTTL
	}
	if v.now == nil {
		v.now = time.Now
	}
	v.inner = &sigv4.Verifier{
		Region:      cfg.Region,
		Service:     cfg.Service,
		MaxBodySize: cfg.MaxBodySize,
		KeyLookup:   v,
		Now:         cfg.Now,
	}
	return v
}

// Middleware wraps next so every request is verified locally against a cached
// signing key. On success the caller identity is stored in the request
// context (see Caller). Error mapping matches the local sigv4 middleware.
func (v *Verifier) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := v.Verify(r)
		if err != nil {
			slog.DebugContext(r.Context(), "iamsts: cannot verify request", "err", err, "method", r.Method, "path", r.URL.Path)
			twirp.WriteError(w, sigv4.TwirpError(r.Context(), err))
			return
		}
		next.ServeHTTP(w, r.WithContext(withCaller(r.Context(), id)))
	})
}

// Verify authenticates r locally: the inner sigv4 verifier performs every
// pre-check (clock skew, scope pinning, signed host, payload hash) and the
// constant-time signature comparison, pulling the derived key through this
// Verifier's cache. The request body is buffered and reset so downstream
// handlers can read it. On success it returns the caller identity.
func (v *Verifier) Verify(r *http.Request) (*Identity, error) {
	keyID, err := v.inner.Verify(r)
	if err != nil {
		return nil, err
	}

	// The signature checked out, so the credential parses and its scope
	// entry was just used; re-read it for the identity. In the unlikely case
	// it expired in between, this refetches — still off the hot path.
	cred, err := sigv4.ParseCredential(r.Header.Get("Authorization"))
	if err != nil {
		return nil, err
	}
	e, err := v.entry(r.Context(), scopeKey{cred.AccessKeyID, cred.Date, cred.Region, cred.Service})
	if err != nil {
		return nil, err
	}

	id := &Identity{
		AccessKeyID:    keyID,
		OrganizationID: e.identity.GetOrganizationId(),
		PrincipalID:    e.identity.GetPrincipalId(),
		DisplayName:    e.identity.GetDisplayName(),
	}
	if t, terr := time.Parse(amzTimeFormat, r.Header.Get("X-Amz-Date")); terr == nil {
		id.SignedAt = t
	}
	return id, nil
}

// LookupSigningKey implements sigv4.SigningKeyLookuper through the cache. The
// inner verifier only calls it after the clock-skew and scope checks pass, so
// unverifiable garbage never triggers an RPC.
func (v *Verifier) LookupSigningKey(ctx context.Context, accessKeyID, date, region, service string) ([]byte, error) {
	e, err := v.entry(ctx, scopeKey{accessKeyID, date, region, service})
	if err != nil {
		return nil, err
	}
	return e.signingKey, nil
}

// entry returns the cache slot for k, fetching it once (singleflight) on a
// miss. Remembered refusals return their error.
func (v *Verifier) entry(ctx context.Context, k scopeKey) (*entry, error) {
	now := v.now()
	v.mu.Lock()
	if e, ok := v.cache[k]; ok && now.Before(e.expiresAt) {
		v.mu.Unlock()
		if e.err != nil {
			return nil, e.err
		}
		return e, nil
	}
	v.mu.Unlock()

	res, err, _ := v.sf.Do(k.String(), func() (any, error) {
		// The singleflight leader runs this fetch on behalf of every caller
		// collapsed onto it, not just the first one whose context happens to
		// be attached here. If that first caller's request is canceled (e.g.
		// the client disconnects), ctx must not cancel the in-flight RPC and
		// fail every collapsed waiter with it — an attacker could otherwise
		// induce a 500 burst for an uncached scope just by opening a request
		// and dropping the connection. context.WithoutCancel detaches from
		// the caller's cancellation/deadline while preserving trace/log
		// values (fetch logs with this ctx), and the explicit timeout below
		// replaces the deadline WithoutCancel drops.
		fctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		return v.fetch(fctx, k)
	})
	if err != nil {
		return nil, err
	}
	e := res.(*entry)
	if e.err != nil {
		return nil, e.err
	}
	return e, nil
}

// fetch performs the GetSigningKey RPC and stores the result. NOT_FOUND and
// PERMISSION_DENIED become cached refusals surfacing as ErrUnknownKey, so a
// probe cannot distinguish unknown from disabled credentials; the distinction
// is logged here. Any other failure (iamd down, transport fault) is returned
// uncached and surfaces as an internal error — an IAM outage must read as an
// outage, not as a denial. Signing key bytes are never logged.
func (v *Verifier) fetch(ctx context.Context, k scopeKey) (*entry, error) {
	resp, err := v.client.GetSigningKey(ctx, &stsv1.GetSigningKeyRequest{
		AccessKeyId: k.accessKeyID,
		Date:        k.date,
		Region:      k.region,
		Service:     k.service,
	})
	if err != nil {
		var te twirp.Error
		if errors.As(err, &te) {
			switch te.Code() {
			case twirp.NotFound, twirp.PermissionDenied:
				slog.InfoContext(ctx, "iamsts: signing key refused",
					"code", string(te.Code()),
					"access_key_id", k.accessKeyID,
					"date", k.date,
				)
				e := &entry{err: sigv4.ErrUnknownKey, expiresAt: v.now().Add(v.negTTL)}
				v.store(k, e)
				return e, nil
			}
		}
		return nil, err
	}

	e := &entry{
		signingKey: resp.GetSigningKey(),
		identity:   resp.GetIdentity(),
	}
	// Honor the server's caching bounds exactly: cache_until is the TTL,
	// clamped by not_valid_after. A response without cache_until is used for
	// this request but never cached — the server declined to grant a TTL and
	// we do not invent one.
	if cu := resp.GetCacheUntil(); cu != nil {
		e.expiresAt = cu.AsTime()
		if nva := resp.GetNotValidAfter(); nva != nil && nva.AsTime().Before(e.expiresAt) {
			e.expiresAt = nva.AsTime()
		}
	}
	if e.expiresAt.After(v.now()) {
		v.store(k, e)
	}
	return e, nil
}

// store inserts e and sweeps expired slots, so scopes for rolled-over dates
// do not accumulate: at UTC midnight every cached key expires and the next
// insert evicts the stale day.
func (v *Verifier) store(k scopeKey, e *entry) {
	now := v.now()
	v.mu.Lock()
	defer v.mu.Unlock()
	for old, oe := range v.cache {
		if !now.Before(oe.expiresAt) {
			delete(v.cache, old)
		}
	}
	v.cache[k] = e
}

// cacheLen reports the live slot count, for tests.
func (v *Verifier) cacheLen() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return len(v.cache)
}

// Identity is the verified caller stored in the request context on success.
// The fields come from the SigningKeyService response identity; SignedAt is
// parsed from the request's X-Amz-Date.
type Identity struct {
	AccessKeyID    string
	OrganizationID string
	PrincipalID    string
	DisplayName    string
	SignedAt       time.Time
}

type ctxKey struct{}

func withCaller(ctx context.Context, c *Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, c)
}

// Caller returns the verified identity stored by Middleware, if any.
func Caller(ctx context.Context) (*Identity, bool) {
	c, ok := ctx.Value(ctxKey{}).(*Identity)
	return c, ok
}
