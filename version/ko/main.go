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
		ver = runtime.Version()
	} else {
		ver = "go" + ver
	}

	version.Run(ver)
}
