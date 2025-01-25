package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"within.website/x/internal"
)

var (
	bind = flag.String("bind", ":3000", "TCP port to bind to")
)

func main() {
	internal.HandleStartup()

	mux := http.NewServeMux()

	mux.HandleFunc("/.within/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		contains := strings.Contains(r.Header.Get("Accept"), "text/html")

		if contains {
			w.Header().Add("Content-Type", "text/html")
			fmt.Fprint(w, "<pre id=\"main\"><code>")
		}

		fmt.Println("---")
		r.Write(io.MultiWriter(w, os.Stdout))

		if contains {
			fmt.Fprintln(w, "</pre></code>")
		}
	})

	slog.Info("listening", "url", "http://localhost"+*bind)
	log.Fatal(http.ListenAndServe(*bind, mux))
}
