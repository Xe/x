package twirpslog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/twitchtv/twirp/ctxsetters"
	xslog "within.website/x/internal/slog"
)

// TestInterceptor checks the counting semantics: every call increments
// invocations, only failures increment errors, and only successes are billed
// to the caller.
func TestInterceptor(t *testing.T) {
	lg := slog.New(slog.DiscardHandler)

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

// TestInterceptorContextAttrs proves that the interceptor attaches the call's
// package, service, and method to the context, so a service method that logs
// via slog.InfoContext surfaces them without threading a logger by hand.
func TestInterceptorContextAttrs(t *testing.T) {
	var buf bytes.Buffer
	lg := slog.New(xslog.NewContextHandler(slog.NewJSONHandler(&buf, nil)))

	prev := slog.Default()
	slog.SetDefault(lg)
	t.Cleanup(func() { slog.SetDefault(prev) })

	ctx := ctxsetters.WithPackageName(context.Background(), "test.pkg")
	ctx = ctxsetters.WithServiceName(ctx, "Svc")
	ctx = ctxsetters.WithMethodName(ctx, "Attrs")

	m := Interceptor(lg)(func(ctx context.Context, req any) (any, error) {
		// A handler deep inside the call logs through the default logger with
		// only the request context in hand.
		slog.InfoContext(ctx, "inside handler")
		return "resp", nil
	})
	if _, err := m(ctx, "req"); err != nil {
		t.Fatalf("err = %v", err)
	}

	var handlerLine map[string]any
	for _, line := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal(line, &rec); err != nil {
			t.Fatalf("unmarshal %q: %v", line, err)
		}
		if rec["msg"] == "inside handler" {
			handlerLine = rec
		}
	}
	if handlerLine == nil {
		t.Fatalf("no %q line in output:\n%s", "inside handler", buf.String())
	}

	for k, want := range map[string]string{
		"package": "test.pkg",
		"service": "Svc",
		"method":  "Attrs",
	} {
		if got := handlerLine[k]; got != want {
			t.Errorf("%s = %v, want %q", k, got, want)
		}
	}
}
