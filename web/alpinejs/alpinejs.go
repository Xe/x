// Package alpinejs contains an embedded copy of alpine.js for responsive web apps.
package alpinejs

import (
	"embed"
	"net/http"

	"within.website/x/internal"
)

//go:generate go tool templ generate

var (
	//go:embed *.js
	Static embed.FS
)

const Version = "3.5.11"

// URL is the folder path where alpine.js is served from.
const URL = "/.within.website/x/web/alpinejs"

func Mount(mux *http.ServeMux) {
	hdlr := http.StripPrefix(URL, http.FileServer(http.FS(Static)))
	hdlr = internal.UnchangingCache(hdlr)

	mux.Handle(URL, hdlr)
}
