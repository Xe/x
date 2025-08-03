package logging

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Example usage of log filtering

func ExampleBasicFiltering() {
	// Initialize with HTTP noise filtering
	InitSlogWithHTTPFilter("INFO")

	// This will be filtered out
	slog.Info("HTTP request", "path", "/health", "method", "GET")

	// This will be logged
	slog.Info("HTTP request", "path", "/api/users", "method", "POST")
}

func ExampleComponentFiltering() {
	// Only log from specific components
	InitSlogWithComponentFilter("DEBUG", "auth", "database", "api")

	// This will be logged
	slog.Info("user authenticated", "component", "auth", "user_id", 123)

	// This will be filtered out
	slog.Info("background task completed", "component", "scheduler")
}

func ExampleAdvancedFiltering() {
	// Initialize with custom filtering configuration
	InitSlogWithFilters("DEBUG", &FilteringConfig{
		NoiseHTTP:         true,
		AllowedComponents: []string{"auth", "api"},
		BlockedMessages:   []string{"debug trace", "temp file"},
		MinLevel:          ptr(slog.LevelInfo), // Additional level filtering
	})
}

func ExampleCustomFilters() {
	// Initialize basic slog
	InitSlog("DEBUG")

	// Create a custom filter for sensitive operations
	sensitiveFilter := func(ctx context.Context, r slog.Record) bool {
		// Skip logs containing "password" or "secret"
		var skipLog bool
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == "password" || a.Key == "secret" || a.Key == "token" {
				skipLog = true
				return false
			}
			return true
		})
		return !skipLog
	}

	// Get a filtered logger for this specific use case
	logger := GetFilteredLogger(sensitiveFilter)

	// This will be filtered out
	logger.Info("login attempt", "username", "admin", "password", "secret123")

	// This will be logged
	logger.Info("login attempt", "username", "admin", "success", true)
}

func ExampleHTTPMiddleware() http.Handler {
	// Create a logger specifically for HTTP requests with noise filtering
	httpLogger := GetFilteredLogger(
		FilterNoiseHTTP(),
		FilterByAttribute("level", "DEBUG"), // Skip debug logs in HTTP context
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// This won't be logged if it's a health check endpoint
		httpLogger.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		// Process request...

		httpLogger.Info("HTTP response",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start),
			"status", 200,
		)
	})

	return handler
}

func ExampleConditionalFiltering() {
	// Initialize basic slog
	InitSlog("DEBUG")

	// Create a filter that only allows error logs during business hours
	businessHoursFilter := func(ctx context.Context, r slog.Record) bool {
		now := time.Now()
		hour := now.Hour()

		// During business hours (9 AM - 5 PM), only allow ERROR and above
		if hour >= 9 && hour <= 17 {
			return r.Level >= slog.LevelError
		}

		// Outside business hours, allow all logs
		return true
	}

	logger := GetFilteredLogger(businessHoursFilter)

	// Usage
	logger.Debug("This might be filtered during business hours")
	logger.Error("This will always be logged")
}

func ExampleMultipleFilterCombination() {
	// Combine multiple filters
	combinedFilter := CombineFilters(
		FilterNoiseHTTP(),
		FilterByComponent("api", "auth"),
		FilterByLevel(slog.LevelInfo),
	)

	logger := GetFilteredLogger(combinedFilter)

	// Only logs that pass ALL filters will be shown
	logger.Info("API request", "component", "api", "endpoint", "/users")
}

func ExampleFilterByOR() {
	// Allow logs from either auth component OR error level
	orFilter := AnyFilter(
		FilterByAttributeAllow("component", "auth"),
		FilterByLevel(slog.LevelError),
	)

	logger := GetFilteredLogger(orFilter)

	// Both of these will be logged
	logger.Info("authentication success", "component", "auth")
	logger.Error("database connection failed", "component", "database")
}

// Helper function
func ptr[T any](v T) *T {
	return &v
}
