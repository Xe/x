package main

import (
	"os"

	_ "github.com/Xe/x/tg/covfefe/commands"
	_ "github.com/joho/godotenv/autoload"
	"github.com/syfaro/finch"
	_ "github.com/syfaro/finch/commands/help"
	_ "github.com/syfaro/finch/commands/info"
)

func main() {
	f := finch.NewFinch(os.Getenv("TELEGRAM_TOKEN"))

	f.Start()
}
