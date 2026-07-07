package sigv4a

import (
	"context"

	iamv1 "within.website/x/gen/within/website/x/iam/v1"
)

type ctxKey struct{}

func withKeyID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// KeyID returns the verified access key id stored by Middleware, if any.
func KeyID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(ctxKey{}).(string)
	return id, ok
}

type userCtxKey struct{}

// WithUser stores the IAM user resolved for the verified caller in ctx. It is
// the local-verification counterpart to iamsts.Caller: middleware resolves the
// access key id from KeyID to its owning user and stashes it here so downstream
// code (logging interceptors, handlers) can attribute the call without
// re-querying the backing store.
func WithUser(ctx context.Context, u *iamv1.User) context.Context {
	return context.WithValue(ctx, userCtxKey{}, u)
}

// User returns the IAM user associated with the verified caller, if any. The
// bool is false when no middleware populated it.
func User(ctx context.Context) (*iamv1.User, bool) {
	u, ok := ctx.Value(userCtxKey{}).(*iamv1.User)
	return u, ok
}
