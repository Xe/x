package xev

import (
	"bytes"
	"context"
	"flag"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/subcommands"
	"within.website/x/web/systemone"
)

func floatPtr(v float64) *float64 {
	return &v
}

func TestDefaultConfig(t *testing.T) {
	if *xevURL != "http://kadonomy.local:11434" {
		t.Errorf("xevURL = %q, want %q", *xevURL, "http://kadonomy.local:11434")
	}
	if defaultModel != "clef" {
		t.Errorf("defaultModel = %q, want %q", defaultModel, "clef")
	}
}

func TestOutputPrint(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		all          bool
		json         bool
		questionName string
		resp         *systemone.Response
		want         string
		wantErr      string
	}{
		{
			name:         "choice default",
			questionName: "pick",
			resp: &systemone.Response{
				Model: "clef",
				Answers: map[string]systemone.Answer{
					"pick": {
						Type:   systemone.Choice,
						Choice: "technical",
						Probabilities: map[string]float64{
							"billing":   0.1,
							"technical": 0.9,
						},
					},
				},
			},
			want: "technical\n",
		},
		{
			name:         "choice all",
			all:          true,
			questionName: "pick",
			resp: &systemone.Response{
				Model: "clef",
				Answers: map[string]systemone.Answer{
					"pick": {
						Type:   systemone.Choice,
						Choice: "technical",
						Probabilities: map[string]float64{
							"billing":   0.1,
							"technical": 0.9,
						},
						Confidence: floatPtr(0.8),
					},
				},
			},
			want: "0.9000\ttechnical\n0.1000\tbilling\nconfidence\t0.8000\n",
		},
		{
			name:         "noul default yes",
			questionName: "noul",
			resp: &systemone.Response{
				Model: "clef",
				Answers: map[string]systemone.Answer{
					"noul": {
						Type: systemone.Noul,
						Noul: floatPtr(0.9127),
					},
				},
			},
			want: "yes\n",
		},
		{
			name:         "noul default no",
			questionName: "noul",
			resp: &systemone.Response{
				Model: "clef",
				Answers: map[string]systemone.Answer{
					"noul": {
						Type: systemone.Noul,
						Noul: floatPtr(0.1),
					},
				},
			},
			want: "no\n",
		},
		{
			name:         "noul all",
			all:          true,
			questionName: "noul",
			resp: &systemone.Response{
				Model: "clef",
				Answers: map[string]systemone.Answer{
					"noul": {
						Type: systemone.Noul,
						Noul: floatPtr(0.75),
					},
				},
			},
			want: "0.7500\tyes\n0.2500\tno\n",
		},
		{
			name:         "score default with legend",
			questionName: "score",
			resp: &systemone.Response{
				Model: "clef",
				Answers: map[string]systemone.Answer{
					"score": {
						Type:  systemone.Score,
						Score: floatPtr(1.2),
						Legend: map[string]any{
							"0": "Low",
							"1": "High",
						},
						Probabilities: map[string]float64{
							"0": 0.2,
							"1": 0.8,
						},
					},
				},
			},
			want: "High\n",
		},
		{
			name:         "score all with legend",
			all:          true,
			questionName: "score",
			resp: &systemone.Response{
				Model: "clef",
				Answers: map[string]systemone.Answer{
					"score": {
						Type:  systemone.Score,
						Score: floatPtr(0.8),
						Legend: map[string]any{
							"0": "Low",
							"1": "High",
						},
						Probabilities: map[string]float64{
							"0": 0.2,
							"1": 0.8,
						},
					},
				},
			},
			want: "0.8000\tHigh\n0.2000\tLow\nscore\t0.8000\n",
		},
		{
			name:         "missing answer",
			questionName: "pick",
			resp: &systemone.Response{
				Model:   "clef",
				Answers: map[string]systemone.Answer{},
			},
			wantErr: `response does not contain answer "pick"`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			out := output{
				all:  tt.all,
				json: tt.json,
				out:  &buf,
			}

			err := out.print(tt.resp, tt.questionName)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("print() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("print() error = %v, want nil", err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("print() output = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSubcommandsExecute(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/v1/systemone", systemone.NewHandler(systemone.EvaluatorFunc(func(_ context.Context, req *systemone.Request) (*systemone.Response, error) {
		answers := make(map[string]systemone.Answer, len(req.Questions))
		for name, q := range req.Questions {
			switch q.Type {
			case systemone.Choice:
				answers[name] = systemone.Answer{
					Type:   systemone.Choice,
					Choice: "technical",
					Probabilities: map[string]float64{
						"billing":   0.2,
						"technical": 0.8,
					},
				}
			case systemone.Noul:
				answers[name] = systemone.Answer{
					Type: systemone.Noul,
					Noul: floatPtr(0.9),
				}
			case systemone.Score:
				answers[name] = systemone.Answer{
					Type:  systemone.Score,
					Score: floatPtr(1.0),
					Legend: map[string]any{
						"0": "Minor",
						"1": "Critical",
					},
					Probabilities: map[string]float64{
						"0": 0.1,
						"1": 0.9,
					},
				}
			}
		}
		return &systemone.Response{
			Model:   req.Model,
			Answers: answers,
		}, nil
	})))

	srv := httptest.NewServer(mux)
	defer srv.Close()

	oldURL := *xevURL
	*xevURL = srv.URL
	t.Cleanup(func() {
		*xevURL = oldURL
	})

	cases := []struct {
		name       string
		cmd        subcommands.Command
		args       []string
		wantStatus subcommands.ExitStatus
	}{
		{
			name:       "pick valid",
			cmd:        &Pick{},
			args:       []string{"-context", "Checkout failed", "-question", "Which team?", "billing", "technical"},
			wantStatus: subcommands.ExitSuccess,
		},
		{
			name:       "pick too few options",
			cmd:        &Pick{},
			args:       []string{"-context", "Checkout failed", "-question", "Which team?", "billing"},
			wantStatus: subcommands.ExitUsageError,
		},
		{
			name:       "pick duplicate options",
			cmd:        &Pick{},
			args:       []string{"-context", "Checkout failed", "-question", "Which team?", "billing", "billing"},
			wantStatus: subcommands.ExitUsageError,
		},
		{
			name:       "noul valid",
			cmd:        &Noul{},
			args:       []string{"-context", "Checkout failed", "-question", "Is this urgent?"},
			wantStatus: subcommands.ExitSuccess,
		},
		{
			name:       "noul missing question",
			cmd:        &Noul{},
			args:       []string{"-context", "Checkout failed"},
			wantStatus: subcommands.ExitUsageError,
		},
		{
			name:       "score valid",
			cmd:        &Score{},
			args:       []string{"-context", "Checkout failed", "-question", "How severe?", "Minor", "Critical"},
			wantStatus: subcommands.ExitSuccess,
		},
		{
			name:       "score too few levels",
			cmd:        &Score{},
			args:       []string{"-context", "Checkout failed", "-question", "How severe?", "Minor"},
			wantStatus: subcommands.ExitUsageError,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			fs := flag.NewFlagSet(tt.cmd.Name(), flag.ContinueOnError)
			tt.cmd.SetFlags(fs)
			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("Parse(%v): %v", tt.args, err)
			}

			got := tt.cmd.Execute(context.Background(), fs)
			if got != tt.wantStatus {
				t.Errorf("Execute() = %v, want %v", got, tt.wantStatus)
			}
		})
	}
}
