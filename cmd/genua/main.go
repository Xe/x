// Command genua generates an example user agent.
package main

import (
	"flag"
	"fmt"
	"log"

	"within.website/x/web/useragent"
)

func main() {
	flag.Parse()

	if flag.NArg() != 2 {
		log.Fatal("usage: genua <prefix> <infoURL>")
	}

	fmt.Println(useragent.GenUserAgent(flag.Arg(0), flag.Arg(1)))
}
