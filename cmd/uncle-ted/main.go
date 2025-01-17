package main

import (
	_ "embed"
	"flag"
	"log"
	"log/slog"
	"net/http"

	"within.website/x/internal"
)

var (
	bind = flag.String("bind", ":2836", "TCP port to bind on")

	//go:embed bomb.txt.gz.gz
	kaboom []byte
)

func main() {
	internal.HandleStartup()

	http.HandleFunc("/", defenseHandler)

	slog.Info("started up", "url", "http://localhost"+*bind)
	log.Fatal(http.ListenAndServe(*bind, nil))
}

func defenseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Transfer-Encoding", "gzip")

	w.WriteHeader(http.StatusOK)
	w.Write(kaboom)
}
