package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	"github.com/google/subcommands"
	"github.com/rodaine/table"
	"google.golang.org/protobuf/types/known/emptypb"
	"within.website/x/proto/mi"
)

var (
	miURL = flag.String("mi-url", "http://mi.mi.svc.alrest.xeserv.us", "Base mi URL")
)

type miWhoIsFront struct {
	simple bool
}

func (*miWhoIsFront) Name() string     { return "who-is-front" }
func (*miWhoIsFront) Synopsis() string { return "Print who is front of the system." }
func (*miWhoIsFront) Usage() string {
	return `who-is-front [--mi-url]:
Print who is front of the system.
`
}
func (wif *miWhoIsFront) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&wif.simple, "simple", false, "Print only the name of the front member.")
}
func (wif *miWhoIsFront) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	cli := mi.NewSwitchTrackerProtobufClient(*miURL, http.DefaultClient)

	front, err := cli.WhoIsFront(ctx, &emptypb.Empty{})
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitFailure
	}

	if wif.simple {
		fmt.Printf("%s\n", front.Member.Name)
		return subcommands.ExitSuccess
	}

	fmt.Printf("current front: %s\n", front.Member.Name)

	return subcommands.ExitSuccess
}

type miSwitch struct {
	member string
}

func (*miSwitch) Name() string     { return "switch" }
func (*miSwitch) Synopsis() string { return "Switch front to a different member." }
func (*miSwitch) Usage() string {
	return `switch [--member]:
Switch front to a different member.
`
}
func (s *miSwitch) SetFlags(f *flag.FlagSet) {
	f.StringVar(&s.member, "member", "", "Member to switch to.")
}
func (s *miSwitch) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	cli := mi.NewSwitchTrackerProtobufClient(*miURL, http.DefaultClient)

	_, err := cli.Switch(ctx, &mi.SwitchReq{MemberName: s.member})
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

type miListSwitches struct {
	count int
	page  int
}

func (*miListSwitches) Name() string     { return "list-switches" }
func (*miListSwitches) Synopsis() string { return "List switches." }
func (*miListSwitches) Usage() string {
	return `list-switches [--count] [--page]:
List a number of past switches.
`
}
func (ls *miListSwitches) SetFlags(f *flag.FlagSet) {
	f.IntVar(&ls.count, "count", 10, "Number of switches to list.")
	f.IntVar(&ls.page, "page", 0, "Page of switches to list.")
}
func (ls *miListSwitches) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	cli := mi.NewSwitchTrackerProtobufClient(*miURL, http.DefaultClient)

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
