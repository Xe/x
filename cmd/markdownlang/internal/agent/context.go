package agent

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"within.website/x/cmd/markdownlang/internal/parser"
)

// ExecutionContext holds state shared across agent invocations.
//
// This is useful for managing imports, tool availability, and other
// resources that should be shared across multiple agent executions.
type ExecutionContext struct {
	mu sync.RWMutex

	// programs is a cache of loaded programs by name/path.
	programs map[string]*Program

	// imported is a map of import paths to their agent loops.
	imported map[string]AgentLoop

	// tools is a map of available tool names to their handlers.
	tools map[string]ToolHandler

	// config holds configuration for creating new agents.
	config *ExecutionContextConfig
}

// ExecutionContextConfig configures an ExecutionContext.
type ExecutionContextConfig struct {
	// DefaultModel is the default OpenAI model to use.
	DefaultModel string

	// DefaultTemperature is the default temperature for responses.
	DefaultTemperature float64

	// APIKey is the OpenAI API key to use.
	APIKey string

	// BaseURL is the OpenAI base URL to use.
	BaseURL string
}

// NewExecutionContext creates a new ExecutionContext.
func NewExecutionContext(config *ExecutionContextConfig) *ExecutionContext {
	if config == nil {
		config = &ExecutionContextConfig{}
	}

	return &ExecutionContext{
		programs: make(map[string]*Program),
		imported: make(map[string]AgentLoop),
		tools:    make(map[string]ToolHandler),
		config:   config,
	}
}

// LoadProgram loads a program from source and returns an AgentLoop.
func (ec *ExecutionContext) LoadProgram(source string) (AgentLoop, error) {
	prog, err := parser.ParseProgram(source)
	if err != nil {
		return nil, err
	}

	program, err := NewProgram(prog,
		WithAPIKey(ec.config.APIKey),
		WithBaseURL(ec.config.BaseURL),
	)
	if err != nil {
		return nil, err
	}

	ec.mu.Lock()
	ec.programs[prog.Name] = program
	ec.mu.Unlock()

	return program, nil
}

// GetProgram retrieves a program by name from the cache.
func (ec *ExecutionContext) GetProgram(name string) (*Program, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	prog, ok := ec.programs[name]
	return prog, ok
}

// RegisterImport registers an imported agent loop by its import path.
func (ec *ExecutionContext) RegisterImport(path string, loop AgentLoop) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.imported[path] = loop
	slog.Info("registered import", "path", path)
}

// GetImport retrieves an imported agent loop by path.
func (ec *ExecutionContext) GetImport(path string) (AgentLoop, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	loop, ok := ec.imported[path]
	return loop, ok
}

// RegisterTool registers a tool handler.
func (ec *ExecutionContext) RegisterTool(name string, handler ToolHandler) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.tools[name] = handler
	slog.Info("registered tool", "name", name)
}

// GetTool retrieves a tool handler by name.
func (ec *ExecutionContext) GetTool(name string) (ToolHandler, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	handler, ok := ec.tools[name]
	return handler, ok
}

// ListTools returns all registered tool names.
func (ec *ExecutionContext) ListTools() []string {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	names := make([]string, 0, len(ec.tools))
	for name := range ec.tools {
		names = append(names, name)
	}
	return names
}

// ToolHandler defines the interface for handling tool calls.
//
// Tool handlers are called when the LLM requests to use a tool.
// They execute the tool and return the result.
type ToolHandler interface {
	// Execute runs the tool with the given input.
	Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error)

	// Schema returns the JSON Schema for the tool's input.
	Schema() json.RawMessage
}

// ToolHandlerFunc is a function adapter for ToolHandler.
type ToolHandlerFunc func(ctx context.Context, input json.RawMessage) (json.RawMessage, error)

// Execute calls the underlying function.
func (f ToolHandlerFunc) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	return f(ctx, input)
}

// Schema returns a placeholder schema for function tools.
// The actual schema should be provided by the tool implementation.
func (f ToolHandlerFunc) Schema() json.RawMessage {
	return json.RawMessage(`{"type": "object", "properties": {}}`)
}

// StaticSchemaTool wraps a ToolHandlerFunc with a static schema.
type StaticSchemaTool struct {
	F      ToolHandlerFunc
	schema json.RawMessage
}

// NewStaticSchemaTool creates a new tool with a static schema.
func NewStaticSchemaTool(schema json.RawMessage, f ToolHandlerFunc) *StaticSchemaTool {
	return &StaticSchemaTool{
		F:      f,
		schema: schema,
	}
}

// Execute delegates to the underlying function.
func (t *StaticSchemaTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	return t.F(ctx, input)
}

// Schema returns the static schema.
func (t *StaticSchemaTool) Schema() json.RawMessage {
	return t.schema
}

// AgentTool wraps an AgentLoop as a ToolHandler.
type AgentTool struct {
	agent AgentLoop
}

// NewAgentTool creates a new tool from an AgentLoop.
func NewAgentTool(agent AgentLoop) *AgentTool {
	return &AgentTool{agent: agent}
}

// Execute calls the agent's Execute method.
func (t *AgentTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	return t.agent.Execute(ctx, input)
}

// Schema returns the input schema from the agent's program.
func (t *AgentTool) Schema() json.RawMessage {
	if prog, ok := t.agent.(*Program); ok {
		return prog.InputSchema
	}
	return json.RawMessage(`{"type": "object", "properties": {}}`)
}
