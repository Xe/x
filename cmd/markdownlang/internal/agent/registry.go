// Package agent implements a registry for loading and managing imported markdownlang programs.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"within.website/x/cmd/markdownlang/internal/parser"
)

// Registry manages loading and caching of imported markdownlang programs.
//
// The registry is responsible for:
// - Loading programs from file paths
// - Creating executable agents from programs
// - Caching programs for reuse
// - Tracking circular dependencies
type Registry struct {
	mu       sync.RWMutex
	programs map[string]*Program  // Cached programs by path
	agents   map[string]AgentLoop // Executable agents by name
	loading  map[string]bool      // Tracks programs currently being loaded (for circular dependency detection)
	apiKey   string
	baseURL  string
	model    string
	baseDir  string // Base directory for resolving relative paths
}

// RegistryConfig configures a new Registry.
type RegistryConfig struct {
	// APIKey is the OpenAI API key to use for imported agents.
	APIKey string

	// BaseURL is the OpenAI base URL.
	BaseURL string

	// Model is the default model to use for imported agents.
	Model string

	// BaseDir is the base directory for resolving relative import paths.
	// If empty, the current working directory is used.
	BaseDir string
}

// NewRegistry creates a new Registry with the given configuration.
func NewRegistry(config *RegistryConfig) *Registry {
	if config == nil {
		config = &RegistryConfig{}
	}

	var baseDir string
	if config.BaseDir != "" {
		baseDir = config.BaseDir
	} else {
		baseDir = "."
	}

	return &Registry{
		programs: make(map[string]*Program),
		agents:   make(map[string]AgentLoop),
		loading:  make(map[string]bool),
		apiKey:   config.APIKey,
		baseURL:  config.BaseURL,
		model:    config.Model,
		baseDir:  baseDir,
	}
}

// LoadImport loads a program by its import path and returns an AgentLoop.
//
// The import path can be:
// - Relative: ./other.md or ../utils/helper.md
// - Absolute: /path/to/program.md
// - Named: stdlib:name (not yet implemented)
//
// Circular dependencies are detected and result in an error.
func (r *Registry) LoadImport(ctx context.Context, importPath string) (AgentLoop, error) {
	// Check if already loaded as an agent
	r.mu.RLock()
	if agent, ok := r.agents[importPath]; ok {
		r.mu.RUnlock()
		slog.Debug("agent cache hit", "import", importPath)
		return agent, nil
	}
	r.mu.RUnlock()

	// Resolve the file path
	filePath, err := r.resolvePath(importPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve import path %q: %w", importPath, err)
	}

	// Check if already loaded as a program (need to create agent)
	r.mu.RLock()
	if prog, ok := r.programs[filePath]; ok {
		r.mu.RUnlock()
		// Create agent from cached program (convert agent.Program to parser.Program)
		parsedProg := &prog.Program
		agent, err := r.createAgent(parsedProg)
		if err != nil {
			return nil, err
		}
		r.mu.Lock()
		r.agents[importPath] = agent
		r.mu.Unlock()
		return agent, nil
	}
	r.mu.RUnlock()

	// Load the program from file
	return r.loadProgram(ctx, importPath, filePath)
}

// loadProgram loads a program from file and creates an agent.
func (r *Registry) loadProgram(ctx context.Context, importPath, filePath string) (AgentLoop, error) {
	// Check for circular dependencies
	r.mu.Lock()
	if r.loading[filePath] {
		r.mu.Unlock()
		return nil, fmt.Errorf("circular dependency detected: %s is already being loaded", filePath)
	}
	r.loading[filePath] = true
	r.mu.Unlock()

	// Ensure we clear the loading flag
	defer func() {
		r.mu.Lock()
		delete(r.loading, filePath)
		r.mu.Unlock()
	}()

	// Load and parse the program
	prog, err := parser.LoadProgram(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load program from %s: %w", filePath, err)
	}

	// Cache the program
	r.mu.Lock()
	r.programs[filePath] = &Program{Program: *prog} // Convert parser.Program to agent.Program
	r.mu.Unlock()

	// Create the agent
	agent, err := r.createAgent(prog)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent for %s: %w", prog.Name, err)
	}

	// Cache the agent
	r.mu.Lock()
	r.agents[importPath] = agent
	r.mu.Unlock()

	slog.Info("loaded imported agent",
		"name", prog.Name,
		"import", importPath,
		"file", filePath)

	return agent, nil
}

