package main

import (
	"os"

	"github.com/Syfaro/finch"
	_ "github.com/Syfaro/finch/commands/help"
	_ "github.com/Syfaro/finch/commands/info"
	_ "github.com/Syfaro/finch/commands/stats"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	f := finch.NewFinch(os.Getenv("TELEGRAM_TOKEN"))

	f.Start()
}
