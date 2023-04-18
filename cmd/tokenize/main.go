package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	gpt3encoder "github.com/samber/go-gpt-3-encoder"
	"within.website/x/internal"
)

var (
	decode = flag.Bool("decode", false, "if true, decode instead of encode")
)

func main() {
	internal.HandleStartup()

	enc, err := gpt3encoder.NewEncoder()
	if err != nil {
		log.Fatal(err)
	}

	if *decode {
		var tokens []int
		if err := json.NewDecoder(os.Stdin).Decode(&tokens); err != nil {
			log.Fatal(err)
		}

		fmt.Fprintln(os.Stdout, enc.Decode(tokens))
		return
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	tokens, err := enc.Encode(string(data))
	if err != nil {
		log.Fatal(err)
	}

	if err := json.NewEncoder(os.Stdout).Encode(&tokens); err != nil {
		log.Fatal(err)
	}
}
