package xev

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"
	"within.website/x/web/systemone"
)

// Pick implements the "pick" subcommand. It scores a list of options.
type Pick struct {
	promptContext string
	question      string
	model         string
	output        output
}

func (*Pick) Name() string     { return "pick" }
func (*Pick) Synopsis() string { return "Score decision options with System One." }
func (*Pick) Usage() string {
	return `pick [--context] [--question] [--model] [--all] [--json] <option> <option> [option...]:
Score decision options with System One. Pass 2 to 255 options as arguments.
By default this command prints the option with the highest probability.
`
}

func (p *Pick) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.promptContext, "context", "", "Background information for the decision.")
	f.StringVar(&p.question, "question", "", "Question to answer.")
	f.StringVar(&p.model, "model", defaultModel, "Model to use.")
	p.output.setFlags(f)
}

func (p *Pick) Execute(ctx context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	if err := validateQuestion(p.question); err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitUsageError
	}

	options := f.Args()
	if len(options) < 2 {
		fmt.Println("error: pick requires at least two options")
		return subcommands.ExitUsageError
	}

	criteria := make(map[string]any, len(options))
	for _, opt := range options {
		if _, exists := criteria[opt]; exists {
			fmt.Printf("error: duplicate option %q\n", opt)
			return subcommands.ExitUsageError
		}
		criteria[opt] = nil
	}

	req := &systemone.Request{
		Model: p.model,
		State: p.promptContext,
		Questions: map[string]systemone.Question{
			"pick": {
				Type:         systemone.Choice,
				Instructions: p.question,
				Criteria:     criteria,
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

	if err := p.output.print(resp, "pick"); err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
