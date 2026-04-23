package agentloop

import "github.com/openai/openai-go/v3"

func EnableParallelToolCalling(params *openai.ChatCompletionNewParams) {
	params.ParallelToolCalls = openai.Bool(true)
}
