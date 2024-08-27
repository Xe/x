package jufra

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"

	"within.website/x/llm/codeinterpreter/python"
	"within.website/x/web/ollama"
)

var normalTools = []ollama.Function{
	{
		Name:        "code_interpreter",
		Description: "Run the given Python code.",
		Parameters: ollama.Param{
			Type: "object",
			Properties: ollama.Properties{
				"code": {
					Type:        "string",
					Description: "The Python code to run",
				},
			},
			Required: []string{"code"},
		},
	},
	{
		Name:        "none",
		Description: "No tools are relevant for this message",
	},
	// {
	// 	Name:        "reply",
	// 	Description: "Reply to the message",
	// 	Parameters: ollama.Param{
	// 		Type: "object",
	// 		Properties: ollama.Properties{
	// 			"message": {
	// 				Type:        "string",
	// 				Description: "The message to send",
	// 			},
	// 		},
	// 		Required: []string{"message"},
	// 	},
	// },
}

type pythonCodeArgs struct {
	Code string `json:"code"`
}

func (pca *pythonCodeArgs) Valid() error {
	if pca.Code == "" {
		return errors.New("missing code parameter")
	}

	return nil
}

func (m *Module) runPythonCode(ctx context.Context, tc ollama.ToolCall) (*ollama.Message, error) {
	var args pythonCodeArgs
	if err := json.Unmarshal(tc.Arguments, &args); err != nil {
		return nil, err
	}

	tmpdir, err := os.MkdirTemp("", "mimi-python-*")
	if err != nil {
		return nil, err
	}

	defer os.RemoveAll(tmpdir)

	res, err := python.Run(ctx, tmpdir, args.Code)
	if err != nil {
		return &ollama.Message{
			Role:    "tool",
			Content: jsonString(map[string]string{"error": err.Error(), "stdout": res.Stdout, "stderr": res.Stderr}),
		}, nil
	}

	slog.Info("python code ran", "res", res, "args", args)

	return &ollama.Message{
		Role:    "tool",
		Content: jsonString(res),
	}, nil
}

func (m *Module) getTools() []ollama.Tool {
	var result []ollama.Tool

	for _, tool := range normalTools {
		result = append(result, ollama.Tool{
			Type:     "function",
			Function: tool,
		})
	}

	return result
}
