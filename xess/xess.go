// Package xess vendors a copy of Xess and makes it available at /.xess/xess.css
//
// This is intended to be used as a vendored package in other projects.
package xess

import (
	"embed"
	"net/http"
)

//go:generate go run github.com/a-h/templ/cmd/templ@latest generate

var (
	//go:embed xess.css
	Static embed.FS
)

func init() {
	Mount(http.DefaultServeMux)
}

const URL = "/.xess/xess.css"

func Mount(mux *http.ServeMux) {
	mux.Handle("/.xess/", http.StripPrefix("/.xess/", http.FileServerFS(Static)))
}
