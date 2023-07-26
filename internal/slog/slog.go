// Package slog is my set of wrappers around package slog.
package slog

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"golang.org/x/exp/slog"
)

var (
	slogLevel = flag.String("slog-level", "INFO", "log level")

	lock sync.Mutex

	// The current slog handler.
	Handler slog.Handler
)

func Init() {
	var programLevel slog.Level
	if err := (&programLevel).UnmarshalText([]byte(*slogLevel)); err != nil {
		fmt.Fprintf(os.Stderr, "invalid log level %s: %v, using info\n", *slogLevel, err)
		programLevel = slog.LevelInfo
	}

	h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		Level:     programLevel,
	})
	slog.SetDefault(slog.New(h))

	lock.Lock()
	defer lock.Unlock()
	Handler = h
}
