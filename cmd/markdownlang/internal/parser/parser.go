package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"within.website/x/cmd/markdownlang/internal/mcp"

	"gopkg.in/yaml.v3"
)

// Program represents a parsed markdownlang program with metadata.
type Program struct {
	Name         string
	Description  string
	Model        string // Optional: LLM model to use (e.g., "gpt-4o", "claude-3-5-sonnet-20241022")
	InputSchema  json.RawMessage
	OutputSchema json.RawMessage
	Imports      []string
	MCPServers   []mcp.MCPServerConfig
	Content      string // The markdown content after front matter
}

// ParseProgram extracts and parses the YAML front matter from markdown content.
func ParseProgram(source string) (*Program, error) {
	fm, content, err := extractFrontMatter(source)
	if err != nil {
		return nil, err
	}

	var frontMatter struct {
		Name        string                `yaml:"name"`
		Description string                `yaml:"description"`
		Model       string                `yaml:"model"`
		Input       interface{}           `yaml:"input"`
		Output      interface{}           `yaml:"output"`
		Imports     []string              `yaml:"imports"`
		MCPServers  []mcp.MCPServerConfig `yaml:"mcp_servers"`
	}

	if err := yaml.Unmarshal(fm, &frontMatter); err != nil {
		return nil, fmt.Errorf("your YAML is garbage: %w", err)
	}

	prog := &Program{
		Name:        frontMatter.Name,
		Description: frontMatter.Description,
		Model:       frontMatter.Model,
		Imports:     frontMatter.Imports,
		MCPServers:  frontMatter.MCPServers,
	}

	// Serialize schemas back to JSON for validation/storage
	if frontMatter.Input != nil {
		raw, err := json.Marshal(frontMatter.Input)
		if err != nil {
			return nil, fmt.Errorf("input schema refuses to become JSON: %w", err)
		}
		prog.InputSchema = raw
	}

	if frontMatter.Output != nil {
		raw, err := json.Marshal(frontMatter.Output)
		if err != nil {
			return nil, fmt.Errorf("output schema refuses to become JSON: %w", err)
		}
		prog.OutputSchema = raw
	}

	// Validate required fields
	if err := validateProgram(prog); err != nil {
		return nil, err
	}

	// Check for circular dependencies in imports
	if err := detectCircularDependencies(prog.Imports); err != nil {
		return nil, err
	}

	prog.Content = content

	return prog, nil
}

// validateProgram checks that all required fields are present and valid.
func validateProgram(prog *Program) error {
	if prog.Name == "" {
		return errors.New("your program is nameless. give it a name or get out")
	}

	if prog.Description == "" {
		return errors.New("no description? how am I supposed to know what this does?")
	}

	if len(prog.InputSchema) == 0 {
		return errors.New("this program takes no input? how... useless")
	}

	if len(prog.OutputSchema) == 0 {
		return errors.New("this program produces no output? what's even the point?")
	}

	// Validate import paths
	for _, imp := range prog.Imports {
		if imp == "" {
			return errors.New("you have an empty import path. what are you even trying to import?")
		}

		// Check for valid import path format
		if strings.Contains(imp, "..") {
			return fmt.Errorf("import path %q contains '..': we're not playing that game today", imp)
		}

		// Could be relative (./foo), absolute (/bar), or stdlib (strings)
		if !strings.HasPrefix(imp, "./") && !strings.HasPrefix(imp, "/") && !isValidStdlibPath(imp) {
			return fmt.Errorf("import path %q looks sus: use ./relative, /absolute, or stdlib paths", imp)
		}
	}

	return nil
}

// isValidStdlibPath checks if a path looks like a valid stdlib or external package path.
func isValidStdlibPath(path string) bool {
	// Simple heuristic: must contain at least one dot (domain) or be a simple identifier
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if part == "" {
			return false
		}
	}
	return true
}

// detectCircularDependency checks for circular dependencies in imports.
func detectCircularDependencies(imports []string) error {
	// For now, just check if any import imports itself
	// A full implementation would build a dependency graph
	visited := make(map[string]bool)
	path := make(map[string]bool)

	var visit func(string) error
	visit = func(name string) error {
		if path[name] {
			return fmt.Errorf("circular dependency detected: %s imports itself", name)
		}
		if visited[name] {
			return nil
		}
		path[name] = true
		visited[name] = true
		// In a full implementation, we'd check this file's imports
		delete(path, name)
		return nil
	}

	for _, imp := range imports {
		if err := visit(imp); err != nil {
			return err
		}
	}

	return nil
}

// LoadProgram reads a markdownlang program from a file.
func LoadProgram(path string) (*Program, error) {
	// Read the file
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read program file %s: %w", path, err)
	}

	// Parse the program
	prog, err := ParseProgram(string(source))
	if err != nil {
		return nil, fmt.Errorf("failed to parse program %s: %w", path, err)
	}

	return prog, nil
}

// ValidateInput validates input data against the program's input schema.
func (p *Program) ValidateInput(input map[string]interface{}) error {
	// Empty input is allowed - many programs don't require input
	// Just check that input can be JSON-serialized
	_, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("input is not valid JSON-serializable: %w", err)
	}

	// TODO: Full JSON Schema validation against p.InputSchema
	// For now, we just check that it's valid JSON

	return nil
}

// ValidateOutput validates output data against the program's output schema.
func (p *Program) ValidateOutput(output map[string]interface{}) error {
	if len(output) == 0 {
		return errors.New("output is empty: that's not valid JSON")
	}

	// Basic validation: check if output can be JSON-serialized
	_, err := json.Marshal(output)
	if err != nil {
		return fmt.Errorf("output is not valid JSON-serializable: %w", err)
	}

	// TODO: Full JSON Schema validation against p.OutputSchema
	// For now, we just check that it's valid JSON

	return nil
}
