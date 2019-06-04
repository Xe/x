//+build ignore

package main

import (
	"flag"
	"log"
	"net/http"

	"within.website/x/idp/idpmiddleware"
	"within.website/x/internal"
	"within.website/ln/ex"
)

var (
	port    = flag.String("port", "6060", "TCP port")
	server  = flag.String("server", "https://idp.christine.website", "idp server")
	me      = flag.String("me", "https://christine.website/", "self-identity")
	selfURL = flag.String("self-url", "http://127.0.0.1:6060/", "redirect URL for the self.")
)

func main() {
	internal.HandleStartup()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "success!", http.StatusOK)
	})

	log.Fatal(http.ListenAndServe(":"+*port, ex.HTTPLog(idpmiddleware.Protect(*server, *me, *selfURL)(http.DefaultServeMux))))
}
