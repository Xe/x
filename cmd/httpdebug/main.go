package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"

	"within.website/x/internal"
)

var (
	bind = flag.String("bind", ":3000", "TCP port to bind to")
)

func main() {
	internal.HandleStartup()

	slog.Info("listening", "url", "http://localhost"+*bind)
	log.Fatal(http.ListenAndServe(*bind, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("---")
			r.Write(io.MultiWriter(w, os.Stdout))
			fmt.Println("---")
		},
	)))
}
