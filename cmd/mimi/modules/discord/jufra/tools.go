package jufra

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"within.website/x/llm/codeinterpreter/python"
	"within.website/x/web/ollama"
)

var normalTools = []ollama.Function{
	{
		Name:        "run_python_code",
		Description: "Run the given Python code in a sandboxed environment",
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
		return nil, nil
	}

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
