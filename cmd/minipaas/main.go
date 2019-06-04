package main

import (
	"flag"
	"log"
	"strings"

	"within.website/x/internal"
	"within.website/x/internal/minipaas"
)

func main() {
	flag.Parse()
	internal.HandleStartup()

	err := minipaas.Exec(strings.Join(flag.Args(), " "))
	if err != nil {
		log.Fatal(err)
	}
}
