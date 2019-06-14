// Command minipaas is a simple client for minipaas.xeserv.us. This is not useful without access to that server.
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
