package main

import (
	"os"

	"github.com/Xe/x/version"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	version.Run(os.Getenv("GO_VERSION"))
}
