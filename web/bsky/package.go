package bsky

import (
	"fmt"
	"log/slog"
)

type Error struct {
	ErrorKind string `json:"error"`
	Message   string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("bsky: %s: %s", e.ErrorKind, e.Message)
}

func (e Error) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("error", e.ErrorKind),
		slog.String("msg", e.Message),
	)
}
