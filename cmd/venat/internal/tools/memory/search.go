package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/philippgille/chromem-go"
)

var (
	ErrNoQuery = errors.New("memory: no search query for memory")
)

type MemorySearchInput struct {
	Query string `json:"query" jsonschema:"The search query for internal memory"`
}

func (msi MemorySearchInput) Valid() error {
	if msi.Query == "" {
		return ErrNoQuery
	}

	return nil
}

type MemorySearch struct {
	Coll *chromem.Collection
}

func (*MemorySearch) Name() string {
	return "memory_search"
}

func (*MemorySearch) Usage() openai.FunctionDefinitionParam {
	return openai.FunctionDefinitionParam{
		Name:        "memory_search",
		Description: openai.String("Use to search your memory for relevant information. Use this tool before generating a response to the user."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]string{
					"type": "string",
				},
			},
			"required": []string{"query"},
		},
	}
}

func (*MemorySearch) Valid(data []byte) error {
	var msi MemorySearchInput
	if err := json.Unmarshal(data, &msi); err != nil {
		return fmt.Errorf("can't parse json: %w", err)
	}

	return msi.Valid()
}

func (ma *MemorySearch) Run(ctx context.Context, data []byte) ([]byte, error) {
	var msi MemorySearchInput
	if err := json.Unmarshal(data, &msi); err != nil {
		return nil, fmt.Errorf("can't parse json: %w", err)
	}

	results, err := ma.Coll.Query(ctx, msi.Query, 10, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("can't search memory: %w", err)
	}

	buf := bytes.NewBuffer(nil)

	if len(results) == 0 {
		return []byte("No matches found."), nil
	}

	fmt.Fprintln(buf, "Found the following results")

	for _, result := range results {
		fmt.Fprintf(buf, "---memory ID %s---\n%s\n", result.ID, result.Content)
	}

	return buf.Bytes(), nil
}
