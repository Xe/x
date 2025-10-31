package main

import (
	"context"
	"os"

	"within.website/x/llm/codeinterpreter/python"
)

type PythonInput struct {
	Code string `json:"code" jsonschema:"The python code to execute"`
}

func Python(ctx context.Context, input PythonInput) (*python.Result, error) {
	dir, err := os.MkdirTemp("", "python-wasm-mcp-*")
	if err != nil {
		return nil, err
	}

	defer os.RemoveAll(dir)

	result, err := python.Run(ctx, dir, input.Code)
	if err != nil {
		return nil, err
	}

	return result, nil
}
