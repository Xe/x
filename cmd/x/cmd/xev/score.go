package xev

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"
	"within.website/x/web/systemone"
)

// Score implements the "score" subcommand. It evaluates an ordered list of levels.
type Score struct {
	promptContext string
	question      string
	model         string
	output        output
}

func (*Score) Name() string     { return "score" }
func (*Score) Synopsis() string { return "Score ordered levels with System One." }
func (*Score) Usage() string {
	return `score [--context] [--question] [--model] [--all] [--json] <level> <level> [level...]:
Score ordered levels with System One. Pass 2 to 10 levels as arguments.
By default this command prints the level with the highest probability.
`
}

func (s *Score) SetFlags(f *flag.FlagSet) {
	f.StringVar(&s.promptContext, "context", "", "Background information for the decision.")
	f.StringVar(&s.question, "question", "", "Question to answer.")
	f.StringVar(&s.model, "model", defaultModel, "Model to use.")
	s.output.setFlags(f)
}

func (s *Score) Execute(ctx context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	if err := validateQuestion(s.question); err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitUsageError
	}

	req := &systemone.Request{
		Model: s.model,
		State: s.promptContext,
		Questions: map[string]systemone.Question{
			"score": {
				Type:         systemone.Score,
				Instructions: s.question,
				Criteria:     f.Args(),
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

	if err := s.output.print(resp, "score"); err != nil {
		fmt.Printf("error: %v\n", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
