package main

import (
	"log"
	"os"
	"runtime"

	"github.com/Xe/x/version"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	ver := os.Getenv("GO_VERSION")

	if ver == "" {
		log.Printf("ko: No GO_VERSION specified (wanted GO_VERSION=%[1]s) specified, assuming %[1]s", runtime.Version())
	}

	ver = "go" + ver

	version.Run(ver)
}
