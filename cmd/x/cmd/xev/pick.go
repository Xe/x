package xev

import (
	"context"
	"flag"
	"fmt"

	"buf.build/go/protovalidate"
	"github.com/google/subcommands"

	xevv1 "within.website/x/gen/xeiaso/net/xev/v1"
)

// Pick implements the "pick" subcommand. It scores a list of options.
type Pick struct {
	promptContext string
	question      string
	model         string
	output        output
}

func (*Pick) Name() string     { return "pick" }
func (*Pick) Synopsis() string { return "Score decision options with xev." }
func (*Pick) Usage() string {
	return `pick [--context] [--question] [--model] [--all] [--json] <option> <option> [option...]:
Score decision options with xev. Pass two to 26 options as arguments.
By default this prints the option with the highest confidence.
`
}

func (p *Pick) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.promptContext, "context", "", "Background information for the decision.")
	f.StringVar(&p.question, "question", "", "Question to answer.")
	f.StringVar(&p.model, "model", "", "Model to use. Leave empty to let xev decide.")
	p.output.setFlags(f)
}

func (p *Pick) Execute(ctx context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	model, err := parseModel(p.model)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitUsageError
	}

	req := &xevv1.PickRequest{
		Context:  p.promptContext,
		Question: p.question,
		Options:  f.Args(),
		Model:    model,
	}

	if err := protovalidate.Validate(req); err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitUsageError
	}

	cli, err := New()
	if err != nil {
		fmt.Printf("can't connect to xev: %v\n", err)
		return subcommands.ExitFailure
	}
	defer cli.Close()

	resp, err := cli.Decision.Pick(ctx, req)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitFailure
	}

	if err := p.output.print(resp); err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
