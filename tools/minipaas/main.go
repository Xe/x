package main

import (
	"flag"
	"log"
	"strings"

	"github.com/Xe/x/internal"
	"github.com/Xe/x/internal/minipaas"
)

func main() {
	flag.Parse()
	internal.HandleLicense()

	err := minipaas.Exec(strings.Join(flag.Args(), " "))
	if err != nil {
		log.Fatal(err)
	}
}
