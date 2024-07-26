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

	// Switch tracker
	subcommands.Register(&miListSwitches{}, "switch-tracker")
	subcommands.Register(&miSwitch{}, "switch-tracker")
	subcommands.Register(&miWhoIsFront{}, "switch-tracker")

	// Events
	subcommands.Register(&miListEvents{}, "events")
	subcommands.Register(&miAddEvent{}, "events")
	subcommands.Register(&miRemoveEvent{}, "events")

	// Sanguisuga
	subcommands.Register(&sanguisugaAnimeList{}, "sanguisuga")
	subcommands.Register(&sanguisugaAnimeTrack{}, "sanguisuga")
	subcommands.Register(&sanguisugaTVList{}, "sanguisuga")
	subcommands.Register(&sanguisugaTVTrack{}, "sanguisuga")

	internal.HandleStartup()
	ctx := context.Background()

	os.Exit(int(subcommands.Execute(ctx)))
}
