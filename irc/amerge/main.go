package main

import "flag"

var (
	fname = flag.String("file", "./services.db", "database to read from")
)

func main() {
	flag.Parse()

	_, err := NewDatabase(*fname)
	if err != nil {
		panic(err)
	}
}
