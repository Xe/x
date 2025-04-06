package main

import (
	"flag"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	"within.website/x/internal"
)

var (
	bind    = flag.String("bind", ":3004", "port to listen on")
	proxyTo = flag.String("proxy-to", "http://localhost:5000", "where to reverse proxy to")
)

func main() {
	internal.HandleStartup()

	slog.Info("starting",
		"bind", *bind,
		"proxy-to", *proxyTo,
	)

	u, err := url.Parse(*proxyTo)
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(
		http.ListenAndServe(
			*bind,
			httputil.NewSingleHostReverseProxy(u),
		),
	)
}
