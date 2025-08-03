package logging

import (
	"context"
	"log/slog"
	"slices"
	"strings"
)

// FilteringHandler wraps an slog.Handler and provides filtering capabilities
type FilteringHandler struct {
	handler slog.Handler
	filters []LogFilter
}

// LogFilter defines a function that determines whether a log record should be processed
type LogFilter func(ctx context.Context, r slog.Record) bool

// NewFilteringHandler creates a new filtering handler with the given base handler and filters
func NewFilteringHandler(handler slog.Handler, filters ...LogFilter) *FilteringHandler {
	return &FilteringHandler{
		handler: handler,
		filters: filters,
	}
}

// Enabled implements slog.Handler
func (h *FilteringHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle implements slog.Handler and applies all filters before delegating to the base handler
func (h *FilteringHandler) Handle(ctx context.Context, r slog.Record) error {
	// Apply all filters - if any filter returns false, skip the log
	for _, filter := range h.filters {
		if !filter(ctx, r) {
			return nil // Skip this log record
		}
	}
	return h.handler.Handle(ctx, r)
}

// WithAttrs implements slog.Handler
func (h *FilteringHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &FilteringHandler{
		handler: h.handler.WithAttrs(attrs),
		filters: h.filters,
	}
}

// WithGroup implements slog.Handler
func (h *FilteringHandler) WithGroup(name string) slog.Handler {
	return &FilteringHandler{
		handler: h.handler.WithGroup(name),
		filters: h.filters,
	}
}

// AddFilter adds a new filter to the handler
func (h *FilteringHandler) AddFilter(filter LogFilter) {
	h.filters = append(h.filters, filter)
}

// Common filter functions

// FilterByMessage filters logs that contain any of the specified message substrings
func FilterByMessage(contains ...string) LogFilter {
	return func(ctx context.Context, r slog.Record) bool {
		msg := r.Message
		for _, substr := range contains {
			if strings.Contains(msg, substr) {
				return false // Filter out (skip) this log
			}
		}
		return true // Allow this log
	}
}

// FilterByMessageAllow only allows logs that contain any of the specified message substrings
func FilterByMessageAllow(contains ...string) LogFilter {
	return func(ctx context.Context, r slog.Record) bool {
		msg := r.Message
		for _, substr := range contains {
			if strings.Contains(msg, substr) {
				return true // Allow this log
			}
		}
		return false // Filter out (skip) this log
	}
}

// FilterByLevel filters out logs below the specified level
func FilterByLevel(minLevel slog.Level) LogFilter {
	return func(ctx context.Context, r slog.Record) bool {
		return r.Level >= minLevel
	}
}

// FilterByAttribute filters logs based on the presence and value of attributes
func FilterByAttribute(key string, values ...any) LogFilter {
	return func(ctx context.Context, r slog.Record) bool {
		var found bool
		var attrValue any

		r.Attrs(func(a slog.Attr) bool {
			if a.Key == key {
				found = true
				attrValue = a.Value.Any()
				return false // Stop iteration
			}
			return true // Continue iteration
		})

		if !found {
			return true // Allow if attribute not present
		}

		// If no specific values specified, filter out any logs with this attribute
		if len(values) == 0 {
			return false
		}

		// Check if the attribute value matches any of the allowed values
		for _, value := range values {
			if attrValue == value {
				return false // Filter out this log
			}
		}

		return true // Allow this log
	}
}

// FilterByAttributeAllow only allows logs with specific attribute values
func FilterByAttributeAllow(key string, values ...any) LogFilter {
	return func(ctx context.Context, r slog.Record) bool {
		var found bool
		var attrValue any

		r.Attrs(func(a slog.Attr) bool {
			if a.Key == key {
				found = true
				attrValue = a.Value.Any()
				return false // Stop iteration
			}
			return true // Continue iteration
		})

		if !found {
			return false // Filter out if attribute not present
		}

		// Check if the attribute value matches any of the allowed values
		for _, value := range values {
			if attrValue == value {
				return true // Allow this log
			}
		}

		return false // Filter out this log
	}
}

// FilterByComponent filters logs based on a "component" attribute
func FilterByComponent(allowedComponents ...string) LogFilter {
	return func(ctx context.Context, r slog.Record) bool {
		var component string
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == "component" {
				component = a.Value.String()
				return false
			}
			return true
		})

		if component == "" {
			return true // Allow logs without component
		}

		return slices.Contains(allowedComponents, component)
	}
}

// FilterNoiseHTTP filters out common HTTP noise logs
func FilterNoiseHTTP() LogFilter {
	noisyPaths := []string{
		"/health", "/healthz", "/metrics", "/favicon.ico",
		"/.well-known", "/robots.txt", "/.within/health",
	}

	return func(ctx context.Context, r slog.Record) bool {
		var path string
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == "path" || a.Key == "url" {
				path = a.Value.String()
				return false
			}
			return true
		})

		for _, noisyPath := range noisyPaths {
			if strings.Contains(path, noisyPath) {
				return false // Filter out noisy HTTP requests
			}
		}

		return true
	}
}

// CombineFilters combines multiple filters with AND logic (all must pass)
func CombineFilters(filters ...LogFilter) LogFilter {
	return func(ctx context.Context, r slog.Record) bool {
		for _, filter := range filters {
			if !filter(ctx, r) {
				return false
			}
		}
		return true
	}
}

// AnyFilter combines multiple filters with OR logic (any can pass)
func AnyFilter(filters ...LogFilter) LogFilter {
	return func(ctx context.Context, r slog.Record) bool {
		for _, filter := range filters {
			if filter(ctx, r) {
				return true
			}
		}
		return false
	}
}
