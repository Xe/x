package agent

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"within.website/x/cmd/markdownlang/internal/parser"
)

func TestNewProgram(t *testing.T) {
	source := `---
name: test-program
description: A test program
input:
  type: object
  properties:
    message: {type: string}
  required: [message]
output:
  type: object
  properties:
    result: {type: string}
  required: [result]
---

This is a test program.
`

	prog, err := parser.ParseProgram(source)
	if err != nil {
		t.Fatalf("failed to parse program: %v", err)
	}

	program, err := NewProgram(prog)
	if err != nil {
		t.Fatalf("failed to create program: %v", err)
	}

	if program.Name != "test-program" {
		t.Errorf("expected name %q, got %q", "test-program", program.Name)
	}

	if program.model != DefaultModel {
		t.Errorf("expected model %q, got %q", DefaultModel, program.model)
	}

	if program.temperature != DefaultTemperature {
		t.Errorf("expected temperature %f, got %f", DefaultTemperature, program.temperature)
	}
}

func TestNewProgramWithOptions(t *testing.T) {
	source := `---
name: test-program
description: A test program
input:
  type: object
  properties:
    message: {type: string}
  required: [message]
output:
  type: object
  properties:
    result: {type: string}
  required: [result]
---

This is a test program.
`

	prog, err := parser.ParseProgram(source)
	if err != nil {
		t.Fatalf("failed to parse program: %v", err)
	}

	customModel := "o1-pro"
	customTemp := 0.5

	program, err := NewProgram(prog,
		WithModel(customModel),
		WithTemperature(customTemp),
	)
	if err != nil {
		t.Fatalf("failed to create program: %v", err)
	}

	if program.model != customModel {
		t.Errorf("expected model %q, got %q", customModel, program.model)
	}

	if program.temperature != customTemp {
		t.Errorf("expected temperature %f, got %f", customTemp, program.temperature)
	}
}

func TestNewProgramNil(t *testing.T) {
	_, err := NewProgram(nil)
	if err == nil {
		t.Fatal("expected error for nil program, got nil")
	}
}

func TestProgramValidateInput(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		input   string
		wantErr bool
	}{
		{
			name: "valid input",
			source: `---
name: test
description: test
input:
  type: object
  properties:
    message: {type: string}
  required: [message]
output:
  type: object
  properties:
    result: {type: string}
  required: [result]
---
test`,
			input:   `{"message": "hello"}`,
			wantErr: false,
		},
		{
			name: "empty input",
			source: `---
name: test
description: test
input:
  type: object
  properties:
    message: {type: string}
  required: [message]
output:
  type: object
  properties:
    result: {type: string}
  required: [result]
---
test`,
			input:   "",
			wantErr: true,
		},
		{
			name: "invalid json",
			source: `---
name: test
description: test
input:
  type: object
  properties:
    message: {type: string}
  required: [message]
output:
  type: object
  properties:
    result: {type: string}
  required: [result]
---
test`,
			input:   `{not json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := parser.ParseProgram(tt.source)
			if err != nil {
				t.Fatalf("failed to parse program: %v", err)
			}

			program, err := NewProgram(prog)
			if err != nil {
				t.Fatalf("failed to create program: %v", err)
			}

			err = program.validateInput(json.RawMessage(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("validateInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProgramValidateOutput(t *testing.T) {
	source := `---
name: test
description: test
input:
  type: object
  properties:
    message: {type: string}
  required: [message]
output:
  type: object
  properties:
    result: {type: string}
  required: [result]
---
test`

	prog, err := parser.ParseProgram(source)
	if err != nil {
		t.Fatalf("failed to parse program: %v", err)
	}

	program, err := NewProgram(prog)
	if err != nil {
		t.Fatalf("failed to create program: %v", err)
	}

	tests := []struct {
		name    string
		output  string
		wantErr bool
	}{
		{
			name:    "valid output",
			output:  `{"result": "success"}`,
			wantErr: false,
		},
		{
			name:    "empty output",
			output:  "",
			wantErr: true,
		},
		{
			name:    "invalid json",
			output:  `{not json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := program.validateOutput(json.RawMessage(tt.output))
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOutput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProgramBuildSystemMessage(t *testing.T) {
	source := `---
name: test-program
description: A test program for testing
input:
  type: object
  properties:
    message: {type: string}
  required: [message]
output:
  type: object
  properties:
    result: {type: string}
  required: [result]
---

This is a test program.
`

	prog, err := parser.ParseProgram(source)
	if err != nil {
		t.Fatalf("failed to parse program: %v", err)
	}

	program, err := NewProgram(prog)
	if err != nil {
		t.Fatalf("failed to create program: %v", err)
	}

	input := json.RawMessage(`{"message": "hello world"}`)

	// First iteration (no error)
	msg := program.buildSystemMessage(input, 0, nil)
	if msg == "" {
		t.Error("expected non-empty system message")
	}

	// Should contain the program name
	if !contains(msg, "test-program") {
		t.Error("system message should contain program name")
	}

	// Should contain the input
	if !contains(msg, "hello world") {
		t.Error("system message should contain input data")
	}

	// Second iteration with error
	msgWithError := program.buildSystemMessage(input, 1, &ValidationError{})
	if !contains(msgWithError, "Previous Error") {
		t.Error("system message with error should contain error section")
	}
}

func TestProgramExecute(t *testing.T) {
	if os.Getenv("AGENT_REALWORLD") == "" {
		t.Skip("AGENT_REALWORLD not set, skipping real execution test")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping real execution test")
	}

	source := `---
name: echo
description: Echo the input message
input:
  type: object
  properties:
    message: {type: string}
  required: [message]
output:
  type: object
  properties:
    echoed: {type: string}
  required: [echoed]
---

Echo back the message you receive.
`

	prog, err := parser.ParseProgram(source)
	if err != nil {
		t.Fatalf("failed to parse program: %v", err)
	}

	program, err := NewProgram(prog, WithAPIKey(apiKey))
	if err != nil {
		t.Fatalf("failed to create program: %v", err)
	}

	input := json.RawMessage(`{"message": "hello world"}`)
	ctx := context.Background()

	output, err := program.Execute(ctx, input)
	if err != nil {
		t.Fatalf("failed to execute program: %v", err)
	}

	// Validate output is JSON
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Check we got the required field
	if _, ok := result["echoed"]; !ok {
		t.Error("output missing required field 'echoed'")
	}

	// Check metrics
	metrics := program.Metrics()
	if metrics.Iterations == 0 {
		t.Error("expected at least one iteration")
	}

	t.Logf("Metrics: %s", metrics.String())
}

// Helper types and functions

type ValidationError struct {
	msg string
}

func (e *ValidationError) Error() string {
	return e.msg
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr))))
}
