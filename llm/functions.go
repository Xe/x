package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

type FunctionMessage struct {
	Role         string     `json:"role"`
	SystemPrompt string     `json:"content"`
	UserQuestion string     `json:"user_question"`
	Functions    []Function `json:"functions"`
}

type Function struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Arguments   []Argument `json:"arguments"`
}

type FunctionResponse struct {
	Function  string            `json:"function"`
	Arguments map[string]string `json:"arguments"`
}

type Argument struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

func (m FunctionMessage) ChatML() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "<s>[INST] <<SYS>>\n%s The following functions are available for you to fetch further data to answer user questions, if relevant:\n\n", m.SystemPrompt)
	enc := json.NewEncoder(&sb)

	for _, function := range m.Functions {
		enc.Encode(function)
	}

	fmt.Fprintf(&sb, `
	To call a function, respond - immediately and only - with a JSON object of the following format:
	{
		"function": "function_name",
		"arguments": {
			"argument1": "argument_value",
			"argument2": "argument_value"
		}
	}
	<</SYS>>
	
	`)
	fmt.Fprintf(&sb, "%s [/INST]", m.UserQuestion)

	return sb.String()
}
