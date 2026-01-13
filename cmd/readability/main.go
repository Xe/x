package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"

	"codeberg.org/readeck/go-readability/v2"
	"within.website/x/internal"
)

func main() {
	internal.HandleStartup()

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "usage: %s <url>\n", os.Args[0])
		os.Exit(2)
	}

	pageURL := flag.Arg(0)
	u, err := url.Parse(pageURL)
	if err != nil {
		log.Fatal("can't parse URL:", err)
	}

	art, err := readability.FromReader(os.Stdin, u)
	if err != nil {
		log.Fatal("can't parse article:", err)
	}

	fmt.Fprintf(os.Stdout, "<html><body><h1>%s</h1>\n", art.Title())

	if err := art.RenderHTML(os.Stdout); err != nil {
		log.Fatal("can't render simplified HTML:", err)
	}

	fmt.Fprintf(os.Stdout, "</body></html>\n")
}
