# Integrating iamsts with a Twirp API

`iamsts` authenticates incoming SigV4A-signed requests locally: it fetches the
public verification key for each access key id from iamd's
`SigningKeyService` once, caches it for the server-advised TTL, and
recomputes signatures itself. When the cache is warm there are **zero** IAM
RPCs on the request path.

Unlike the classic SigV4 `iamsts` this guide is modeled on, the cache holds
only public material: even a full compromise of this service's cache lets an
attacker verify signatures, never mint them.

## Prerequisites

- An IAM credential for your service (the "verifier credential"), used only to
  authenticate the `GetPublicKey` fetches to iamd. Create one with
  `KeyService/CreateKey` or take it from iamd's bootstrap output.
- The iamd base URL.
- The region and service your own verifier checks incoming requests against
  (`iamsts.Config.Region`/`.Service`). This is **your service's own choice**,
  not a fleet-wide constant: `GetPublicKey` returns a credential's public key
  regardless of what scope the original request declared, so iamd never
  refuses a key for "the wrong" region or service the way `GetSigningKey`
  does. The scope only has to agree with what your own clients sign with via
  `sigv4aclient`/`sigv4a.NewSigner`.
- Separately, the credential your service authenticates _to iamd_ with must
  satisfy iamd's own scope (its `-region`/`-service` flags, defaults
  `us-east-1` / `iam`) — that's the scope `sigv4aclient.Config` below signs
  the `GetPublicKey` call itself with, and it need not match the first bullet.

## Wiring

Build a signed HTTP client with `sigv4aclient`, construct the verifier with
`iamsts.New`, and wrap each Twirp service handler when registering it on the
mux:

```go
import (
	"net/http"

	"github.com/twitchtv/twirp"

	widgetsv1 "within.website/x/gen/within/website/x/widgets/v1" // your service
	"within.website/x/web/middleware/sigv4a/iamsts"
	"within.website/x/web/middleware/sigv4a/sigv4aclient"
)

func newMux() (*http.ServeMux, error) {
	// The transport that signs GetPublicKey calls to iamd with this
	// service's own IAM credential, scoped to iamd's own (region, service).
	rt, err := sigv4aclient.NewSigV4ARoundTripper(&sigv4aclient.Config{
		Region:      "us-east-1",
		AccessKey:   verifierAKID,   // this service's credential
		SecretKey:   verifierSecret, // never an end user's
		ServiceName: "iam",
	}, nil)
	if err != nil {
		return nil, err
	}

	verifier := iamsts.New(iamsts.Config{
		BaseURL:     "https://iamd.example",
		HTTPClient:  &http.Client{Transport: rt},
		Region:      "us-east-1", // your service's own scope for its callers
		Service:     "widgets",
		MaxBodySize: 1 << 20, // cap bytes buffered for the payload-hash check
	})

	mux := http.NewServeMux()
	svc := widgetsv1.NewWidgetServiceServer(&widgetServer{})
	mux.Handle(widgetsv1.WidgetServicePathPrefix, verifier.Middleware(svc))
	return mux, nil
}
```

One `*iamsts.Verifier` should be shared across every route on the mux — the
key cache lives on it. Wrap each Twirp `PathPrefix` handler the same way (or
wrap the whole mux if every route requires auth).

`Config` also accepts `NegativeTTL` (how long a refusal is remembered,
default 30s) and `Now` (a clock override for tests). Leave both zero in
production.

## Reading the caller identity

On success the middleware stores the verified caller in the request context.
Twirp handlers receive that context directly:

```go
func (s *widgetServer) MakeWidget(ctx context.Context, req *widgetsv1.MakeWidgetReq) (*widgetsv1.MakeWidgetResp, error) {
	caller, ok := iamsts.Caller(ctx)
	if !ok {
		return nil, twirp.NewError(twirp.Unauthenticated, "no verified caller")
	}
	// caller.PrincipalID is the IAM user UUID; caller.DisplayName the user
	// name; caller.AccessKeyID the key that signed the request.
	_ = caller
	...
}
```

`twirp/twirpslog.Interceptor` already reads this package's `iamsts.Caller`
(checked ahead of the classic `sigv4`/`sigv4/iamsts` pair) for its `user_id`
log attribute and billing metric, so add it as a server interceptor and
attribution comes for free.

## Behavior to know about

- **Rejections are Twirp errors** (JSON), matching the local `sigv4a`
  middleware: missing Authorization → `unauthenticated` (401); clock skew,
  scope mismatch, missing signed `x-amz-region-set`/`host`, oversized or
  streaming bodies → `invalid_argument` (400); unknown key, disabled key, and
  signature mismatch are indistinguishable to the client → `permission_denied`
  (403).
- **iamd outages fail closed as 500**, never as a denial, and only bite on
  cold keys — warm traffic keeps verifying from cache.
- **Revocation latency is bounded by iamd's `-signing-key-cache-ttl`**
  (default 5m): a disabled key keeps verifying here until its cached entry
  expires, then starts failing.
- **Keys have no date scope**, unlike the classic derived-key chain: a
  cached public key stays valid for the credential's whole lifetime, not
  until UTC midnight, so there is no once-a-day refetch. A key is only
  refetched when its `cache_until` lapses or after a remembered refusal's
  `NegativeTTL` expires.
- A leaked `GetPublicKey` response contains only public material: an
  attacker who reads it can verify signatures, the same as anyone who already
  has the credential's access key id, but can never mint one.
- The request body is buffered (up to `MaxBodySize`) to check
  `x-amz-content-sha256`, then reset, so downstream handlers read it
  normally.

## Replay protection

SigV4A authenticates the sender by proving possession of the signing key,
but it does NOT prevent replay. Any request whose `X-Amz-Date` is within
`MaxClockSkew` (15 minutes by default) verifies a second time, so an
eavesdropper who sniffs a single valid request can replay it for the
duration of the window. If your API needs request freshness, layer your own
nonce, sequence number, or single-use challenge on top of this middleware
and reject ids you have already seen within the window.
