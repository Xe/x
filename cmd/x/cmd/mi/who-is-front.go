package mi

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"
	"google.golang.org/protobuf/types/known/emptypb"
)

type WhoIsFront struct {
	simple bool
}

func (*WhoIsFront) Name() string     { return "who-is-front" }
func (*WhoIsFront) Synopsis() string { return "Print who is front of the system." }
func (*WhoIsFront) Usage() string {
	return `who-is-front [--mi-url]:
Print who is front of the system.
`
}
func (wif *WhoIsFront) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&wif.simple, "simple", false, "Print only the name of the front member.")
}
func (wif *WhoIsFront) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	mi, err := New()
	if err != nil {
		fmt.Printf("can't connect to mi %v\n", err)
		return subcommands.ExitFailure
	}
	cli := mi.SwitchTracker

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
