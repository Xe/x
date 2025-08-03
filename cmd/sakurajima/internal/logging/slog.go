package logging

import (
	"fmt"
	"log/slog"
	"os"
)

// FilteringConfig holds configuration for log filtering
type FilteringConfig struct {
	// NoiseHTTP filters out common HTTP noise logs
	NoiseHTTP bool
	// AllowedComponents specifies which components to allow logs from
	AllowedComponents []string
	// BlockedMessages specifies message substrings to filter out
	BlockedMessages []string
	// AllowedMessages specifies message substrings to allow (if set, only these are allowed)
	AllowedMessages []string
	// MinLevel specifies minimum log level (in addition to the global level)
	MinLevel *slog.Level
}

func InitSlog(level string) *slog.Logger {
	return InitSlogWithFilters(level, nil)
}

func InitSlogWithFilters(level string, filterConfig *FilteringConfig) *slog.Logger {
	var programLevel slog.Level
	if err := (&programLevel).UnmarshalText([]byte(level)); err != nil {
		fmt.Fprintf(os.Stderr, "invalid log level %s: %v, using info\n", level, err)
		programLevel = slog.LevelInfo
	}

	leveler := &slog.LevelVar{}
	leveler.Set(programLevel)

	baseHandler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		Level:     leveler,
	})

	// If no filter config provided, use the base handler directly
	if filterConfig == nil {
		return slog.New(baseHandler)
	}

	// Build filters based on configuration
	var filters []LogFilter

	if filterConfig.NoiseHTTP {
		filters = append(filters, FilterNoiseHTTP())
	}

	if len(filterConfig.AllowedComponents) > 0 {
		filters = append(filters, FilterByComponent(filterConfig.AllowedComponents...))
	}

	if len(filterConfig.BlockedMessages) > 0 {
		filters = append(filters, FilterByMessage(filterConfig.BlockedMessages...))
	}

	if len(filterConfig.AllowedMessages) > 0 {
		filters = append(filters, FilterByMessageAllow(filterConfig.AllowedMessages...))
	}

	if filterConfig.MinLevel != nil {
		filters = append(filters, FilterByLevel(*filterConfig.MinLevel))
	}

	// Create filtering handler with all configured filters
	var handler slog.Handler = baseHandler
	if len(filters) > 0 {
		handler = NewFilteringHandler(baseHandler, filters...)
	}

	return slog.New(handler)
}

// Common convenience functions

// InitSlogWithHTTPFilter initializes slog with HTTP noise filtering
func InitSlogWithHTTPFilter(level string) {
	InitSlogWithFilters(level, &FilteringConfig{
		NoiseHTTP: true,
	})
}

// InitSlogWithComponentFilter initializes slog with component-based filtering
func InitSlogWithComponentFilter(level string, allowedComponents ...string) {
	InitSlogWithFilters(level, &FilteringConfig{
		AllowedComponents: allowedComponents,
	})
}

// GetFilteredLogger returns a logger with additional filters applied
func GetFilteredLogger(filters ...LogFilter) *slog.Logger {
	currentHandler := slog.Default().Handler()

	// If current handler is already a FilteringHandler, add to its filters
	if fh, ok := currentHandler.(*FilteringHandler); ok {
		for _, filter := range filters {
			fh.AddFilter(filter)
		}
		return slog.New(fh)
	}

	// Otherwise, wrap the current handler
	return slog.New(NewFilteringHandler(currentHandler, filters...))
}
