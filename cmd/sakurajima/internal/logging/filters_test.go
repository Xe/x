package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

// testHandler captures log records for testing
type testHandler struct {
	records []slog.Record
	buf     *bytes.Buffer
}

func newTestHandler() *testHandler {
	buf := &bytes.Buffer{}
	return &testHandler{
		records: make([]slog.Record, 0),
		buf:     buf,
	}
}

func (h *testHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *testHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)

	// Also write to buffer for JSON parsing tests
	attrs := make(map[string]any)
	attrs["time"] = r.Time
	attrs["level"] = r.Level.String()
	attrs["msg"] = r.Message

	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	json.NewEncoder(h.buf).Encode(attrs)
	return nil
}

func (h *testHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h // Simplified for testing
}

func (h *testHandler) WithGroup(name string) slog.Handler {
	return h // Simplified for testing
}

func (h *testHandler) Reset() {
	h.records = h.records[:0]
	h.buf.Reset()
}

func TestFilterByMessage(t *testing.T) {
	baseHandler := newTestHandler()
	filter := FilterByMessage("debug", "trace")
	filterHandler := NewFilteringHandler(baseHandler, filter)
	logger := slog.New(filterHandler)

	// These should be filtered out
	logger.Info("debug information")
	logger.Info("trace details")

	// This should pass through
	logger.Info("user login successful")

	if len(baseHandler.records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(baseHandler.records))
	}

	if baseHandler.records[0].Message != "user login successful" {
		t.Errorf("Expected 'user login successful', got '%s'", baseHandler.records[0].Message)
	}
}

func TestFilterByMessageAllow(t *testing.T) {
	baseHandler := newTestHandler()
	filter := FilterByMessageAllow("error", "warning")
	filterHandler := NewFilteringHandler(baseHandler, filter)
	logger := slog.New(filterHandler)

	// These should pass through
	logger.Info("error occurred")
	logger.Info("warning: deprecated")

	// This should be filtered out
	logger.Info("info message")

	if len(baseHandler.records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(baseHandler.records))
	}
}

func TestFilterByLevel(t *testing.T) {
	baseHandler := newTestHandler()
	filter := FilterByLevel(slog.LevelWarn)
	filterHandler := NewFilteringHandler(baseHandler, filter)
	logger := slog.New(filterHandler)

	// These should be filtered out
	logger.Debug("debug message")
	logger.Info("info message")

	// These should pass through
	logger.Warn("warning message")
	logger.Error("error message")

	if len(baseHandler.records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(baseHandler.records))
	}

	for _, record := range baseHandler.records {
		if record.Level < slog.LevelWarn {
			t.Errorf("Expected level >= WARN, got %s", record.Level)
		}
	}
}

func TestFilterByAttribute(t *testing.T) {
	baseHandler := newTestHandler()
	filter := FilterByAttribute("env", "test")
	filterHandler := NewFilteringHandler(baseHandler, filter)
	logger := slog.New(filterHandler)

	// This should be filtered out
	logger.Info("test message", "env", "test")

	// These should pass through
	logger.Info("prod message", "env", "production")
	logger.Info("message without env")

	if len(baseHandler.records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(baseHandler.records))
	}
}

func TestFilterByAttributeAllow(t *testing.T) {
	baseHandler := newTestHandler()
	filter := FilterByAttributeAllow("component", "auth", "api")
	filterHandler := NewFilteringHandler(baseHandler, filter)
	logger := slog.New(filterHandler)

	// These should pass through
	logger.Info("auth message", "component", "auth")
	logger.Info("api message", "component", "api")

	// These should be filtered out
	logger.Info("db message", "component", "database")
	logger.Info("message without component")

	if len(baseHandler.records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(baseHandler.records))
	}
}

func TestFilterByComponent(t *testing.T) {
	baseHandler := newTestHandler()
	filter := FilterByComponent("auth", "api")
	filterHandler := NewFilteringHandler(baseHandler, filter)
	logger := slog.New(filterHandler)

	// These should pass through
	logger.Info("auth message", "component", "auth")
	logger.Info("api message", "component", "api")
	logger.Info("message without component") // Should pass (no component)

	// This should be filtered out
	logger.Info("db message", "component", "database")

	if len(baseHandler.records) != 3 {
		t.Errorf("Expected 3 records, got %d", len(baseHandler.records))
	}
}

func TestFilterNoiseHTTP(t *testing.T) {
	baseHandler := newTestHandler()
	filter := FilterNoiseHTTP()
	filterHandler := NewFilteringHandler(baseHandler, filter)
	logger := slog.New(filterHandler)

	// These should be filtered out
	logger.Info("request", "path", "/health")
	logger.Info("request", "path", "/metrics")
	logger.Info("request", "path", "/favicon.ico")
	logger.Info("request", "url", "https://example.com/.well-known/something")

	// This should pass through
	logger.Info("request", "path", "/api/users")

	if len(baseHandler.records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(baseHandler.records))
	}

	if baseHandler.records[0].Message != "request" {
		t.Errorf("Expected 'request', got '%s'", baseHandler.records[0].Message)
	}
}