// createAgent creates an executable AgentLoop from a parsed program.
func (r *Registry) createAgent(prog *parser.Program) (*Program, error) {
	opts := []ProgramOption{}

	// Set API key if provided
	if r.apiKey != "" {
		opts = append(opts, WithAPIKey(r.apiKey))
	}

	// Set base URL if provided
	if r.baseURL != "" {
		opts = append(opts, WithBaseURL(r.baseURL))
	}

	// Set model if provided, otherwise use program's model or default
	model := r.model
	if model == "" {
		model = prog.Model
	}
	if model != "" {
		opts = append(opts, WithModel(model))
	}

	return NewProgram(prog, opts...)
}

// LoadImports loads multiple import paths and returns a map of import paths to agents.
func (r *Registry) LoadImports(ctx context.Context, importPaths []string) (map[string]AgentLoop, error) {
	agents := make(map[string]AgentLoop)
	var errs []error

	for _, importPath := range importPaths {
		agent, err := r.LoadImport(ctx, importPath)
		if err != nil {
			slog.Warn("failed to load import", "import", importPath, "error", err)
			errs = append(errs, err)
			continue
		}
		agents[importPath] = agent
	}

	if len(errs) > 0 {
		return agents, fmt.Errorf("failed to load some imports: %v", errs)
	}

	return agents, nil
}

// GetAgent retrieves an imported agent by its import path.
func (r *Registry) GetAgent(importPath string) (AgentLoop, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agent, ok := r.agents[importPath]
	return agent, ok
}

// ListAgents returns all loaded agent import paths.
func (r *Registry) ListAgents() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	paths := make([]string, 0, len(r.agents))
	for path := range r.agents {
		paths = append(paths, path)
	}
	return paths
}

// resolvePath resolves an import path to an absolute file path.
func (r *Registry) resolvePath(importPath string) (string, error) {
	// Handle stdlib imports (not yet implemented)
	if strings.HasPrefix(importPath, "stdlib:") {
		return "", fmt.Errorf("stdlib imports are not yet implemented: %s", importPath)
	}

	// Handle absolute paths
	if filepath.IsAbs(importPath) {
		if err := r.validateFilePath(importPath); err != nil {
			return "", err
		}
		return importPath, nil
	}

	// Handle relative paths
	if !strings.HasPrefix(importPath, "./") && !strings.HasPrefix(importPath, "../") {
		return "", fmt.Errorf("invalid import path format: %q (must start with ./, ../, or be absolute)", importPath)
	}

	// Resolve relative to base directory
	filePath := filepath.Join(r.baseDir, importPath)

	// Clean the path
	filePath = filepath.Clean(filePath)

	if err := r.validateFilePath(filePath); err != nil {
		return "", err
	}

	return filePath, nil
}

// validateFilePath checks that a file path is valid and safe.
func (r *Registry) validateFilePath(filePath string) error {
	// Check for path traversal attempts
	if strings.Contains(filePath, "..") {
		// Clean the path and check if it still tries to escape
		cleaned := filepath.Clean(filePath)
		absPath, err := filepath.Abs(cleaned)
		if err != nil {
			return fmt.Errorf("invalid file path: %w", err)
		}

		// Check if the resolved path is within allowed bounds
		baseAbs, err := filepath.Abs(r.baseDir)
		if err == nil {
			rel, err := filepath.Rel(baseAbs, absPath)
			if err == nil && strings.HasPrefix(rel, "..") {
				return fmt.Errorf("file path escapes base directory: %s", filePath)
			}
		}
	}

	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", filePath)
		}
		return fmt.Errorf("failed to access file: %w", err)
	}

	// Check if it's a file (not a directory)
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", filePath)
	}

	// Check file extension
	if filepath.Ext(filePath) != ".md" {
		return fmt.Errorf("import file must have .md extension: %s", filePath)
	}

	return nil
}

