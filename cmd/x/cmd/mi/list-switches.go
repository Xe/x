package mi

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"
	"github.com/rodaine/table"

	mi "within.website/x/gen/within/website/x/mi/v1"
)

// miListSwitches implements the "list-switches" subcommand.
// ListSwitches implements the "list-switches" subcommand.
type ListSwitches struct {
	count int
	page  int
}

func (*ListSwitches) Name() string     { return "list-switches" }
func (*ListSwitches) Synopsis() string { return "List switches." }
func (*ListSwitches) Usage() string {
	return `list-switches [--count] [--page]:
List a number of past switches.
`
}

func (ls *ListSwitches) SetFlags(f *flag.FlagSet) {
	f.IntVar(&ls.count, "count", 10, "Number of switches to list.")
	f.IntVar(&ls.page, "page", 0, "Page of switches to list.")
}

func (ls *ListSwitches) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	// Initialise client.
	client, err := New()
	if err != nil {
		fmt.Printf("can't connect to mi %v\n", err)
		return subcommands.ExitFailure
	}
	cli := client.SwitchTracker

	resp, err := cli.ListSwitches(ctx, &mi.ListSwitchesReq{Count: int32(ls.count), Page: int32(ls.page)})
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitFailure
	}

	tbl := table.New("Started at", "Ended at", "Member")
	for _, sw := range resp.Switches {
		tbl.AddRow(sw.GetSwitch().GetStartedAt(), sw.Switch.GetEndedAt(), sw.GetMember().GetName())
	}

	tbl.Print()
	return subcommands.ExitSuccess
}
