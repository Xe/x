package python

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestExecute_ErrorPaths(t *testing.T) {
	ctx := context.Background()

	t.Run("partial result with error", func(t *testing.T) {
		// Code that produces output before failing
		input := Input{
			Code: `print("before error")
raise ValueError("test error")`,
		}

		req := &mcp.CallToolRequest{}

		result, err := Execute(ctx, req, input)
		// Execute should not return an error
		if err != nil {
			t.Errorf("Execute() should not return error, got: %v", err)
		}

		if result == nil {
			t.Fatal("Execute() result should not be nil")
		}

		// Verify we got content
		if len(result.Content) == 0 {
			t.Error("Execute() should return content even on Python errors")
		}

		// Verify the content contains JSON with result
		textContent, ok := result.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatal("Execute() content should be TextContent")
		}

		// Parse the JSON to verify structure
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &data); err != nil {
			t.Errorf("Execute() should return valid JSON: %v", err)
		}

		resultData, ok := data["result"].(map[string]interface{})
		if !ok {
			t.Fatal("Execute() result should contain 'result' field")
		}

		// Verify we got stdout before the error
		stdout, _ := resultData["stdout"].(string)
		if !strings.Contains(stdout, "before error") {
			t.Error("Execute() should capture stdout before error")
		}
	})

	t.Run("syntax error with partial output", func(t *testing.T) {
		input := Input{
			Code: `print("first")
print("unclosed string`,
		}

		req := &mcp.CallToolRequest{}

		result, err := Execute(ctx, req, input)
		if err != nil {
			t.Errorf("Execute() should not return error, got: %v", err)
		}

		if result == nil {
			t.Fatal("Execute() result should not be nil")
		}

		// Should still get content
		if len(result.Content) == 0 {
			t.Error("Execute() should return content for syntax errors")
		}
	})

	t.Run("empty code", func(t *testing.T) {
		input := Input{
			Code: ``,
		}

		req := &mcp.CallToolRequest{}

		result, err := Execute(ctx, req, input)
		if err != nil {
			t.Errorf("Execute() with empty code should not error: %v", err)
		}

		if result == nil {
			t.Fatal("Execute() result should not be nil for empty code")
		}

		// Should return valid JSON result
		textContent, ok := result.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatal("Execute() content should be TextContent")
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &data); err != nil {
			t.Errorf("Execute() should return valid JSON for empty code: %v", err)
		}
	})
}

func TestExecuteWithConfig_ErrorPaths(t *testing.T) {
	ctx := context.Background()

	t.Run("error with custom timeout", func(t *testing.T) {
		input := Input{
			Code: `raise RuntimeError("test")`,
		}

		req := &mcp.CallToolRequest{}

		cfg := Config{
			Timeout: 1,
			Stdin:   "test input",
		}

		result, err := ExecuteWithConfig(ctx, req, input, cfg)
		if err != nil {
			t.Errorf("ExecuteWithConfig() should not return error, got: %v", err)
		}

		if result == nil {
			t.Fatal("ExecuteWithConfig() result should not be nil")
		}

		// Verify error is captured in result
		textContent, ok := result.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatal("ExecuteWithConfig() content should be TextContent")
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &data); err != nil {
			t.Errorf("ExecuteWithConfig() should return valid JSON: %v", err)
		}

		resultData, ok := data["result"].(map[string]interface{})
		if !ok {
			t.Fatal("ExecuteWithConfig() result should contain 'result' field")
		}

		// Should have error information
		errorMsg, _ := resultData["error"].(string)
		if errorMsg == "" {
			stderr, _ := resultData["stderr"].(string)
			if stderr == "" {
				t.Error("ExecuteWithConfig() should capture error in error or stderr field")
			}
		}
	})

	t.Run("with stdin on error", func(t *testing.T) {
		input := Input{
			Code: `1 / 0`,
		}

		req := &mcp.CallToolRequest{}

		cfg := Config{
			Stdin: "some input",
		}

		result, err := ExecuteWithConfig(ctx, req, input, cfg)
		if err != nil {
			t.Errorf("ExecuteWithConfig() should not return error: %v", err)
		}

		if result == nil {
			t.Fatal("ExecuteWithConfig() result should not be nil")
		}
	})

	t.Run("with memory limit", func(t *testing.T) {
		input := Input{
			Code: `print("test")`,
		}

		req := &mcp.CallToolRequest{}

		cfg := Config{
			MemoryLimit: 1024, // Very small limit
		}

		result, err := ExecuteWithConfig(ctx, req, input, cfg)
		// Should work for simple code even with small limit
		if err != nil {
			t.Errorf("ExecuteWithConfig() with small memory limit should work for simple code: %v", err)
		}

		if result == nil {
			t.Fatal("ExecuteWithConfig() result should not be nil")
		}
	})
}

