package sigv4

import "context"

type ctxKey struct{}

func withKeyID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// KeyID returns the verified access key id stored by Middleware, if any.
func KeyID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(ctxKey{}).(string)
	return id, ok
}
