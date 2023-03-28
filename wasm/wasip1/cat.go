package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

func main() {
	flag.Usage = func() {
		fmt.Printf("%s <file>\n\nprints file to standard out\n", os.Args[0])
	}
	flag.Parse()

	if flag.NArg() != 1 {
		log.Fatalf("wanted 1 arg, got %#v", os.Args)
	}

	fin, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	defer fin.Close()

	_, err = io.Copy(os.Stdout, fin)
	if err != nil {
		log.Fatal(err)
	}
}
