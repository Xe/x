// Package xess vendors a copy of Xess and makes it available at /.xess/xess.css
//
// This is intended to be used as a vendored package in other projects.
package xess

import (
	"embed"
	"net/http"

	"within.website/x/internal"
)

//go:generate go run github.com/a-h/templ/cmd/templ@latest generate

var (
	//go:embed xess.css static
	Static embed.FS
)

func init() {
	Mount(http.DefaultServeMux)
}

const URL = "/.within.website/x/xess/xess.css"

func Mount(mux *http.ServeMux) {
	mux.Handle("/.within.website/x/xess/", internal.UnchangingCache(http.StripPrefix("/.within.website/x/xess/", http.FileServerFS(Static))))
}
