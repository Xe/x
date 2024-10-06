package main

import (
	"fmt"
	"log/slog"
)

type flySlogger struct{}

func (flySlogger) Debug(v ...any) {
	slog.Debug("fly logs", "vals", fmt.Sprint(v...))
}

func (flySlogger) Debugf(format string, v ...any) {
	slog.Debug("fly logs", "vals", fmt.Sprintf(format, v...))
}