func TestCombineFilters(t *testing.T) {
	baseHandler := newTestHandler()
	combined := CombineFilters(
		FilterByLevel(slog.LevelInfo),
		FilterByAttributeAllow("component", "auth"), // Use FilterByAttributeAllow instead
	)
	filterHandler := NewFilteringHandler(baseHandler, combined)
	logger := slog.New(filterHandler)

	// Should pass through (INFO level AND auth component)
	logger.Info("auth success", "component", "auth")

	// Should be filtered out (DEBUG level)
	logger.Debug("auth debug", "component", "auth")

	// Should be filtered out (wrong component)
	logger.Info("api call", "component", "api")

	// Should be filtered out (no component)
	logger.Info("general message")

	if len(baseHandler.records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(baseHandler.records))
	}
}

func TestAnyFilter(t *testing.T) {
	baseHandler := newTestHandler()
	anyFilter := AnyFilter(
		FilterByLevel(slog.LevelError),
		FilterByAttributeAllow("component", "auth"),
	)
	filterHandler := NewFilteringHandler(baseHandler, anyFilter)
	logger := slog.New(filterHandler)

	// Should pass through (ERROR level)
	logger.Error("database error", "component", "database")

	// Should pass through (auth component)
	logger.Info("auth success", "component", "auth")

	// Should be filtered out (INFO level AND not auth component)
	logger.Info("api call", "component", "api")

	if len(baseHandler.records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(baseHandler.records))
	}
}

func TestMultipleFilters(t *testing.T) {
	baseHandler := newTestHandler()
	filterHandler := NewFilteringHandler(baseHandler,
		FilterByLevel(slog.LevelInfo),
		FilterByMessage("debug"),
		FilterByComponent("auth", "api"),
	)
	logger := slog.New(filterHandler)

	// Should pass through
	logger.Info("auth success", "component", "auth")

	// Should be filtered out by level
	logger.Debug("auth debug", "component", "auth")

	// Should be filtered out by message content
	logger.Info("debug trace", "component", "auth")

	// Should be filtered out by component
	logger.Info("database query", "component", "database")

	if len(baseHandler.records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(baseHandler.records))
	}
}

func TestAddFilter(t *testing.T) {
	baseHandler := newTestHandler()
	filterHandler := NewFilteringHandler(baseHandler)
	logger := slog.New(filterHandler)

	// Initially no filters, should pass through
	logger.Info("test message")

	if len(baseHandler.records) != 1 {
		t.Errorf("Expected 1 record before adding filter, got %d", len(baseHandler.records))
	}

	baseHandler.Reset()

	// Add a filter
	filterHandler.AddFilter(FilterByMessage("test"))

	// Now should be filtered out
	logger.Info("test message")
	logger.Info("other message")

	if len(baseHandler.records) != 1 {
		t.Errorf("Expected 1 record after adding filter, got %d", len(baseHandler.records))
	}

	if baseHandler.records[0].Message != "other message" {
		t.Errorf("Expected 'other message', got '%s'", baseHandler.records[0].Message)
	}
}

func TestWithAttrsAndWithGroup(t *testing.T) {
	baseHandler := newTestHandler()
	filter := FilterByComponent("auth")
	filterHandler := NewFilteringHandler(baseHandler, filter)

	// Test WithAttrs
	withAttrsHandler := filterHandler.WithAttrs([]slog.Attr{
		slog.String("service", "web"),
	})

	// Test WithGroup
	withGroupHandler := withAttrsHandler.WithGroup("request")

	logger := slog.New(withGroupHandler)

	// Should pass through
	logger.Info("auth success", "component", "auth", "user_id", 123)

	// Should be filtered out
	logger.Info("api call", "component", "api", "endpoint", "/users")

	if len(baseHandler.records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(baseHandler.records))
	}
}

func BenchmarkFilteringHandler(b *testing.B) {
	baseHandler := newTestHandler()
	filterHandler := NewFilteringHandler(baseHandler,
		FilterByLevel(slog.LevelInfo),
		FilterByComponent("auth", "api"),
		FilterNoiseHTTP(),
	)
	logger := slog.New(filterHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("test message",
			"component", "auth",
			"user_id", 123,
			"path", "/api/users",
			"method", "GET",
		)
	}
}

func BenchmarkNoFiltering(b *testing.B) {
	baseHandler := newTestHandler()
	logger := slog.New(baseHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("test message",
			"component", "auth",
			"user_id", 123,
			"path", "/api/users",
			"method", "GET",
		)
	}
}
