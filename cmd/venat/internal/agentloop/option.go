package agentloop

import "github.com/openai/openai-go/v2"

func EnableParallelToolCalling(params *openai.ChatCompletionNewParams) {
	params.ParallelToolCalls = openai.Bool(true)
}