func TestTool_Comprehensive(t *testing.T) {
	tool := Tool()

	t.Run("tool structure validation", func(t *testing.T) {
		if tool.Name != "python-interpreter" {
			t.Errorf("Tool().Name = %q, want 'python-interpreter'", tool.Name)
		}

		if tool.Description == "" {
			t.Error("Tool().Description should not be empty")
		}

		// Verify description mentions key concepts
		keyTerms := []string{
			"secure",
			"wasm",
			"sandbox",
			"calculations",
			"data processing",
		}

		for _, term := range keyTerms {
			if !strings.Contains(strings.ToLower(tool.Description), term) {
				t.Errorf("Tool().Description should mention %q", term)
			}
		}
	})

	t.Run("input schema validation", func(t *testing.T) {
		schema, ok := tool.InputSchema.(map[string]interface{})
		if !ok {
			t.Fatal("Tool().InputSchema should be a map")
		}

		// Check type
		if schema["type"] != "object" {
			t.Errorf("Tool().InputSchema type = %v, want 'object'", schema["type"])
		}

		// Check properties
		properties, ok := schema["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("Tool().InputSchema properties should be a map")
		}

		// Check code property
		codeProp, ok := properties["code"].(map[string]interface{})
		if !ok {
			t.Fatal("Tool().InputSchema should have 'code' property")
		}

		if codeProp["type"] != "string" {
			t.Errorf("code property type = %v, want 'string'", codeProp["type"])
		}

		description, ok := codeProp["description"].(string)
		if !ok || description == "" {
			t.Error("code property should have a description")
		}

		// Check required fields
		required, ok := schema["required"].([]interface{})
		if !ok {
			requiredStr, ok := schema["required"].([]string)
			if !ok {
				t.Fatal("Tool().InputSchema required should be a slice")
			}
			if len(requiredStr) == 0 {
				t.Error("Tool().InputSchema should have required fields")
			}
		} else {
			if len(required) == 0 {
				t.Error("Tool().InputSchema should have required fields")
			}

			// Verify 'code' is required
			foundCode := false
			for _, r := range required {
				if r == "code" {
					foundCode = true
					break
				}
			}
			if !foundCode {
				t.Error("'code' should be in required fields")
			}
		}
	})
}

func TestInstruction_Comprehensive(t *testing.T) {
	instruction := Instruction()

	t.Run("instruction structure", func(t *testing.T) {
		if len(instruction) < 100 {
			t.Error("Instruction() should be detailed (at least 100 characters)")
		}

		// Count lines
		lines := strings.Split(instruction, "\n")
		if len(lines) < 5 {
			t.Error("Instruction() should have multiple lines")
		}
	})

	t.Run("instruction content", func(t *testing.T) {
		// Check for key information
		requiredPhrases := []string{
			"python-interpreter",
			"WebAssembly",
			"30 second",
			"128 MB",
			"calculations",
			"data processing",
			"algorithms",
			"json",
		}

		lowerInstruction := strings.ToLower(instruction)
		for _, phrase := range requiredPhrases {
			if !strings.Contains(lowerInstruction, strings.ToLower(phrase)) {
				t.Errorf("Instruction() should mention %q", phrase)
			}
		}
	})

	t.Run("instruction formatting", func(t *testing.T) {
		// Check for markdown-like formatting
		if !strings.Contains(instruction, "-") && !strings.Contains(instruction, "*") {
			t.Error("Instruction() should use some formatting (bullets, etc)")
		}
	})
}

func TestInputOutput_Structures(t *testing.T) {
	t.Run("Input struct", func(t *testing.T) {
		input := Input{
			Code: `print("test")`,
		}

		// Verify jsonschema tag is respected
		if input.Code == "" {
			t.Error("Input.Code should not be empty after initialization")
		}

		// Test with empty code
		emptyInput := Input{}
		if emptyInput.Code != "" {
			t.Error("Empty Input should have empty Code field")
		}
	})

	t.Run("Output struct", func(t *testing.T) {
		result := &Result{
			Stdout:   "test",
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

		if output.Result.Stdout != "test" {
			t.Errorf("Output.Result.Stdout = %q, want 'test'", output.Result.Stdout)
		}

		// Test with nil result
		outputNil := Output{}
		if outputNil.Result != nil {
			t.Error("Empty Output should have nil Result")
		}
	})
}

func TestExecute_JSONMarshaling(t *testing.T) {
	ctx := context.Background()

	t.Run("verify JSON structure", func(t *testing.T) {
		input := Input{
			Code: `print("test output")`,
		}

		req := &mcp.CallToolRequest{}

		result, err := Execute(ctx, req, input)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		textContent, ok := result.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatal("Execute() content should be TextContent")
		}

		// Verify it's valid JSON
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &data); err != nil {
			t.Fatalf("Execute() should return valid JSON: %v", err)
		}

		// Verify structure
		if _, ok := data["result"]; !ok {
			t.Error("JSON should have 'result' key")
		}

		resultData, ok := data["result"].(map[string]interface{})
		if !ok {
			t.Fatal("result should be an object")
		}

		// Check for expected fields (error is omitempty, so may not be present)
		requiredFields := []string{"stdout", "stderr", "duration"}
		for _, field := range requiredFields {
			if _, ok := resultData[field]; !ok {
				t.Errorf("result should have '%s' field", field)
			}
		}

		// error field is optional (omitempty)
		if _, ok := resultData["error"]; ok {
			t.Logf("result has 'error' field (optional)")
		}
	})

	t.Run("JSON with error", func(t *testing.T) {
		input := Input{
			Code: `invalid syntax`,
		}

		req := &mcp.CallToolRequest{}

		result, err := Execute(ctx, req, input)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		textContent, ok := result.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatal("Execute() content should be TextContent")
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &data); err != nil {
			t.Fatalf("Execute() should return valid JSON even for errors: %v", err)
		}

		resultData, ok := data["result"].(map[string]interface{})
		if !ok {
			t.Fatal("result should be an object")
		}

		// Should have error information
		if errorMsg, ok := resultData["error"].(string); ok && errorMsg == "" {
			if stderr, ok := resultData["stderr"].(string); !ok || stderr == "" {
				t.Error("result should have error or stderr populated for syntax errors")
			}
		}
	})
}
