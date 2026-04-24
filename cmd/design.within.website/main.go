// Command design.within.website is a living demo of xess, the within.website
// design system. It renders every component, explains why it exists, and
// shows the source you'd paste to use it.
package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/a-h/templ"
	"within.website/x/internal"
	"within.website/x/xess"
)

var bind = flag.String("bind", ":2135", "HTTP bind address")

//go:generate go tool templ generate
//go:generate go tool templ fmt .
//go:generate go fmt ./...

func main() {
	internal.HandleStartup()

	mux := http.NewServeMux()
	xess.Mount(mux)

	mux.Handle("GET /static/", http.FileServerFS(staticFS))

	mux.Handle("/{$}", templ.Handler(
		xess.Base(
			"xess — the within.website design system",
			demoStyleLink(),
			topNav(),
			Index(),
			footer(),
		),
	))

	mux.Handle("/", templ.Handler(
		xess.Simple("Not found", notFound()),
		templ.WithStatus(http.StatusNotFound),
	))

	slog.Info("listening", "bind", *bind)
	if err := http.ListenAndServe(*bind, mux); err != nil {
		slog.Error("http server stopped", "err", err)
		os.Exit(1)
	}
}
