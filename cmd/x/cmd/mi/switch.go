package mi

import (
	"context"
	"flag"
	"fmt"

	"buf.build/go/protovalidate"
	"github.com/google/subcommands"
	"google.golang.org/protobuf/types/known/emptypb"

	mi "within.website/x/gen/within/website/x/mi/v1"
)

// Switch implements the "switch" subcommand which changes the front member.
type Switch struct {
	member string
}

func (*Switch) Name() string     { return "switch" }
func (*Switch) Synopsis() string { return "Switch front to a different member." }
func (*Switch) Usage() string {
	return `switch [--member]:
Switch front to a different member.
`
}

func (s *Switch) SetFlags(f *flag.FlagSet) {
	f.StringVar(&s.member, "member", "", "Member to switch to.")
}

func (s *Switch) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	// Initialise client.
	client, err := New()
	if err != nil {
		fmt.Printf("can't connect to mi %v\n", err)
		return subcommands.ExitFailure
	}
	cli := client.SwitchTracker

	// Prompt for member if not supplied.
	if s.member == "" {
		fmt.Print("Member to switch to: ")
		fmt.Scanln(&s.member)
	}

	// Validate request.
	sr := &mi.SwitchReq{MemberName: s.member}
	if err := protovalidate.Validate(sr); err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitFailure
	}

	// Retrieve current members to map old member ID to name.
	members, err := cli.Members(ctx, &emptypb.Empty{})
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitFailure
	}

	// Perform the switch.
	sw, err := cli.Switch(ctx, sr)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitFailure
	}

	var oldMember string
	for _, m := range members.Members {
		if m.Id == sw.Old.MemberId {
			oldMember = m.Name
			break
		}
	}

	fmt.Printf("switched from %s to %s\n", oldMember, s.member)
	return subcommands.ExitSuccess
}
