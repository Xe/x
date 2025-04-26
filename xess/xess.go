// Package xess vendors a copy of Xess and makes it available at /.xess/xess.css
//
// This is intended to be used as a vendored package in other projects.
package xess

import (
	"embed"
	"net/http"
	"path/filepath"

	"within.website/x"
	"within.website/x/internal"
)

//go:generate go tool templ generate
//go:generate npm ci
//go:generate npm run build

var (
	//go:embed xess.min.css xess.css static
	Static embed.FS

	URL = "/.within.website/x/xess/xess.css"
)

func init() {
	Mount(http.DefaultServeMux)

	if x.Version != "devel" {
		URL = filepath.Join(filepath.Dir(URL), "xess.min.css")
	}

	URL = URL + "?cachebuster=" + x.Version
}

func Mount(mux *http.ServeMux) {
	mux.Handle("/.within.website/x/xess/", internal.UnchangingCache(http.StripPrefix("/.within.website/x/xess/", http.FileServerFS(Static))))
}
