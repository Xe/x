package xev

import (
	"context"
	"flag"
	"fmt"

	"buf.build/go/protovalidate"
	"github.com/google/subcommands"

	xevv1 "within.website/x/gen/xeiaso/net/xev/v1"
)

// Noul implements the "noul" subcommand. It scores a yes or no question.
type Noul struct {
	promptContext string
	question      string
	model         string
	output        output
}

func (*Noul) Name() string     { return "noul" }
func (*Noul) Synopsis() string { return "Score a yes or no question with xev." }
func (*Noul) Usage() string {
	return `noul [--context] [--question] [--model] [--all] [--json]:
Score a yes or no question with xev.
By default this prints the answer with the highest confidence.
`
}

func (n *Noul) SetFlags(f *flag.FlagSet) {
	f.StringVar(&n.promptContext, "context", "", "Background information for the decision.")
	f.StringVar(&n.question, "question", "", "Question to answer.")
	f.StringVar(&n.model, "model", "", "Model to use. Leave empty to let xev decide.")
	n.output.setFlags(f)
}

func (n *Noul) Execute(ctx context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	model, err := parseModel(n.model)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitUsageError
	}

	req := &xevv1.NoulRequest{
		Context:  n.promptContext,
		Question: n.question,
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

	resp, err := cli.Decision.Noul(ctx, req)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitFailure
	}

	if err := n.output.print(resp); err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
