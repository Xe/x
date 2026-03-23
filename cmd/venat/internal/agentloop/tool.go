package agentloop

import (
	"context"

	"github.com/openai/openai-go/v2"
)

type Tool interface {
	Name() string
	Usage() openai.FunctionDefinitionParam
	Valid(data []byte) (err error)
	Run(ctx context.Context, data []byte) ([]byte, error)
}
