package agentloop

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/openai/openai-go/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ErrSentinelAbort = errors.New("agentloop: tool requested the agent loop to abort")
	ErrSentinelOkay  = errors.New("agentloop: tool requested the agent loop to stop (status okay)")

	tokensUsed = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "venat",
		Subsystem: "agentloop",
		Name:      "tokens_used",
	}, []string{"kind"})
)

type Impl struct {
	Name, ID     string
	Tools        map[string]Tool
	SystemPrompt string

	model    string
	cli      *openai.Client
	messages []openai.ChatCompletionMessageParamUnion
	lock     sync.Mutex
}

func New(name, id, systemPrompt, model string, tools []Tool, cli *openai.Client) *Impl {
	if id == "" {
		id = uuid.Must(uuid.NewV7()).String()
	}

	toolMap := map[string]Tool{}
	for _, tool := range tools {
		toolMap[tool.Name()] = tool
	}

	result := Impl{
		Name:         name,
		ID:           id,
		Tools:        toolMap,
		SystemPrompt: systemPrompt,
		model:        model,
		cli:          cli,
		messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
		},
	}

	return &result
}

func (i *Impl) Run(ctx context.Context, prompt string) error {
	i.lock.Lock()
	defer i.lock.Unlock()

	lg := slog.With("component", "agentloop", "name", i.Name, "id", i.ID, "model", i.model)

	i.messages = append(i.messages, openai.UserMessage(prompt))

	failCount := 0
	const failMax = 5

	for {
		select {
		case <-ctx.Done():
			lg.Error("context done", "err", ctx.Err())
			return ctx.Err()
		default:
		}

		params := openai.ChatCompletionNewParams{
			Messages: i.messages,
			Model:    openai.ChatModel(i.model),
		}

		for _, tool := range i.Tools {
			params.Tools = append(params.Tools, openai.ChatCompletionFunctionTool(tool.Usage()))
		}

		completion, err := i.cli.Chat.Completions.New(ctx, params)
		if err != nil {
			failCount++

			if failCount == failMax {
				return fmt.Errorf("can't reach remote API: %w", err)
			}

			lg.Error("can't get completion, sleeping and retrying", "err", err, "failCount", failCount, "failMax", failMax)
			time.Sleep(time.Duration(failCount) * time.Second)
			continue
		}

		tokensUsed.WithLabelValues("input").Add(float64(completion.Usage.PromptTokens))
		tokensUsed.WithLabelValues("output").Add(float64(completion.Usage.CompletionTokens))

		i.messages = append(i.messages, completion.Choices[0].Message.ToParam())

		toolCalls := completion.Choices[0].Message.ToolCalls

		for _, tc := range toolCalls {
			lg := lg.With("tool", tc.Function.Name, "toolcall_id", tc.ID)
			tool, ok := i.Tools[tc.Function.Name]
			if !ok {
				lg.Error("AI model chose tool that did not exist, asking it to try again")
				i.messages = append(i.messages, openai.UserMessage(fmt.Sprintf("Tool %q does not exist, please try again.", tc.Function.Name)))
				continue
			}

			args := []byte(tc.Function.Arguments)
			if err := tool.Valid(args); err != nil {
				lg.Error("AI model produced invalid arguments", "err", err)
				i.messages = append(i.messages, openai.UserMessage(fmt.Sprintf("When calling tool %q, you got an argument validation error: %v", tool.Name(), err)))
				continue
			}

			result, err := tool.Run(ctx, args)
			if err != nil {
				switch {
				case errors.Is(err, ErrSentinelOkay):
					lg.Info("tool requested happy exit", "err", err)
					return err
				case errors.Is(err, ErrSentinelAbort):
					lg.Info("tool requested unhappy abort", "err", err)
					return err
				default:
					lg.Error("failed to run tool", "err", err)
					i.messages = append(i.messages, openai.ToolMessage(fmt.Sprintf("internal error when running tool %q: %v", tool.Name(), err), tc.ID))
					continue
				}
			}

			i.messages = append(i.messages, openai.ToolMessage(string(result), tc.ID))
		}
	}
}
