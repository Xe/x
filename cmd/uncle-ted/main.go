package main

import (
	"embed"
	"flag"
	"log"
	"log/slog"
	"net/http"

	"github.com/NYTimes/gziphandler"
	"within.website/x/internal"
)

var (
	bind = flag.String("bind", ":2836", "TCP port to bind on")

	//go:embed bomb.txt.gz.gz
	kaboom []byte

	//go:embed bee-movie.txt
	static embed.FS
)

func main() {
	internal.HandleStartup()

	beeMovieHDLR, err := gziphandler.GzipHandlerWithOpts(gziphandler.CompressionLevel(9))
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/bee-movie", beeMovieHDLR(http.HandlerFunc(beeMovie)))
	http.HandleFunc("/gzip-bomb", gzipBomb)

	slog.Info("started up", "url", "http://localhost"+*bind)
	log.Fatal(http.ListenAndServe(*bind, nil))
}

func beeMovie(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, static, "bee-movie.txt")
}

func gzipBomb(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Transfer-Encoding", "gzip")

	w.WriteHeader(http.StatusOK)
	w.Write(kaboom)
}
