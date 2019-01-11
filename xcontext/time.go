package xcontext

import (
	"context"
	"time"
)

type ctxKey int

const timeKkey ctxKey = iota

// WithTime stores a mock time in a context for testing timeouts and other operations.
func WithTime(ctx context.Context, t time.Time) context.Context {
	return context.WithValue(ctx, timeKey, t)
}

func TimeFuncFrom(ctx context.Context) func() time.Time {
	t, ok := ctx.Value(timeKey).(time.Time)

	if !ok {
		return time.Now
	}

	return func() time.Time { return t }
}
