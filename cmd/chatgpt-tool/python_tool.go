package main

import (
	"context"

	"within.website/x/llm/codeinterpreter/python"
)

type PythonInput struct {
	Code string `json:"code" jsonschema:"The python code to execute"`
}

func Python(ctx context.Context, input PythonInput) (*python.Result, error) {
	result, err := python.Run(ctx, nil, input.Code)
	if err != nil {
		return nil, err
	}

	return result, nil
}
