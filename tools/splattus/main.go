package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/Xe/tools/splattus/lib"
	"github.com/mgutz/ansi"
)

var (
	firstColor  = flag.String("firstColor", "208", "first color to use")
	secondColor = flag.String("secondColor", "21", "second color to use")
	prefixMsg   = flag.String("prefix", "Current Splatoon rotation: ", "prefix messsage with this")
)

func init() {
	flag.Parse()
}

func main() {
	sd, err := splattus.Lookup()
	if err != nil {
		log.Fatal(err)
	}

	data := sd[0]
	stage1 := data.Stages[0]
	stage2 := data.Stages[1]

	fmt.Printf(
		"%s%s and %s\n",
		*prefixMsg,
		ansi.Color(splattus.Englishify(stage1), *firstColor),
		ansi.Color(splattus.Englishify(stage2), *secondColor),
	)
}
