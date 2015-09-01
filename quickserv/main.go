package main

import (
	"flag"
	"net/http"
)

var (
	port = flag.String("port", "3000", "port to use")
)

func main() {
	flag.Parse()
	http.Handle("/", http.FileServer(http.Dir(".")))
	http.ListenAndServe(":"+*port, nil)
}
