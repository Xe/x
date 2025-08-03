package logging

import (
	"fmt"
	"log/slog"
	"os"
)

func InitSlog(level string) *slog.Logger {
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

	return slog.New(baseHandler)
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
