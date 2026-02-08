package python

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestExecute_CornerCases tests additional edge cases for the Execute function
func TestExecute_CornerCases(t *testing.T) {
	ctx := context.Background()

	t.Run("result with all fields populated", func(t *testing.T) {
		// Code that produces stdout, stderr, and an error
		input := Input{
			Code: `import sys
print("stdout output")
sys.stderr.write("stderr output\n")
# This will produce an error after some output
raise Exception("test error")`,
		}

		req := &mcp.CallToolRequest{}

		result, err := Execute(ctx, req, input)
		if err != nil {
			t.Errorf("Execute() should not return error: %v", err)
		}

		if result == nil {
			t.Fatal("Execute() result should not be nil")
		}

		// Should have content
		if len(result.Content) == 0 {
			t.Error("Execute() should return content")
		}
	})

	t.Run("multiple rapid executions", func(t *testing.T) {
		input := Input{
			Code: `print("test")`,
		}

		req := &mcp.CallToolRequest{}

		// Execute multiple times to verify stability
		for i := 0; i < 5; i++ {
			result, err := Execute(ctx, req, input)
			if err != nil {
				t.Errorf("Execute() iteration %d failed: %v", i, err)
			}

			if result == nil {
				t.Errorf("Execute() iteration %d returned nil result", i)
			}
		}
	})

	t.Run("code with only newlines", func(t *testing.T) {
		input := Input{
			Code: "\n\n\n",
		}

		req := &mcp.CallToolRequest{}

		result, err := Execute(ctx, req, input)
		if err != nil {
			t.Errorf("Execute() with newlines should not error: %v", err)
		}

		if result == nil {
			t.Fatal("Execute() result should not be nil")
		}
	})
}

// TestExecuteWithConfig_CornerCases tests additional edge cases for ExecuteWithConfig
func TestExecuteWithConfig_CornerCases(t *testing.T) {
	ctx := context.Background()

	t.Run("config with maximum values", func(t *testing.T) {
		input := Input{
			Code: `print("test")`,
		}

		req := &mcp.CallToolRequest{}

		cfg := Config{
			Timeout:     1 << 62, // Very large timeout
			MemoryLimit: 1 << 30, // 1 GB
			Stdin:       "",
		}

		result, err := ExecuteWithConfig(ctx, req, input, cfg)
		if err != nil {
			t.Errorf("ExecuteWithConfig() with large config should work: %v", err)
		}

		if result == nil {
			t.Fatal("ExecuteWithConfig() result should not be nil")
		}
	})

	t.Run("stdin with special characters", func(t *testing.T) {
		input := Input{
			Code: `print("test")`,
		}

		req := &mcp.CallToolRequest{}

		cfg := Config{
			Stdin: "\t\n\r\"'\\",
		}

		result, err := ExecuteWithConfig(ctx, req, input, cfg)
		if err != nil {
			t.Errorf("ExecuteWithConfig() with special stdin chars should work: %v", err)
		}

		if result == nil {
			t.Fatal("ExecuteWithConfig() result should not be nil")
		}
	})

	t.Run("config with zero timeout", func(t *testing.T) {
		input := Input{
			Code: `print("no timeout")`,
		}

		req := &mcp.CallToolRequest{}

		cfg := Config{
			Timeout: 0,
		}

		result, err := ExecuteWithConfig(ctx, req, input, cfg)
		if err != nil {
			t.Errorf("ExecuteWithConfig() with zero timeout should work: %v", err)
		}

		if result == nil {
			t.Fatal("ExecuteWithConfig() result should not be nil")
		}
	})
}

// TestResult_AllFields tests the Result struct with all possible field states
func TestResult_AllFields(t *testing.T) {
	t.Run("result with all empty fields", func(t *testing.T) {
		result := &Result{
			Stdout:   "",
			Stderr:   "",
			Error:    "",
			Duration: 0,
		}

		// Verify we can access all fields without panicking
		_ = result.Stdout
		_ = result.Stderr
		_ = result.Error
		_ = result.Duration
	})

	t.Run("result with only stdout", func(t *testing.T) {
		result := &Result{
			Stdout:   "output",
			Stderr:   "",
			Error:    "",
			Duration: 1000000,
		}

		if result.Stdout != "output" {
			t.Errorf("Result.Stdout = %q, want 'output'", result.Stdout)
		}
	})

	t.Run("result with only stderr", func(t *testing.T) {
		result := &Result{
			Stdout:   "",
			Stderr:   "error output",
			Error:    "",
			Duration: 1000000,
		}

		if result.Stderr != "error output" {
			t.Errorf("Result.Stderr = %q, want 'error output'", result.Stderr)
		}
	})

	t.Run("result with only error", func(t *testing.T) {
		result := &Result{
			Stdout:   "",
			Stderr:   "",
			Error:    "execution failed",
			Duration: 1000000,
		}

		if result.Error != "execution failed" {
			t.Errorf("Result.Error = %q, want 'execution failed'", result.Error)
		}
	})
}

// TestConfig_AllValues tests the Config struct with various value combinations
func TestConfig_AllValues(t *testing.T) {
	t.Run("config with all fields set", func(t *testing.T) {
		cfg := Config{
			Timeout:     1000000000,
			MemoryLimit: 1024 * 1024,
			Stdin:       "test input",
		}

		if cfg.Timeout != 1000000000 {
			t.Errorf("Config.Timeout = %v, want 1000000000", cfg.Timeout)
		}

		if cfg.MemoryLimit != 1024*1024 {
			t.Errorf("Config.MemoryLimit = %d, want %d", cfg.MemoryLimit, 1024*1024)
		}

		if cfg.Stdin != "test input" {
			t.Errorf("Config.Stdin = %q, want 'test input'", cfg.Stdin)
		}
	})

	t.Run("config with only timeout", func(t *testing.T) {
		cfg := Config{
			Timeout: 500000000,
		}

		if cfg.Timeout != 500000000 {
			t.Errorf("Config.Timeout = %v, want 500000000", cfg.Timeout)
		}

		if cfg.MemoryLimit != 0 {
			t.Errorf("Config.MemoryLimit = %d, want 0", cfg.MemoryLimit)
		}

		if cfg.Stdin != "" {
			t.Errorf("Config.Stdin = %q, want empty", cfg.Stdin)
		}
	})

	t.Run("config with only memory limit", func(t *testing.T) {
		cfg := Config{
			MemoryLimit: 2048 * 1024,
		}

		if cfg.Timeout != 0 {
			t.Errorf("Config.Timeout = %v, want 0", cfg.Timeout)
		}

		if cfg.MemoryLimit != 2048*1024 {
			t.Errorf("Config.MemoryLimit = %d, want %d", cfg.MemoryLimit, 2048*1024)
		}
	})
}
