package main

import (
	"flag"
	"log"
	"net/http"
)

var (
	port = flag.String("port", "3000", "port to use")
	dir  = flag.String("dir", ".", "directory to serve")
)

func main() {
	flag.Parse()
	http.Handle("/", http.FileServer(http.Dir(*dir)))
	log.Printf("Serving %s on port %s", *dir, *port)
	http.ListenAndServe(":"+*port, nil)
}
