package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/c-bata/go-prompt"
	"github.com/google/subcommands"
	"github.com/rodaine/table"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
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

type miListEvents struct{}

func (*miListEvents) Name() string     { return "list-events" }
func (*miListEvents) Synopsis() string { return "List events to be attended." }
func (*miListEvents) Usage() string {
	return `list-events:
List events to be attended.
`
}
func (*miListEvents) SetFlags(f *flag.FlagSet) {}

func (le *miListEvents) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	cli := mi.NewEventsProtobufClient(*miURL, http.DefaultClient)

	resp, err := cli.Get(ctx, &emptypb.Empty{})
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitFailure
	}

	tbl := table.New("Name", "Start Date", "End Date", "Location")
	for _, ev := range resp.Events {
		tbl.AddRow(ev.Name, ev.StartDate.AsTime().Format("2006-01-02"), ev.EndDate.AsTime().Format("2006-01-02"), ev.Location)
	}

	tbl.Print()

	return subcommands.ExitSuccess
}

type miAddEvent struct {
	name        string
	url         string
	startDate   string
	endDate     string
	location    string
	description string
}

func (*miAddEvent) Name() string     { return "add-event" }
func (*miAddEvent) Synopsis() string { return "Add an event to be attended." }
func (*miAddEvent) Usage() string {
	return `add-event [--name] [--url] [--start-date] [--end-date] [--location] [--description]:
Add an event to be attended.
`
}
func (ae *miAddEvent) SetFlags(f *flag.FlagSet) {
	f.StringVar(&ae.name, "name", "", "Name of the event.")
	f.StringVar(&ae.url, "url", "", "URL of the event.")
	f.StringVar(&ae.startDate, "start-date", "", "Start date of the event.")
	f.StringVar(&ae.endDate, "end-date", "", "End date of the event.")
	f.StringVar(&ae.location, "location", "", "Location of the event.")
	f.StringVar(&ae.description, "description", "", "Description of the event.")
}

func (ae *miAddEvent) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if ae.name == "" {
		ae.name = prompt.Input("Event name: ", func(d prompt.Document) []prompt.Suggest {
			return nil
		})
	}

	if ae.url == "" {
		ae.url = prompt.Input("Event URL: ", func(d prompt.Document) []prompt.Suggest {
			return nil
		})
	}

	if ae.startDate == "" {
		for {
			ae.startDate = prompt.Input("Event start date (YYYY-MM-DD): ", func(d prompt.Document) []prompt.Suggest {
				return nil
			})

			_, err := time.Parse("2006-01-02", ae.startDate)
			if err != nil {
				fmt.Printf("error parsing date: %v\n", err)
				continue
			}

			break
		}
	}

	if ae.endDate == "" {
		for {
			ae.endDate = prompt.Input("Event end date (YYYY-MM-DD): ", func(d prompt.Document) []prompt.Suggest {
				return nil
			})

			_, err := time.Parse("2006-01-02", ae.endDate)
			if err != nil {
				fmt.Printf("error parsing date: %v\n", err)
				continue
			}

			break
		}
	}

	if ae.location == "" {
		ae.location = prompt.Input("Event location: ", func(d prompt.Document) []prompt.Suggest {
			s := []prompt.Suggest{
				{Text: "Remote", Description: "Remote event"},
				{Text: "San Francisco", Description: "San Francisco, CA, USA"},
				{Text: "New York", Description: "New York, NY, USA"},
				{Text: "Ottawa", Description: "Ottawa, ON, Canada"},
				{Text: "Montreal", Description: "Montreal, QC, Canada"},
				{Text: "Toronto", Description: "Toronto, ON, Canada"},
			}
			return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
		})
	}

	if ae.description == "" {
		ae.description = prompt.Input("What will you be doing there? (first person): ", func(d prompt.Document) []prompt.Suggest {
			return nil
		})
	}

	cli := mi.NewEventsProtobufClient(*miURL, http.DefaultClient)

	startDate, err := time.Parse("2006-01-02", ae.startDate)
	if err != nil {
		fmt.Printf("error parsing start date: %v\n", err)
		return subcommands.ExitFailure
	}

	endDate, err := time.Parse("2006-01-02", ae.endDate)
	if err != nil {
		fmt.Printf("error parsing end date: %v\n", err)
		return subcommands.ExitFailure
	}

	ev := &mi.Event{
		Name:        ae.name,
		Url:         ae.url,
		StartDate:   timestamppb.New(startDate),
		EndDate:     timestamppb.New(endDate),
		Location:    ae.location,
		Description: ae.description,
	}

	slog.Info("adding event", "event", ev)

	_, err = cli.Add(ctx, ev)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
