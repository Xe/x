package main

import (
	"flag"
	"log"
	"time"

	"github.com/Xe/x/internal"
	"github.com/Xe/x/internal/mainsa"
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
