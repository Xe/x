package slog

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"runtime"
	"strings"
	"testing"
	"testing/slogtest"
)

// TestContextHandlerConformance runs the standard library's handler
// conformance suite against ContextHandler wrapping a JSONHandler. With no
// context attributes present the wrapper delegates verbatim, so it must behave
// exactly like the handler it wraps.
func TestContextHandlerConformance(t *testing.T) {
	var buf bytes.Buffer
	h := NewContextHandler(slog.NewJSONHandler(&buf, nil))

	results := func() []map[string]any {
		var recs []map[string]any
		for _, line := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
			if len(line) == 0 {
				continue
			}
			var m map[string]any
			if err := json.Unmarshal(line, &m); err != nil {
				t.Fatalf("unmarshal %q: %v", line, err)
			}
			recs = append(recs, m)
		}
		return recs
	}

	if err := slogtest.TestHandler(h, results); err != nil {
		t.Fatalf("slogtest: %v", err)
	}
}

// decode logs one record through lg into a fresh buffer and returns the parsed
// JSON object.
func decode(t *testing.T, buf *bytes.Buffer, log func()) map[string]any {
	t.Helper()
	buf.Reset()
	log()
	var m map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &m); err != nil {
		t.Fatalf("unmarshal %q: %v", buf.String(), err)
	}
	return m
}

func TestContextHandlerAttrs(t *testing.T) {
	var buf bytes.Buffer
	lg := slog.New(NewContextHandler(slog.NewJSONHandler(&buf, nil)))

	t.Run("basic", func(t *testing.T) {
		ctx := ContextWithAttrs(context.Background(), slog.String("request_id", "abc"))
		m := decode(t, &buf, func() { lg.InfoContext(ctx, "hi") })
		if got := m["request_id"]; got != "abc" {
			t.Errorf("request_id = %v, want abc", got)
		}
	})

	t.Run("nested accumulates", func(t *testing.T) {
		ctx := ContextWithAttrs(context.Background(), slog.String("a", "1"))
		ctx = ContextWithAttrs(ctx, slog.String("b", "2"))
		m := decode(t, &buf, func() { lg.InfoContext(ctx, "hi") })
		if m["a"] != "1" || m["b"] != "2" {
			t.Errorf("got a=%v b=%v, want a=1 b=2", m["a"], m["b"])
		}
	})

	t.Run("siblings do not leak", func(t *testing.T) {
		parent := ContextWithAttrs(context.Background(), slog.String("shared", "p"))
		left := ContextWithAttrs(parent, slog.String("side", "left"))
		right := ContextWithAttrs(parent, slog.String("side", "right"))

		ml := decode(t, &buf, func() { lg.InfoContext(left, "hi") })
		mr := decode(t, &buf, func() { lg.InfoContext(right, "hi") })

		if ml["side"] != "left" {
			t.Errorf("left side = %v, want left", ml["side"])
		}
		if mr["side"] != "right" {
			t.Errorf("right side = %v, want right", mr["side"])
		}
		if _, ok := ml["shared"]; !ok {
			t.Error("left missing shared attr")
		}
	})

	t.Run("survives With", func(t *testing.T) {
		derived := lg.With("logger", "derived")
		ctx := ContextWithAttrs(context.Background(), slog.String("request_id", "abc"))
		m := decode(t, &buf, func() { derived.InfoContext(ctx, "hi") })
		if m["logger"] != "derived" {
			t.Errorf("logger = %v, want derived", m["logger"])
		}
		if m["request_id"] != "abc" {
			t.Errorf("request_id = %v, want abc", m["request_id"])
		}
	})

	t.Run("preserves source location", func(t *testing.T) {
		// AddSource on the inner handler must report the caller's site, not
		// somewhere inside ContextHandler.Handle or log/slog. This guards
		// against a refactor that re-logs through a fresh Logger in Handle
		// instead of threading the original record (which carries the PC).
		var sourceBuf bytes.Buffer
		src := slog.New(NewContextHandler(slog.NewJSONHandler(&sourceBuf, &slog.HandlerOptions{
			AddSource: true,
		})))
		ctx := ContextWithAttrs(context.Background(), slog.String("request_id", "abc"))

		_, _, wantLine, _ := runtime.Caller(0)
		src.InfoContext(ctx, "hi") // must stay on the line after runtime.Caller
		wantLine++

		var m map[string]any
		if err := json.Unmarshal(bytes.TrimSpace(sourceBuf.Bytes()), &m); err != nil {
			t.Fatalf("unmarshal %q: %v", sourceBuf.String(), err)
		}
		source, ok := m["source"].(map[string]any)
		if !ok {
			t.Fatalf("source = %v, want object", m["source"])
		}
		file, _ := source["file"].(string)
		if !strings.HasSuffix(file, "ctxhandler_test.go") {
			t.Errorf("source.file = %q, want the test file (not ctxhandler.go or log/slog)", file)
		}
		if line, _ := source["line"].(float64); int(line) != wantLine {
			t.Errorf("source.line = %v, want %d (the InfoContext call site)", line, wantLine)
		}
		if fn, _ := source["function"].(string); !strings.Contains(fn, "TestContextHandlerAttrs") {
			t.Errorf("source.function = %q, want the test function", fn)
		}
	})

	t.Run("top level under WithGroup", func(t *testing.T) {
		grouped := lg.WithGroup("g").With("inside", "yes")
		ctx := ContextWithAttrs(context.Background(), slog.String("request_id", "abc"))
		m := decode(t, &buf, func() { grouped.InfoContext(ctx, "hi", "own", 1) })

		// Context attr must be at the top level, not inside the group.
		if m["request_id"] != "abc" {
			t.Errorf("request_id = %v at top level, want abc", m["request_id"])
		}
		g, ok := m["g"].(map[string]any)
		if !ok {
			t.Fatalf("group g = %v, want object", m["g"])
		}
		if _, buried := g["request_id"]; buried {
			t.Error("request_id leaked into group g, want it at top level")
		}
		if g["inside"] != "yes" || g["own"] != float64(1) {
			t.Errorf("group contents = %v, want inside=yes own=1", g)
		}
	})
}
