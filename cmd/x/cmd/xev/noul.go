package xev

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"
	"within.website/x/web/systemone"
)

// Noul implements the "noul" subcommand. It scores a yes or no question.
type Noul struct {
	promptContext string
	question      string
	model         string
	output        output
}

func (*Noul) Name() string     { return "noul" }
func (*Noul) Synopsis() string { return "Score a yes or no question with System One." }
func (*Noul) Usage() string {
	return `noul [--context] [--question] [--model] [--all] [--json]:
Score a yes or no question with System One.
By default this command prints the answer with the highest probability.
`
}

func (n *Noul) SetFlags(f *flag.FlagSet) {
	f.StringVar(&n.promptContext, "context", "", "Background information for the decision.")
	f.StringVar(&n.question, "question", "", "Question to answer.")
	f.StringVar(&n.model, "model", defaultModel, "Model to use.")
	n.output.setFlags(f)
}

func (n *Noul) Execute(ctx context.Context, _ *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	if err := validateQuestion(n.question); err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitUsageError
	}

	req := &systemone.Request{
		Model: n.model,
		State: n.promptContext,
		Questions: map[string]systemone.Question{
			"noul": {
				Type:         systemone.Noul,
				Instructions: n.question,
			},
		},
	}

	if err := req.Valid(); err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitUsageError
	}

	resp, err := New().Evaluate(ctx, req)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitFailure
	}

	if err := n.output.print(resp, "noul"); err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
