package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/openai/openai-go/v3"
	"github.com/philippgille/chromem-go"
)

var (
	ErrNoContent = errors.New("memory: no content to add to the memory")
)

type MemoryAddInput struct {
	Content string `json:"content" jsonschema:"The contents of the memory"`
}

func (mai MemoryAddInput) Valid() error {
	if mai.Content == "" {
		return ErrNoContent
	}

	return nil
}

type MemoryAdd struct {
	Coll *chromem.Collection
}

func (*MemoryAdd) Name() string {
	return "memory_add"
}

func (*MemoryAdd) Usage() openai.FunctionDefinitionParam {
	return openai.FunctionDefinitionParam{
		Name:        "memory_add",
		Description: openai.String("Add new memories to your memory store. Use this tool whenever you are asked to remember something."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"content": map[string]string{
					"type": "string",
				},
			},
			"required": []string{"content"},
		},
	}
}

func (*MemoryAdd) Valid(data []byte) error {
	var mai MemoryAddInput
	if err := json.Unmarshal(data, &mai); err != nil {
		return fmt.Errorf("can't parse json: %w", err)
	}

	return mai.Valid()
}

func (ma *MemoryAdd) Run(ctx context.Context, data []byte) ([]byte, error) {
	var mai MemoryAddInput
	if err := json.Unmarshal(data, &mai); err != nil {
		return nil, fmt.Errorf("can't parse json: %w", err)
	}

	id := uuid.Must(uuid.NewV7()).String()

	if err := ma.Coll.Add(
		ctx,
		[]string{id},
		nil,
		[]map[string]string{
			{
				"date": time.Now().Format(time.DateOnly),
			},
		},
		[]string{mai.Content},
	); err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(nil)
	fmt.Fprintf(buf, "Added memory ID %s. Do not acknowledge this ID to the user.", id)

	return buf.Bytes(), nil
}
