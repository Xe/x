package main

import (
	"embed"
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/a-h/templ"
	"within.website/x/xess"
)

//go:generate go run github.com/a-h/templ/cmd/templ@latest generate

var (
	port     = flag.String("port", "23698", "TCP port to listen on")
	sockPath = flag.String("socket", "", "Unix socket to listen on")

	//go:embed quips.json
	content embed.FS

	quips []string
)

func main() {
	flag.Parse()

	fin, err := content.Open("quips.json")
	if err != nil {
		log.Fatal(err)
	}

	err = json.NewDecoder(fin).Decode(&quips)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	xess.Mount(mux)
	mux.Handle("/{$}", index())
	mux.Handle("/api", api())

	var l net.Listener
	if *sockPath != "" {
		os.Remove(*sockPath)
		l, err = net.Listen("unix", *sockPath)
	} else {
		l, err = net.Listen("tcp", ":"+*port)
	}

	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Handler: mux,
	}
	log.Fatal(srv.Serve(l))
}

func api() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")

		then := time.Date(2020, time.March, 1, 0, 0, 0, 0, time.UTC)
		now := time.Now().UTC()
		dur := now.Sub(then)

		json.NewEncoder(w).Encode(struct {
			Day  int    `json:"day"`
			Quip string `json:"quip"`
		}{
			Day:  int(dur.Hours()/24) + 1,
			Quip: quips[rand.Intn(len(quips))],
		})
	})
}

func index() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		then := time.Date(2020, time.March, 1, 0, 0, 0, 0, time.UTC)
		now := time.Now().UTC()
		dur := now.Sub(then)

		day := int(dur.Hours()/24) + 1
		quip := quips[rand.Intn(len(quips))]

		templ.Handler(xess.Base(
			"What day of 2020 is it?",
			headArea(),
			nil,
			body(day, quip),
			footer(),
		)).ServeHTTP(w, r)
	})
}
