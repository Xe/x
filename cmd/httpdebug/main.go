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
	bind   = flag.String("bind", ":3000", "TCP port to bind to")
	silent = flag.Bool("silent", false, "if set, don't log http headers")
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

		if !*silent {
			fmt.Println("---")
			r.Header.Write(io.MultiWriter(w, os.Stdout))
		} else {
			r.Header.Write(w)
		}

		if contains {
			fmt.Fprintln(w, "</pre></code>")
		}
	})

	slog.Info("listening", "url", "http://localhost"+*bind)
	log.Fatal(http.ListenAndServe(*bind, mux))
}