// CreateToolForAgent wraps an imported agent as a ToolHandler.
func (r *Registry) CreateToolForAgent(importPath string) (*AgentTool, error) {
	agent, ok := r.GetAgent(importPath)
	if !ok {
		return nil, fmt.Errorf("agent not found for import path: %s", importPath)
	}

	return NewAgentTool(agent), nil
}

// GetToolSchemas returns JSON schemas for all imported agents as tools.
func (r *Registry) GetToolSchemas(ctx context.Context) (map[string]json.RawMessage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schemas := make(map[string]json.RawMessage)

	for importPath, agent := range r.agents {
		// Get the program to extract metadata
		if prog, ok := agent.(*Program); ok {
			// Create a tool schema
			toolSchema := map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"input": map[string]interface{}{
						"type":                 "object",
						"description":          fmt.Sprintf("Input for agent %s", prog.Name),
						"additionalProperties": true,
					},
				},
				"required": []string{"input"},
			}

			schemaBytes, err := json.Marshal(toolSchema)
			if err != nil {
				slog.Warn("failed to marshal tool schema", "agent", prog.Name, "error", err)
				continue
			}

			schemas[importPath] = json.RawMessage(schemaBytes)
		}
	}

	return schemas, nil
}

// AgentToolWithRegistry extends AgentTool with registry information.
type AgentToolWithRegistry struct {
	*AgentTool
	ImportPath string
	Name       string
}

// NewAgentToolWithRegistry creates a new tool wrapper with metadata.
func NewAgentToolWithRegistry(agent AgentLoop, importPath, name string) *AgentToolWithRegistry {
	return &AgentToolWithRegistry{
		AgentTool:  NewAgentTool(agent),
		ImportPath: importPath,
		Name:       name,
	}
}

// ToolDefinition returns a tool definition for function calling.
func (t *AgentToolWithRegistry) ToolDefinition() map[string]interface{} {
	def := map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        t.Name,
			"description": fmt.Sprintf("Call agent %s", t.ImportPath),
			"parameters":  t.Schema(),
		},
	}
	return def
}

// ValidateImports checks that all imports in a program are valid and can be loaded.
func (r *Registry) ValidateImports(ctx context.Context, imports []string) error {
	var errs []error

	for _, imp := range imports {
		// Try to load the import
		_, err := r.LoadImport(ctx, imp)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", imp, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("import validation failed: %v", errs)
	}

	return nil
}

// Clear clears all cached programs and agents.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.programs = make(map[string]*Program)
	r.agents = make(map[string]AgentLoop)
	r.loading = make(map[string]bool)
}

// Stats returns statistics about the registry.
func (r *Registry) Stats() RegistryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return RegistryStats{
		ProgramCount:      len(r.programs),
		AgentCount:        len(r.agents),
		LoadingCount:      len(r.loading),
		CachedImportPaths: r.ListAgents(),
	}
}

// RegistryStats contains statistics about the registry.
type RegistryStats struct {
	ProgramCount      int      `json:"program_count"`
	AgentCount        int      `json:"agent_count"`
	LoadingCount      int      `json:"loading_count"`
	CachedImportPaths []string `json:"cached_import_paths"`
}

// GlobalOpenAIClient is a helper for creating OpenAI clients with the registry's configuration.
func (r *Registry) GlobalOpenAIClient() openai.Client {
	opts := []option.RequestOption{}
	if r.apiKey != "" {
		opts = append(opts, option.WithAPIKey(r.apiKey))
	}
	if r.baseURL != "" {
		opts = append(opts, option.WithBaseURL(r.baseURL))
	}

	if len(opts) == 0 {
		return openai.NewClient()
	}

	return openai.NewClient(opts...)
}

