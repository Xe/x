package main

import (
	"flag"
	"log"
	"time"

	"within.website/x/internal"
	"within.website/x/internal/mainsa"
)

func main() {
	flag.Parse()
	internal.HandleStartup()

	tn, err := mainsa.At(time.Now())
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s", tn)
}
