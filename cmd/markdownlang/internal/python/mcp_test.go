package python

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestExecute(t *testing.T) {
	ctx := context.Background()

	t.Run("successful execution", func(t *testing.T) {
		input := Input{
			Code: `print("Hello from MCP!")`,
		}

		req := &mcp.CallToolRequest{}

		result, err := Execute(ctx, req, input)
		if err != nil {
			t.Errorf("Execute() error = %v", err)
			return
		}

		if result == nil {
			t.Error("Execute() result should not be nil")
			return
		}

		if len(result.Content) == 0 {
			t.Error("Execute() result should have content")
			return
		}

		// Verify the content contains the expected output
		textContent, ok := result.Content[0].(*mcp.TextContent)
		if !ok {
			t.Error("Execute() content should be TextContent")
			return
		}

		if textContent.Text == "" {
			t.Error("Execute() text content should not be empty")
		}

		t.Logf("Execute() returned: %s", textContent.Text)
	})

	t.Run("execution with error", func(t *testing.T) {
		input := Input{
			Code: `print("unclosed string`,
		}

		req := &mcp.CallToolRequest{}

		result, err := Execute(ctx, req, input)
		// Execute should not return an error even if Python fails
		// Instead, the error should be in the result
		if err != nil {
			t.Errorf("Execute() should not return error for Python failures, got: %v", err)
			return
		}

		if result == nil {
			t.Error("Execute() result should not be nil even on Python error")
			return
		}

		// The result should contain error information
		textContent, ok := result.Content[0].(*mcp.TextContent)
		if !ok {
			t.Error("Execute() content should be TextContent")
			return
		}

		if textContent.Text == "" {
			t.Error("Execute() should return error information in text content")
		}
	})
}

func TestExecuteWithConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("with custom config", func(t *testing.T) {
		input := Input{
			Code: `print("custom config")`,
		}

		req := &mcp.CallToolRequest{}

		cfg := Config{
			Timeout: 10 * 1000000000, // 10 seconds in nanoseconds
		}

		result, err := ExecuteWithConfig(ctx, req, input, cfg)
		if err != nil {
			t.Errorf("ExecuteWithConfig() error = %v", err)
			return
		}

		if result == nil {
			t.Error("ExecuteWithConfig() result should not be nil")
			return
		}

		textContent, ok := result.Content[0].(*mcp.TextContent)
		if !ok {
			t.Error("ExecuteWithConfig() content should be TextContent")
			return
		}

		if textContent.Text == "" {
			t.Error("ExecuteWithConfig() text content should not be empty")
		}
	})
}

func TestTool(t *testing.T) {
	tool := Tool()

	if tool == nil {
		t.Fatal("Tool() should not return nil")
	}

	if tool.Name != "python-interpreter" {
		t.Errorf("Tool().Name = %q, want %q", tool.Name, "python-interpreter")
	}

	if tool.Description == "" {
		t.Error("Tool().Description should not be empty")
	}

	if tool.InputSchema == nil {
		t.Error("Tool().InputSchema should not be nil")
	}

	// Verify input schema structure
	schema, ok := tool.InputSchema.(map[string]interface{})
	if !ok {
		t.Error("Tool().InputSchema should be a map")
		return
	}

	if schema["type"] != "object" {
		t.Errorf("Tool().InputSchema type = %v, want object", schema["type"])
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Error("Tool().InputSchema properties should be a map")
		return
	}

	if _, hasCode := properties["code"]; !hasCode {
		t.Error("Tool().InputSchema should have 'code' property")
	}

	required, ok := schema["required"].([]string)
	if !ok {
		// The JSON schema might unmarshal as []interface{}
		requiredInterface, ok := schema["required"].([]interface{})
		if !ok {
			t.Error("Tool().InputSchema required should be a slice")
			return
		}
		if len(requiredInterface) == 0 {
			t.Error("Tool().InputSchema should have required fields")
		}
		return
	}

	if len(required) == 0 {
		t.Error("Tool().InputSchema should have required fields")
	}
}

func TestInstruction(t *testing.T) {
	instruction := Instruction()

	if instruction == "" {
		t.Error("Instruction() should not return empty string")
	}

	// Verify the instruction mentions key concepts
	keyTerms := []string{
		"python-interpreter",
		"WebAssembly",
		"calculations",
		"data processing",
		"algorithms",
	}

	for _, term := range keyTerms {
		if !contains(instruction, term) {
			t.Errorf("Instruction() should mention %q", term)
		}
	}

	t.Logf("Instruction length: %d characters", len(instruction))
}

func TestInput(t *testing.T) {
	t.Run("valid input", func(t *testing.T) {
		input := Input{
			Code: `print("test")`,
		}

		if input.Code == "" {
			t.Error("Input.Code should not be empty")
		}
	})

	t.Run("empty code", func(t *testing.T) {
		input := Input{
			Code: "",
		}

		// Empty code is valid input (it will just produce no output)
		if input.Code != "" {
			t.Error("Input.Code should be empty")
		}
	})
}

func TestOutput(t *testing.T) {
	t.Run("output with result", func(t *testing.T) {
		result := &Result{
			Stdout:   "test output",
			Stderr:   "",
			Error:    "",
			Duration: 1000000,
		}

		output := Output{
			Result: result,
		}

		if output.Result == nil {
			t.Error("Output.Result should not be nil")
		}

		if output.Result.Stdout != "test output" {
			t.Errorf("Output.Result.Stdout = %q, want %q", output.Result.Stdout, "test output")
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[0:1] == substr[0:1] && contains(s[1:], substr[1:])) ||
			(len(s) > 1 && contains(s[1:], substr))))
}
