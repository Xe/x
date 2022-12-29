package main

import (
	"embed"
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"
)

var (
	port     = flag.String("port", "23698", "TCP port to listen on")
	sockPath = flag.String("socket", "", "Unix socket to listen on")

	//go:embed templates/* quips.json
	content embed.FS

	quips []string
)

func main() {
	flag.Parse()

	tmpl := template.Must(template.ParseFS(content, "templates/index.html"))

	fin, err := content.Open("quips.json")
	if err != nil {
		log.Fatal(err)
	}

	err = json.NewDecoder(fin).Decode(&quips)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", index(tmpl))
	mux.Handle("/api", api(tmpl))

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

func api(tmpl *template.Template) http.Handler {
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

func index(tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")

		then := time.Date(2020, time.March, 1, 0, 0, 0, 0, time.UTC)
		now := time.Now().UTC()
		dur := now.Sub(then)

		tmpl.Execute(w, struct {
			Day  int
			Quip string
		}{
			Day:  int(dur.Hours()/24) + 1,
			Quip: quips[rand.Intn(len(quips))],
		})
	})
}
