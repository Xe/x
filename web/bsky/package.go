package bsky

import (
	"fmt"

	"within.website/ln"
)

type Error struct {
	ErrorKind string `json:"error"`
	Message   string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("bsky: %s: %s", e.ErrorKind, e.Message)
}

func (e Error) F() ln.F {
	return ln.F{
		"error":   e.ErrorKind,
		"message": e.Message,
	}
}
