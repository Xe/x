# Integrating iamsts with a Twirp API

`iamsts` authenticates incoming SigV4-signed requests locally: it fetches the
derived signing key for each credential scope from iamd's `SigningKeyService`
once, caches it for the server-advised TTL, and recomputes signatures itself.
When the cache is warm there are **zero** IAM RPCs on the request path.

## Prerequisites

- An IAM credential for your service (the "verifier credential"), used only to
  authenticate the `GetSigningKey` fetches to iamd. Create one with
  `KeyService/CreateKey` or take it from iamd's bootstrap output.
- The iamd base URL.
- The fleet-wide SigV4 scope. Every client and verifier must sign with the
  same region/service that iamd runs with (`-region`, `-service`, defaults
  `us-east-1` / `iam`); iamd refuses to issue keys for any other scope.

## Wiring

Build a signed HTTP client with `sigv4client`, construct the verifier with
`iamsts.New`, and wrap each Twirp service handler when registering it on the
mux:

```go
import (
	"net/http"

	"github.com/twitchtv/twirp"

	widgetsv1 "within.website/x/gen/within/website/x/widgets/v1" // your service
	"within.website/x/web/middleware/sigv4/iamsts"
	"within.website/x/web/middleware/sigv4/sigv4client"
)

func newMux() (*http.ServeMux, error) {
	// The transport that signs GetSigningKey calls to iamd with this
	// service's own IAM credential.
	rt, err := sigv4client.NewSigV4RoundTripper(&sigv4client.Config{
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
		Region:      "us-east-1",
		Service:     "iam",
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

`Config` also accepts `NegativeTTL` (how long a refused key is remembered,
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

`twirp/twirpslog.Interceptor` already reads `iamsts.Caller` for its `user_id`
log attribute and billing metric, so add it as a server interceptor and
attribution comes for free.

## Behavior to know about

- **Rejections are Twirp errors** (JSON), matching the local `sigv4`
  middleware: missing Authorization → `unauthenticated` (401); clock skew,
  scope mismatch, oversized or streaming bodies → `invalid_argument` (400);
  unknown key, disabled key, and signature mismatch are indistinguishable to
  the client → `permission_denied` (403).
- **iamd outages fail closed as 500**, never as a denial, and only bite on
  cold scopes — warm traffic keeps verifying from cache.
- **Revocation latency is bounded by iamd's `-signing-key-cache-ttl`**
  (default 5m): a disabled key keeps verifying here until its cached entry
  expires, then starts failing.
- **UTC midnight rolls the credential scope date**, so the first request per
  key after midnight triggers one fresh `GetSigningKey`; concurrent misses
  for the same scope collapse into a single RPC.
- The request body is buffered (up to `MaxBodySize`) to check
  `x-amz-content-sha256`, then reset, so downstream handlers read it
  normally.
