// +build ignore

package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/PonyvilleFM/aura/recording"
)

var (
	url   = flag.String("url", "", "url to record")
	fname = flag.String("fname", "", "filename to record to")
	debug = flag.Bool("debug", false, "debug mode")

	askedToDie bool
)

func main() {
	flag.Parse()

	r, err := recording.New(*url, *fname)
	if err != nil {
		log.Printf("%s -> %s: %v", *url, *fname, err)
		log.Fatal(err)
	}

	r.Debug = *debug

	go func() {
		log.Printf("Starting download of stream %s to %s", *url, *fname)
		err := r.Start()
		if err != nil {
			log.Fatal(err)
		}
	}()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)

		for _ = range c {
			if askedToDie {
				os.Exit(2)
			}

			log.Println("Stopping recording... (^C again to kill now)")
			r.Cancel()

			askedToDie = true
		}
	}()

	<-r.Done()
	log.Printf("stream %s recorded to %s", *url, *fname)
}
