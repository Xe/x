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
	}, []string{"model", "kind"})
)

type Impl struct {
	Name, ID     string
	Tools        map[string]Tool
	SystemPrompt string

	model string
	cli   *openai.Client
	lg    *slog.Logger

	messages []openai.ChatCompletionMessageParamUnion
	lock     sync.Mutex
}

func New(name, id, systemPrompt, model string, tools []Tool, cli *openai.Client, lg *slog.Logger) *Impl {
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

type Result struct {
	Messages []openai.ChatCompletionMessageParamUnion
	Response string

	PromptTokens              int64
	PromptCachedTokens        int64
	CompletionTokens          int64
	CompletionReasoningTokens int64
}

func (i *Impl) Run(ctx context.Context, prompt string) (*Result, error) {
	i.lock.Lock()
	defer i.lock.Unlock()

	lg := i.lg.With("component", "agentloop", "name", i.Name, "id", i.ID, "model", i.model)

	i.messages = append(i.messages, openai.UserMessage(prompt))

	failCount := 0
	const failMax = 5

	result := Result{}

	for {
		select {
		case <-ctx.Done():
			lg.Error("context done", "err", ctx.Err())
			return &result, ctx.Err()
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
				return &result, fmt.Errorf("can't reach remote API: %w", err)
			}

			lg.Error("can't get completion, sleeping and retrying", "err", err, "failCount", failCount, "failMax", failMax)
			time.Sleep(time.Duration(failCount) * time.Second)
			continue
		}

		tokensUsed.WithLabelValues(i.model, "input").Add(float64(completion.Usage.PromptTokens))
		tokensUsed.WithLabelValues(i.model, "output").Add(float64(completion.Usage.CompletionTokens))
		tokensUsed.WithLabelValues(i.model, "cached").Add(float64(completion.Usage.PromptTokensDetails.CachedTokens))
		tokensUsed.WithLabelValues(i.model, "reasoning").Add(float64(completion.Usage.CompletionTokensDetails.ReasoningTokens))

		result.PromptTokens += completion.Usage.PromptTokens
		result.PromptCachedTokens += completion.Usage.PromptTokensDetails.CachedTokens
		result.CompletionTokens += completion.Usage.CompletionTokens
		result.CompletionReasoningTokens += completion.Usage.CompletionTokensDetails.ReasoningTokens

		resp := completion.Choices[0].Message

		i.messages = append(i.messages, resp.ToParam())
		result.Messages = i.messages

		if resp.Content != "" {
			result.Response = resp.Content
			return &result, nil
		}

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

			toolResult, err := tool.Run(ctx, args)
			if err != nil {
				switch {
				case errors.Is(err, ErrSentinelOkay):
					lg.Info("tool requested happy exit", "err", err)
					return &result, err
				case errors.Is(err, ErrSentinelAbort):
					lg.Info("tool requested unhappy abort", "err", err)
					return &result, err
				default:
					lg.Error("failed to run tool", "err", err)
					i.messages = append(i.messages, openai.ToolMessage(fmt.Sprintf("internal error when running tool %q: %v", tool.Name(), err), tc.ID))
					continue
				}
			}

			i.messages = append(i.messages, openai.ToolMessage(string(toolResult), tc.ID))
		}
	}
}
