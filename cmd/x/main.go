package main

import (
	"context"
	"os"

	"github.com/google/subcommands"
	"within.website/x/cmd/x/cmd/grpchc"
	"within.website/x/cmd/x/cmd/importer/chatgpt"
	"within.website/x/cmd/x/cmd/importer/deepseek"
	"within.website/x/cmd/x/cmd/mi"
	"within.website/x/internal"
)

func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")

	subcommands.Register(&grpchc.GRPCHealth{}, "grpc")

	subcommands.Register(&chatgpt.ImportCmd{}, "ai")
	subcommands.Register(&deepseek.ImportCmd{}, "ai")

	// Switch tracker commands
	subcommands.Register(&mi.ListSwitches{}, "switch-tracker")
	subcommands.Register(&mi.Switch{}, "switch-tracker")
	subcommands.Register(&mi.WhoIsFront{}, "switch-tracker")

	// // Events
	// subcommands.Register(&miListEvents{}, "events")
	// subcommands.Register(&miAddEvent{}, "events")
	// subcommands.Register(&miRemoveEvent{}, "events")

	internal.HandleStartup()
	ctx := context.Background()

	os.Exit(int(subcommands.Execute(ctx)))
}
