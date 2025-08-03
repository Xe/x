// Package main demonstrates how to use log filtering in a real application
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"within.website/x/cmd/sakurajima/internal/logging"
)

func main() {
	// Example 1: Basic setup with HTTP noise filtering
	if len(os.Args) > 1 && os.Args[1] == "basic" {
		runBasicExample()
		return
	}

	// Example 2: Production-like setup with comprehensive filtering
	if len(os.Args) > 1 && os.Args[1] == "production" {
		runProductionExample()
		return
	}

	// Example 3: Development setup with minimal filtering
	if len(os.Args) > 1 && os.Args[1] == "development" {
		runDevelopmentExample()
		return
	}

	slog.Info("Usage: go run example.go [basic|production|development]")
}

func runBasicExample() {
	// Initialize with simple HTTP noise filtering
	logging.InitSlogWithHTTPFilter("INFO")

	slog.Info("Starting application", "version", "1.0.0")

	// Simulate HTTP requests
	slog.Info("HTTP request", "method", "GET", "path", "/api/users", "status", 200)
	slog.Info("HTTP request", "method", "GET", "path", "/health", "status", 200) // Filtered out
	slog.Info("HTTP request", "method", "POST", "path", "/api/login", "status", 201)

	slog.Info("Application ready")
}

func runProductionExample() {
	// Production configuration: strict filtering
	logging.InitSlogWithFilters("INFO", &logging.FilteringConfig{
		NoiseHTTP:         true,
		AllowedComponents: []string{"auth", "api", "database"},
		BlockedMessages:   []string{"debug", "trace", "temp"},
	})

	slog.Info("Production application starting", "environment", "production")

	// These will be logged (allowed components)
	slog.Info("Database connected", "component", "database", "host", "db.example.com")
	slog.Info("User authenticated", "component", "auth", "user_id", 12345)
	slog.Info("API request processed", "component", "api", "endpoint", "/users")

	// These will be filtered out
	slog.Info("Cache warmed", "component", "cache")          // Not in allowed components
	slog.Info("Debug trace information", "component", "api") // Contains blocked message
	slog.Info("Health check", "path", "/health")             // HTTP noise

	slog.Info("Production application ready")
}

func runDevelopmentExample() {
	// Development: no filtering, see everything
	logging.InitSlog("DEBUG")

	slog.Debug("Development mode enabled")
	slog.Info("Starting development server", "port", 8080)

	// All of these will be logged
	slog.Debug("Loading configuration", "config_file", "dev.yaml")
	slog.Info("Database connected", "component", "database")
	slog.Info("Health check endpoint", "path", "/health")
	slog.Warn("Using mock external service", "service", "payment")

	slog.Info("Development server ready")
}

// Example of creating context-specific loggers
func demonstrateContextualFiltering() {
	// Initialize base logging
	logging.InitSlog("DEBUG")

	// Create a logger for HTTP requests that filters noise
	httpLogger := logging.GetFilteredLogger(logging.FilterNoiseHTTP())

	// Create a logger for auth operations that filters sensitive data
	authLogger := logging.GetFilteredLogger(func(ctx context.Context, r slog.Record) bool {
		// Filter out logs containing sensitive keywords
		var containsSensitive bool
		r.Attrs(func(a slog.Attr) bool {
			switch a.Key {
			case "password", "token", "secret", "key":
				containsSensitive = true
				return false
			}
			return true
		})
		return !containsSensitive
	})

	// Create a logger for background jobs that only logs errors and warnings
	backgroundLogger := logging.GetFilteredLogger(
		logging.FilterByLevel(slog.LevelWarn),
	)

	// Usage examples
	httpLogger.Info("Processing request", "path", "/api/users", "method", "GET")
	httpLogger.Info("Health check", "path", "/health") // Filtered out

	authLogger.Info("Login attempt", "username", "alice", "success", true)
	authLogger.Info("Login attempt", "username", "bob", "password", "secret123") // Filtered out

	backgroundLogger.Debug("Background job progress", "percent", 50) // Filtered out
	backgroundLogger.Error("Background job failed", "error", "connection timeout")

}

// Example HTTP server with filtered logging
func createHTTPServerWithLogging() *http.Server {
	// Setup logging for HTTP server
	logging.InitSlogWithFilters("INFO", &logging.FilteringConfig{
		NoiseHTTP: true,
	})

	// Create HTTP logger
	httpLogger := logging.GetFilteredLogger(
		// Add request duration filtering - skip very fast requests
		func(ctx context.Context, r slog.Record) bool {
			var duration time.Duration
			r.Attrs(func(a slog.Attr) bool {
				if a.Key == "duration" {
					if d, ok := a.Value.Any().(time.Duration); ok {
						duration = d
					}
					return false
				}
				return true
			})

			// Skip requests faster than 1ms (likely health checks)
			return duration >= time.Millisecond
		},
	)

	mux := http.NewServeMux()

	// Add middleware for request logging
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Process request
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))

		// Log request (will be filtered based on our rules)
		httpLogger.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start),
			"status", 200,
		)
	})

	return &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
}
