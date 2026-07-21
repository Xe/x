// Package iamsts authenticates HTTP requests signed with AWS Signature
// Version 4A by verifying them locally against a cached ECDSA public key
// fetched from IAM's key service.
//
// The verifying service holds verification-only material: even a full
// compromise of its cache cannot mint a signature, unlike the classic SigV4
// derived-key scheme this replaces. It fetches the PKIX-encoded public key
// for an access key id once, caches it for the server-advised TTL, and
// verifies signatures itself. When the cache is warm, verification is a pure
// function of the request bytes and the cached key — no IAM RPC on the hot
// path.
//
// Caching rules: entries are keyed by access key id (SigV4A keys have no
// date/region/service scoping), honor the response's cache_until, collapse
// concurrent misses into a single RPC, and remember refusals (unknown or
// disabled keys) briefly so a flood of bad credentials cannot hammer IAM.
//
// Authenticate the client the same way as any other iamd caller: give
// Config.HTTPClient a sigv4aclient transport signing with the verifier's own
// IAM credential.
package iamsts

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/twitchtv/twirp"
	"golang.org/x/sync/singleflight"

	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	"within.website/x/web/middleware/authctx"
	"within.website/x/web/middleware/sigv4a"
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

	// HTTPClient carries the GetPublicKey RPCs. It must authenticate to
	// iamd — typically a sigv4aclient transport signing with the verifier's
	// own IAM credential.
	HTTPClient *http.Client

	// Region and Service pin the credential scope incoming requests must be
	// signed for, exactly as on sigv4a.Verifier.
	Region  string
	Service string

	// MaxBodySize caps the bytes buffered to verify the payload hash. Zero
	// means DefaultMaxBodySize (10 MiB).
	MaxBodySize int64

	// NegativeTTL is how long a refusal is cached. Defaults to 30s.
	NegativeTTL time.Duration

	// Now is overridable for tests. Defaults to time.Now.
	Now func() time.Time
}

// Verifier authenticates requests locally using cached public keys. It
// implements sigv4a.PublicKeyLookuper against a SigningKeyService client.
type Verifier struct {
	client stsv1.SigningKeyService
	inner  *sigv4a.Verifier
	negTTL time.Duration
	now    func() time.Time

	sf    singleflight.Group
	mu    sync.Mutex
	cache map[string]*entry
}

// entry is one cache slot: either a public key with its identity, or a
// remembered refusal (err set). expiresAt of zero means "use once, do not
// serve from cache".
type entry struct {
	pub       *ecdsa.PublicKey
	identity  *stsv1.TokenIdentity
	err       error
	expiresAt time.Time
}

// New returns a Verifier fetching public keys from cfg.BaseURL.
func New(cfg Config) *Verifier {
	v := &Verifier{
		client: stsv1.NewSigningKeyServiceProtobufClient(cfg.BaseURL, cfg.HTTPClient),
		negTTL: cfg.NegativeTTL,
		now:    cfg.Now,
		cache:  make(map[string]*entry),
	}
	if v.negTTL == 0 {
		v.negTTL = defaultNegativeTTL
	}
	if v.now == nil {
		v.now = time.Now
	}
	v.inner = &sigv4a.Verifier{
		Region:      cfg.Region,
		Service:     cfg.Service,
		MaxBodySize: cfg.MaxBodySize,
		KeyLookup:   v,
		Now:         cfg.Now,
	}
	return v
}

// Middleware wraps next so every request is verified locally against a cached
// public key. On success the caller identity is stored in the request
// context (see Caller). Error mapping matches the local sigv4a middleware.
func (v *Verifier) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := v.Verify(r)
		if err != nil {
			slog.DebugContext(r.Context(), "iamsts: cannot verify request", "err", err, "method", r.Method, "path", r.URL.Path)
			twirp.WriteError(w, sigv4a.TwirpError(r.Context(), err))
			return
		}
		next.ServeHTTP(w, r.WithContext(withCaller(r.Context(), id)))
	})
}

