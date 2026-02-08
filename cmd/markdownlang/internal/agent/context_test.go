package agent

import (
	"context"
	"encoding/json"
	"testing"

	"within.website/x/cmd/markdownlang/internal/parser"
)

func TestNewExecutionContext(t *testing.T) {
	config := &ExecutionContextConfig{
		DefaultModel:       "gpt-4o",
		DefaultTemperature: 0.5,
		APIKey:             "test-key",
		BaseURL:            "https://api.openai.com/v1",
	}

	ctx := NewExecutionContext(config)

	if ctx == nil {
		t.Fatal("expected non-nil ExecutionContext")
	}

	if ctx.config != config {
		t.Error("expected config to be set")
	}

	if ctx.programs == nil {
		t.Error("expected programs map to be initialized")
	}

	if ctx.imported == nil {
		t.Error("expected imported map to be initialized")
	}

	if ctx.tools == nil {
		t.Error("expected tools map to be initialized")
	}
}

func TestNewExecutionContextNilConfig(t *testing.T) {
	ctx := NewExecutionContext(nil)

	if ctx == nil {
		t.Fatal("expected non-nil ExecutionContext")
	}

	if ctx.config == nil {
		t.Error("expected config to be initialized")
	}
}

func TestExecutionContextLoadProgram(t *testing.T) {
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

	ctx := NewExecutionContext(nil)

	loop, err := ctx.LoadProgram(source)
	if err != nil {
		t.Fatalf("failed to load program: %v", err)
	}

	if loop == nil {
		t.Fatal("expected non-nil AgentLoop")
	}

	// Check that program was cached
	prog, ok := ctx.GetProgram("test-program")
	if !ok {
		t.Fatal("expected program to be cached")
	}

	if prog.Name != "test-program" {
		t.Errorf("expected program name %q, got %q", "test-program", prog.Name)
	}
}

func TestExecutionContextGetProgram(t *testing.T) {
	ctx := NewExecutionContext(nil)

	// Get non-existent program
	_, ok := ctx.GetProgram("nonexistent")
	if ok {
		t.Error("expected ok to be false for non-existent program")
	}

	// Load a program
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

	_, err := ctx.LoadProgram(source)
	if err != nil {
		t.Fatalf("failed to load program: %v", err)
	}

	// Get existing program
	prog, ok := ctx.GetProgram("test-program")
	if !ok {
		t.Fatal("expected ok to be true for existing program")
	}

	if prog.Name != "test-program" {
		t.Errorf("expected program name %q, got %q", "test-program", prog.Name)
	}
}

func TestExecutionContextImport(t *testing.T) {
	ctx := NewExecutionContext(nil)

	// Create a mock agent loop
	mockLoop := &mockAgentLoop{}

	// Register import
	ctx.RegisterImport("./test-import", mockLoop)

	// Get import
	loop, ok := ctx.GetImport("./test-import")
	if !ok {
		t.Fatal("expected ok to be true for existing import")
	}

	if loop != mockLoop {
		t.Error("expected to get the same loop that was registered")
	}

	// Get non-existent import
	_, ok = ctx.GetImport("./nonexistent")
	if ok {
		t.Error("expected ok to be false for non-existent import")
	}
}

func TestExecutionContextTools(t *testing.T) {
	ctx := NewExecutionContext(nil)

	// Create a mock tool handler
	mockTool := &mockToolHandler{
		schema: json.RawMessage(`{"type": "object"}`),
	}

	// Register tool
	ctx.RegisterTool("test-tool", mockTool)

	// Get tool
	tool, ok := ctx.GetTool("test-tool")
	if !ok {
		t.Fatal("expected ok to be true for existing tool")
	}

	if tool != mockTool {
		t.Error("expected to get the same tool that was registered")
	}

	// Get non-existent tool
	_, ok = ctx.GetTool("nonexistent")
	if ok {
		t.Error("expected ok to be false for non-existent tool")
	}

	// List tools
	tools := ctx.ListTools()
	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}

	if tools[0] != "test-tool" {
		t.Errorf("expected tool name %q, got %q", "test-tool", tools[0])
	}
}

func TestToolHandlerFunc(t *testing.T) {
	var called bool
	f := ToolHandlerFunc(func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
		called = true
		return json.RawMessage(`{"result": "ok"}`), nil
	})

	ctx := context.Background()
	input := json.RawMessage(`{}`)

	output, err := f.Execute(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("expected function to be called")
	}

	if string(output) != `{"result": "ok"}` {
		t.Errorf("expected output %q, got %q", `{"result": "ok"}`, string(output))
	}

	schema := f.Schema()
	if schema == nil {
		t.Error("expected non-nil schema")
	}
}

func TestStaticSchemaTool(t *testing.T) {
	schema := json.RawMessage(`{"type": "object", "properties": {"message": {"type": "string"}}}`)

	var called bool
	f := ToolHandlerFunc(func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
		called = true
		return json.RawMessage(`{"result": "ok"}`), nil
	})

	tool := NewStaticSchemaTool(schema, f)

	// Test execution
	ctx := context.Background()
	output, err := tool.Execute(ctx, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("expected function to be called")
	}

	if string(output) != `{"result": "ok"}` {
		t.Errorf("expected output %q, got %q", `{"result": "ok"}`, string(output))
	}

	// Test schema
	toolSchema := tool.Schema()
	if string(toolSchema) != string(schema) {
		t.Errorf("expected schema %q, got %q", string(schema), string(toolSchema))
	}
}

func TestAgentTool(t *testing.T) {
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

	tool := NewAgentTool(program)

	// Test schema
	schema := tool.Schema()
	if schema == nil {
		t.Error("expected non-nil schema")
	}

	expectedSchema := prog.InputSchema
	if string(schema) != string(expectedSchema) {
		t.Errorf("expected schema %q, got %q", string(expectedSchema), string(schema))
	}

	// Test execute (will fail without API key, but we can test the interface)
	ctx := context.Background()
	_, err = tool.Execute(ctx, json.RawMessage(`{"message": "test"}`))
	// This will fail because we don't have an API key, but that's expected
	// We're just testing that the method exists and has the right signature
	_ = err
}

// Helper types for testing

type mockAgentLoop struct {
	executeCalled bool
	lastInput     json.RawMessage
	lastOutput    json.RawMessage
	lastError     error
}

func (m *mockAgentLoop) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	m.executeCalled = true
	m.lastInput = input
	return m.lastOutput, m.lastError
}

func (m *mockAgentLoop) Metrics() Metrics {
	return Metrics{}
}

type mockToolHandler struct {
	executeCalled bool
	lastInput     json.RawMessage
	lastOutput    json.RawMessage
	lastError     error
	schema        json.RawMessage
}

func (m *mockToolHandler) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	m.executeCalled = true
	m.lastInput = input
	return m.lastOutput, m.lastError
}

func (m *mockToolHandler) Schema() json.RawMessage {
	return m.schema
}
