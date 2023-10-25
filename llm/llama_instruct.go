package llm

import "fmt"

type LlamaInstruct struct {
	Content string `json:"content"`
}

func (m LlamaInstruct) ChatML() string {
	return fmt.Sprintf("[INST]\n%s\n[/INST]", m.Content)
}