// Verify authenticates r locally: the inner sigv4a verifier performs every
// pre-check (clock skew, scope pinning, signed host, payload hash) and the
// signature comparison, pulling the public key through this Verifier's
// cache. The request body is buffered and reset so downstream handlers can
// read it. On success it returns the caller identity.
func (v *Verifier) Verify(r *http.Request) (*Identity, error) {
	keyID, err := v.inner.Verify(r)
	if err != nil {
		return nil, err
	}

	// The signature checked out, so this entry was just used; re-read it for
	// the identity. In the unlikely case it expired in between, this
	// refetches — still off the hot path.
	e, err := v.entry(r.Context(), keyID)
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

// LookupPublicKey implements sigv4a.PublicKeyLookuper through the cache. The
// inner verifier only calls it after the clock-skew and scope checks pass,
// so unverifiable garbage never triggers an RPC.
func (v *Verifier) LookupPublicKey(ctx context.Context, accessKeyID string) (*ecdsa.PublicKey, error) {
	e, err := v.entry(ctx, accessKeyID)
	if err != nil {
		return nil, err
	}
	return e.pub, nil
}

// entry returns the cache slot for accessKeyID, fetching it once
// (singleflight) on a miss. Remembered refusals return their error.
func (v *Verifier) entry(ctx context.Context, accessKeyID string) (*entry, error) {
	now := v.now()
	v.mu.Lock()
	if e, ok := v.cache[accessKeyID]; ok && now.Before(e.expiresAt) {
		v.mu.Unlock()
		if e.err != nil {
			return nil, e.err
		}
		return e, nil
	}
	v.mu.Unlock()

	res, err, _ := v.sf.Do(accessKeyID, func() (any, error) {
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
		return v.fetch(fctx, accessKeyID)
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

// fetch performs the GetPublicKey RPC and stores the result. NOT_FOUND and
// PERMISSION_DENIED become cached refusals surfacing as ErrUnknownKey, so a
// probe cannot distinguish unknown from disabled credentials; the
// distinction is logged here. Any other failure (iamd down, transport fault,
// malformed key) is returned uncached and surfaces as an internal error — an
// IAM outage must read as an outage, not as a denial.
func (v *Verifier) fetch(ctx context.Context, accessKeyID string) (*entry, error) {
	resp, err := v.client.GetPublicKey(ctx, &stsv1.GetPublicKeyRequest{AccessKeyId: accessKeyID})
	if err != nil {
		var te twirp.Error
		if errors.As(err, &te) {
			switch te.Code() {
			case twirp.NotFound, twirp.PermissionDenied:
				slog.InfoContext(ctx, "iamsts: public key refused",
					"code", string(te.Code()),
					"access_key_id", accessKeyID,
				)
				e := &entry{err: sigv4a.ErrUnknownKey, expiresAt: v.now().Add(v.negTTL)}
				v.store(accessKeyID, e)
				return e, nil
			}
		}
		return nil, err
	}

	parsed, err := x509.ParsePKIXPublicKey(resp.GetPublicKey())
	if err != nil {
		return nil, fmt.Errorf("iamsts: response public key does not parse: %w", err)
	}
	pub, ok := parsed.(*ecdsa.PublicKey)
	if !ok || pub.Curve != elliptic.P256() {
		return nil, fmt.Errorf("iamsts: response public key is %T, want ECDSA P-256", parsed)
	}

	e := &entry{pub: pub, identity: resp.GetIdentity()}
	// Honor the server's caching bound exactly: a response without
	// cache_until is used for this request but never cached — the server
	// declined to grant a TTL and we do not invent one.
	if cu := resp.GetCacheUntil(); cu != nil {
		e.expiresAt = cu.AsTime()
	}
	if e.expiresAt.After(v.now()) {
		v.store(accessKeyID, e)
	}
	return e, nil
}

// store inserts e and sweeps TTL-expired entries, so stale credentials do
// not accumulate.
func (v *Verifier) store(accessKeyID string, e *entry) {
	now := v.now()
	v.mu.Lock()
	defer v.mu.Unlock()
	for old, oe := range v.cache {
		if !now.Before(oe.expiresAt) {
			delete(v.cache, old)
		}
	}
	v.cache[accessKeyID] = e
}

// cacheLen reports the live slot count, for tests.
func (v *Verifier) cacheLen() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return len(v.cache)
}

// Identity is the verified caller stored in the request context on success.
// The fields come from the SigningKeyService response identity; SignedAt is
// parsed from the request's X-Amz-Date. It is an alias for authctx.Identity,
// shared with the sigv4/iamsts package, so a caller stored by either package
// is readable through either (or through authctx directly).
type Identity = authctx.Identity

func withCaller(ctx context.Context, c *Identity) context.Context {
	return authctx.WithCaller(ctx, c)
}

// Caller returns the verified identity stored by Middleware, if any.
func Caller(ctx context.Context) (*Identity, bool) {
	return authctx.Caller(ctx)
}
