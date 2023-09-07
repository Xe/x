package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"

	"github.com/antonmedv/expr"
)

var (
	minify  = flag.Bool("minify", false, "minify JSON?")
	noColor = flag.Bool("no-color", false, "disable color output?")
)

func main() {
	flag.Parse()

	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		*noColor = true
	}

	if flag.NArg() != 1 {
		log.Fatal("usage: eq <expression> < data.json")
	}

	program, err := expr.Compile(flag.Arg(0))
	if err != nil {
		log.Fatalf("can't compile program: %v", err)
	}

	dec := json.NewDecoder(os.Stdin)
	enc := json.NewEncoder(os.Stdout)

	if !*minify {
		enc.SetIndent("", "  ")
	}

	for {
		obj := map[string]any{}
		if err := dec.Decode(&obj); err != nil {
			if err == io.EOF {
				break
			}

			log.Fatalf("can't decode JSON: %v", err)
		}

		output, err := expr.Run(program, obj)
		if err != nil {
			log.Fatalf("can't run program: %v", err)
		}

		if err := enc.Encode(output); err != nil {
			log.Fatalf("can't write JSON: %v", err)
		}
	}
}
