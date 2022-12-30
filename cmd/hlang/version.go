package main

import (
	"fmt"
	"os"
)

const version = "1.0.0"

func dumpVersion() {
	fmt.Println("h programming language version", version)
	os.Exit(0)
}
