package python

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openai/openai-go/v2"
	"within.website/x/llm/codeinterpreter/python"
)

var (
	ErrNoCode = errors.New("python: no code provided")
)

type Input struct {
	Code string `json:"code" jsonschema:"The python code to execute"`
}

func (i Input) Valid() error {
	if i.Code == "" {
		return ErrNoCode
	}

	return nil
}

type Impl struct{}

func (Impl) Name() string {
	return "python"
}

func (Impl) Usage() openai.FunctionDefinitionParam {
	return openai.FunctionDefinitionParam{
		Name:        "python",
		Description: openai.String("Execute python code in a restrictive sandbox. Use this tool whenever doing calculations."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"code": map[string]string{
					"type": "string",
				},
			},
			"required": []string{"code"},
		},
	}
}

func (Impl) Valid(data []byte) error {
	var i Input
	if err := json.Unmarshal(data, &i); err != nil {
		return fmt.Errorf("can't parse json: %w", err)
	}

	return i.Valid()
}

func (Impl) Run(ctx context.Context, data []byte) ([]byte, error) {
	var i Input
	if err := json.Unmarshal(data, &i); err != nil {
		return nil, fmt.Errorf("can't parse json: %w", err)
	}

	result, err := python.Run(ctx, nil, i.Code)
	if err != nil {
		return nil, fmt.Errorf("can't execute python code: %w", err)
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("can't marshal result bytes: %w", err)
	}

	return resultBytes, nil
}
