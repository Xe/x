package sigv4

import (
	"context"

	iamv1 "within.website/x/gen/within/website/x/iam/v1"
	"within.website/x/web/middleware/authctx"
)

func withKeyID(ctx context.Context, id string) context.Context {
	return authctx.WithKeyID(ctx, id)
}

// KeyID returns the verified access key id stored by Middleware, if any. The
// value is stored via authctx, so it is also visible to sigv4a.KeyID and
// authctx.KeyID directly.
func KeyID(ctx context.Context) (string, bool) {
	return authctx.KeyID(ctx)
}

// WithUser stores the IAM user resolved for the verified caller in ctx. It is
// the local-verification counterpart to iamsts.Caller: middleware resolves the
// access key id from KeyID to its owning user and stashes it here so
// downstream code (logging interceptors, handlers) can attribute the call
// without re-querying the backing store. The value is stored via authctx, so
// it is also visible to sigv4a.User and authctx.User directly.
func WithUser(ctx context.Context, u *iamv1.User) context.Context {
	return authctx.WithUser(ctx, u)
}

// User returns the IAM user associated with the verified caller, if any. The
// bool is false when no middleware populated it.
func User(ctx context.Context) (*iamv1.User, bool) {
	return authctx.User(ctx)
}
