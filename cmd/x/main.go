package main

import (
	"context"
	"os"

	"github.com/google/subcommands"
	"within.website/x/internal"
)

func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")

	subcommands.Register(&sanguisugaAnimeList{}, "sanguisuga")
	subcommands.Register(&sanguisugaAnimeTrack{}, "sanguisuga")
	subcommands.Register(&sanguisugaTVList{}, "sanguisuga")
	subcommands.Register(&sanguisugaTVTrack{}, "sanguisuga")

	internal.HandleStartup()
	ctx := context.Background()

	os.Exit(int(subcommands.Execute(ctx)))
}
