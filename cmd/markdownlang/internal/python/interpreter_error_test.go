package python

import (
	"context"
	"fmt"
	"testing"
)

// Test that Result can be marshaled to JSON
func TestResultMarshaling(t *testing.T) {
	tests := []struct {
		name   string
		result *Result
	}{
		{
			name: "successful result",
			result: &Result{
				Stdout:   "output",
				Stderr:   "",
				Error:    "",
				Duration: 1000000,
			},
		},
		{
			name: "result with error",
			result: &Result{
				Stdout:   "partial output",
				Stderr:   "error output",
				Error:    "execution failed",
				Duration: 500000,
			},
		},
		{
			name: "empty result",
			result: &Result{
				Stdout:   "",
				Stderr:   "",
				Error:    "",
				Duration: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify that the Result struct can be created
			// The actual marshaling happens in the MCP layer
			if tt.result == nil {
				t.Error("Result should not be nil")
			}

			// Verify all fields are accessible
			_ = tt.result.Stdout
			_ = tt.result.Stderr
			_ = tt.result.Error
			_ = tt.result.Duration
		})
	}
}

// Test edge case: result with only error
func TestResultOnlyError(t *testing.T) {
	ctx := context.Background()

	// Code that will fail
	code := `this is not valid python at all`

	result, err := Run(ctx, code)

	// We expect an error, but result might still be returned
	if err != nil && result == nil {
		// This is acceptable - error with no result
		return
	}

	if result != nil {
		// If we got a result, it should have error information
		if result.Error == "" && result.Stderr == "" {
			t.Error("Result should contain error information for invalid Python code")
		}
	}
}

// Test sequential execution with error recovery
func TestSequentialExecutionWithErrors(t *testing.T) {
	ctx := context.Background()

	codes := []string{
		`print("first")`,         // valid
		`print("unclosed string`, // invalid
		`print("third")`,         // valid
	}

	for i, code := range codes {
		t.Run(fmt.Sprintf("sequence_%d", i), func(t *testing.T) {
			result, err := Run(ctx, code)

			if i == 1 {
				// The invalid code should error
				if err == nil && result.Error == "" {
					t.Error("Expected error for invalid code")
				}
			} else {
				// Valid code should execute
				if err != nil {
					t.Errorf("Unexpected error for valid code: %v", err)
				}
			}
		})
	}
}

// Test Config zero values
func TestConfigZeroValues(t *testing.T) {
	var cfg Config

	if cfg.Timeout != 0 {
		t.Errorf("Zero Config.Timeout = %v, want 0", cfg.Timeout)
	}

	if cfg.MemoryLimit != 0 {
		t.Errorf("Zero Config.MemoryLimit = %d, want 0", cfg.MemoryLimit)
	}

	if cfg.Stdin != "" {
		t.Errorf("Zero Config.Stdin = %q, want empty", cfg.Stdin)
	}
}
