// Package xev contains subcommands that call the System One decision API.
package xev

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"within.website/x/web/systemone"
)

const defaultModel = "clef"

var (
	xevAPIKey = flag.String("xev-api-key", "", "API key for the System One server.")
	xevURL    = flag.String("xev-url", "http://kadonomy.local:11434", "Base System One URL.")
)

// New returns a System One client configured from the command flags.
func New() *systemone.Client {
	cli := systemone.NewClient(*xevURL)
	cli.APIKey = *xevAPIKey
	return cli
}

func validateQuestion(question string) error {
	if strings.TrimSpace(question) == "" {
		return errors.New("question is required")
	}
	return nil
}

// output holds the flags that control how a decision is printed.
type output struct {
	all  bool
	json bool
	out  io.Writer
}

// setFlags registers the output flags on f.
func (o *output) setFlags(f *flag.FlagSet) {
	f.BoolVar(&o.all, "all", false, "Print every option with its probability score.")
	f.BoolVar(&o.json, "json", false, "Print the raw response as JSON.")
}

func (o *output) writer() io.Writer {
	if o.out != nil {
		return o.out
	}
	return os.Stdout
}

// optionScore pairs an option name with its probability score.
type optionScore struct {
	name  string
	value float64
}

// print writes resp to stdout in the format that the output flags select.
func (o *output) print(resp *systemone.Response, questionName string) error {
	w := o.writer()

	if o.json {
		buf, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return fmt.Errorf("cannot encode response as JSON: %w", err)
		}
		fmt.Fprintln(w, string(buf))
		return nil
	}

	if resp == nil {
		return errors.New("server returned an empty response")
	}

	ans, ok := resp.Answers[questionName]
	if !ok {
		return fmt.Errorf("response does not contain answer %q", questionName)
	}

	probs := answerProbabilities(ans)
	ranked := rank(probs)
	if len(ranked) == 0 {
		return errors.New("server returned no probability scores")
	}

	if !o.all {
		if ans.Type == systemone.Choice && ans.Choice != "" {
			fmt.Fprintln(w, ans.Choice)
			return nil
		}
		fmt.Fprintln(w, ranked[0].name)
		return nil
	}

	for _, sc := range ranked {
		fmt.Fprintf(w, "%.4f\t%s\n", sc.value, sc.name)
	}
	if ans.Score != nil {
		fmt.Fprintf(w, "score\t%.4f\n", *ans.Score)
	}
	if ans.Confidence != nil {
		fmt.Fprintf(w, "confidence\t%.4f\n", *ans.Confidence)
	}

	return nil
}

func answerProbabilities(ans systemone.Answer) map[string]float64 {
	if len(ans.Probabilities) > 0 {
		if len(ans.Legend) == 0 {
			return ans.Probabilities
		}
		mapped := make(map[string]float64, len(ans.Probabilities))
		for k, v := range ans.Probabilities {
			if label, ok := ans.Legend[k].(string); ok && label != "" {
				mapped[label] = v
			} else {
				mapped[k] = v
			}
		}
		return mapped
	}

	switch ans.Type {
	case systemone.Noul:
		if ans.Noul != nil {
			p := *ans.Noul
			return map[string]float64{
				"yes": p,
				"no":  1 - p,
			}
		}
	case systemone.Choice:
		if ans.Choice != "" {
			return map[string]float64{ans.Choice: 1}
		}
	}

	return nil
}

// rank sorts probability scores from highest to lowest. Equal scores sort by
// name so that repeated calls print the same order.
func rank(probabilities map[string]float64) []optionScore {
	scores := make([]optionScore, 0, len(probabilities))
	for name, value := range probabilities {
		scores = append(scores, optionScore{name: name, value: value})
	}

	sort.Slice(scores, func(i, j int) bool {
		if scores[i].value != scores[j].value {
			return scores[i].value > scores[j].value
		}
		return scores[i].name < scores[j].name
	})

	return scores
}
