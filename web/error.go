// Package web is a simple collection of high-level error and transport types
// that I end up using over and over.
package web

import (
	"fmt"
	"net/url"

	"within.website/ln"
)

// Error is an API error.
type Error struct {
	WantStatus, GotStatus int
	URL                   *url.URL
	Method                string
	ResponseBody          string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s %s: wanted status code %d, got: %d: %v", e.Method, e.URL, e.WantStatus, e.GotStatus, e.ResponseBody)
}

// F ields for logging.
func (e Error) F() ln.F {
	return ln.F{
		"err_want_status":   e.WantStatus,
		"err_got_status":    e.GotStatus,
		"err_url":           e.URL,
		"err_method":        e.Method,
		"err_response_body": e.ResponseBody,
	}
}