// LoadImportsFromProgram loads all imports defined in a program's front matter.
func (r *Registry) LoadImportsFromProgram(ctx context.Context, prog *parser.Program) (map[string]AgentLoop, error) {
	if len(prog.Imports) == 0 {
		return make(map[string]AgentLoop), nil
	}

	slog.Info("loading imports for program",
		"program", prog.Name,
		"import_count", len(prog.Imports))

	agents, err := r.LoadImports(ctx, prog.Imports)
	if err != nil {
		return nil, fmt.Errorf("failed to load imports for program %s: %w", prog.Name, err)
	}

	slog.Info("loaded imports for program",
		"program", prog.Name,
		"loaded_count", len(agents))

	return agents, nil
}

// CreateToolHandlers creates ToolHandler instances for all imported agents.
func (r *Registry) CreateToolHandlers(ctx context.Context, prog *parser.Program) (map[string]ToolHandler, error) {
	agents, err := r.LoadImportsFromProgram(ctx, prog)
	if err != nil {
		return nil, err
	}

	handlers := make(map[string]ToolHandler)
	for importPath, agent := range agents {
		if prog, ok := agent.(*Program); ok {
			// Use the program's name as the tool name
			toolName := prog.Name
			handlers[toolName] = NewStaticSchemaTool(
				prog.InputSchema,
				func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
					return agent.Execute(ctx, input)
				},
			)

			slog.Debug("created tool handler for imported agent",
				"tool", toolName,
				"import", importPath)
		}
	}

	return handlers, nil
}

// AgentCallResult represents the result of calling an imported agent.
type AgentCallResult struct {
	// ImportPath is the path used to import the agent.
	ImportPath string `json:"import_path"`

	// AgentName is the name of the agent that was called.
	AgentName string `json:"agent_name"`

	// Input is the input that was sent to the agent.
	Input json.RawMessage `json:"input"`

	// Output is the output returned by the agent.
	Output json.RawMessage `json:"output"`

	// Error contains any error that occurred during the call.
	Error string `json:"error,omitempty"`

	// Duration is how long the agent call took.
	Duration string `json:"duration"`

	// Metrics contains runtime metrics from the agent execution.
	Metrics Metrics `json:"metrics"`
}

// CallAgent executes an imported agent with the given input.
func (r *Registry) CallAgent(ctx context.Context, importPath string, input json.RawMessage) (*AgentCallResult, error) {
	agent, ok := r.GetAgent(importPath)
	if !ok {
		return nil, fmt.Errorf("agent not found for import path: %s", importPath)
	}

	// Get the agent name
	var agentName string
	if prog, ok := agent.(*Program); ok {
		agentName = prog.Name
	} else {
		agentName = importPath
	}

	result := &AgentCallResult{
		ImportPath: importPath,
		AgentName:  agentName,
		Input:      input,
	}

	// Execute the agent
	output, err := agent.Execute(ctx, input)
	if err != nil {
		result.Error = err.Error()
		return result, fmt.Errorf("agent execution failed for %s: %w", importPath, err)
	}

	result.Output = output
	result.Metrics = agent.Metrics()

	return result, nil
}

// ValidateProgramImports validates that all imports in a program can be loaded.
func ValidateProgramImports(ctx context.Context, prog *parser.Program, config *RegistryConfig) error {
	registry := NewRegistry(config)
	defer registry.Clear()

	return registry.ValidateImports(ctx, prog.Imports)
}

// errors is a helper for collecting multiple errors.
type importError struct {
	importPath string
	err        error
}

func (e *importError) Error() string {
	return fmt.Sprintf("%s: %s", e.importPath, e.err)
}

// JoinErrors joins multiple import errors into a single error.
func JoinErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}

	var msg strings.Builder
	msg.WriteString("multiple errors occurred:")
	for _, err := range errs {
		msg.WriteString("\n  - ")
		msg.WriteString(err.Error())
	}

	return errors.New(msg.String())
}
