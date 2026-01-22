// Package web is a simple collection of high-level error and transport types
// that I end up using over and over.
package web

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

// NewError creates an Error based on an expected HTTP status code vs data populated
// from an HTTP response.
//
// This consumes the body of the HTTP response.
func NewError(wantStatusCode int, resp *http.Response) error {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()

	loc := resp.Request.URL

	return &Error{
		WantStatus:   wantStatusCode,
		GotStatus:    resp.StatusCode,
		URL:          loc,
		Method:       resp.Request.Method,
		ResponseBody: string(data),
	}
}

// Error is a web response error. Use this when API calls don't work out like you wanted them to.
type Error struct {
	WantStatus, GotStatus int
	URL                   *url.URL
	Method                string
	ResponseBody          string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s %s: wanted status code %d, got: %d: %v", e.Method, e.URL, e.WantStatus, e.GotStatus, e.ResponseBody)
}

// LogValue formats this Error for slog.
func (e Error) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("want_status", e.WantStatus),
		slog.Int("got_status", e.GotStatus),
		slog.String("url", e.URL.String()),
		slog.String("method", e.Method),
		slog.String("body", e.ResponseBody),
	)
}
