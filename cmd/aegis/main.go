package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"path"
	"path/filepath"
)

var (
	hostport = flag.String("hostport", "[::]:31337", "TCP host:port to listen on")
	sockdir  = flag.String("sockdir", "./run", "directory full of unix sockets to monitor")
)

//go:embed "aegis.txt"
var core string

func main() {
	flag.Parse()

	fmt.Print(core)
	log.SetFlags(0)
	log.Printf("%s -> %s", *hostport, *sockdir)

	http.DefaultServeMux.HandleFunc("/", proxyToUnixSocket)

	log.Fatal(http.ListenAndServe(*hostport, nil))
}

func proxyToUnixSocket(w http.ResponseWriter, r *http.Request) {
	name := path.Base(r.URL.Path)

	fname := filepath.Join(*sockdir, name+".sock")
	_, err := os.Stat(fname)
	if os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	ts := &http.Transport{
		Dial: func(_, _ string) (net.Conn, error) {
			return net.Dial("unix", fname)
		},
		DisableKeepAlives: true,
	}

	rp := httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = "aegis"
			req.URL.Path = "/metrics"
			req.URL.RawPath = "/metrics"
		},
		Transport: ts,
	}
	rp.ServeHTTP(w, r)
}
