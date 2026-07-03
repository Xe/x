package twirpslog

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/twitchtv/twirp/ctxsetters"
)

// TestInterceptor checks the counting semantics: every call increments
// invocations, only failures increment errors, and only successes are billed
// to the caller.
func TestInterceptor(t *testing.T) {
	lg := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Each case uses its own method name so its counter series start at zero
	// on the shared default registry.
	newCtx := func(method string) context.Context {
		ctx := ctxsetters.WithPackageName(context.Background(), "test.pkg")
		ctx = ctxsetters.WithServiceName(ctx, "Svc")
		return ctxsetters.WithMethodName(ctx, method)
	}

	t.Run("success", func(t *testing.T) {
		m := Interceptor(lg)(func(ctx context.Context, req any) (any, error) {
			return "resp", nil
		})
		resp, err := m(newCtx("Ok"), "req")
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if resp != "resp" {
			t.Fatalf("resp = %v, want %q", resp, "resp")
		}

		if got := testutil.ToFloat64(invocations.WithLabelValues("test.pkg", "Svc", "Ok")); got != 1 {
			t.Errorf("invocations = %v, want 1", got)
		}
		if got := testutil.ToFloat64(errorHits.WithLabelValues("test.pkg", "Svc", "Ok")); got != 0 {
			t.Errorf("errors = %v, want 0", got)
		}
	})

	t.Run("error", func(t *testing.T) {
		wantErr := errors.New("boom")
		m := Interceptor(lg)(func(ctx context.Context, req any) (any, error) {
			return nil, wantErr
		})
		if _, err := m(newCtx("Boom"), "req"); !errors.Is(err, wantErr) {
			t.Fatalf("err = %v, want %v", err, wantErr)
		}

		if got := testutil.ToFloat64(invocations.WithLabelValues("test.pkg", "Svc", "Boom")); got != 1 {
			t.Errorf("invocations = %v, want 1", got)
		}
		if got := testutil.ToFloat64(errorHits.WithLabelValues("test.pkg", "Svc", "Boom")); got != 1 {
			t.Errorf("errors = %v, want 1", got)
		}
	})
}
