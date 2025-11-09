package tools

import (
	"context"

	"github.com/openai/openai-go/v2"
)

type Implementation interface {
	Name() string
	Usage() openai.FunctionDefinitionParam
	Valid(data []byte) (hide bool, err error)
	Run(ctx context.Context, data []byte) (string, error)
}
