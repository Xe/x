package python

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Input represents the input for the execute tool.
type Input struct {
	// Code is the Python code to execute.
	Code string `json:"code" jsonschema:"The Python code to execute in the wasm sandbox"`
}

// Output represents the output from the execute tool.
type Output struct {
	// Result contains the execution results.
	Result *Result `json:"result"`
}

// Execute runs Python code and returns the output formatted for MCP.
//
// This function is designed to be used as an MCP tool handler. It executes
// the Python code in the wasm sandbox and returns the results in a format
// that MCP clients can consume.
func Execute(ctx context.Context, req *mcp.CallToolRequest, input Input) (*mcp.CallToolResult, error) {
	result, err := Run(ctx, input.Code)
	if err != nil {
		// Even on error, we might have partial output
		if result != nil {
			result.Error = err.Error()
		} else {
			result = &Result{Error: err.Error()}
		}
	}

	// Serialize the result to JSON for the MCP response
	outputData, err := json.Marshal(map[string]interface{}{
		"result": result,
	})
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(outputData),
			},
		},
	}, nil
}

// ExecuteWithConfig runs Python code with custom configuration.
func ExecuteWithConfig(ctx context.Context, req *mcp.CallToolRequest, input Input, cfg Config) (*mcp.CallToolResult, error) {
	result, err := RunWithConfig(ctx, input.Code, cfg)
	if err != nil {
		if result != nil {
			result.Error = err.Error()
		} else {
			result = &Result{Error: err.Error()}
		}
	}

	outputData, err := json.Marshal(map[string]interface{}{
		"result": result,
	})
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(outputData),
			},
		},
	}, nil
}

// Tool returns the MCP tool definition for the Python interpreter.
func Tool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "python-interpreter",
		Description: "Execute Python code in a secure wasm sandbox. Use for calculations, data processing, algorithms, and any computation task. Returns stdout, stderr, execution time, and any errors.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"code": map[string]interface{}{
					"type":        "string",
					"description": "The Python code to execute in the wasm sandbox",
				},
			},
			"required": []string{"code"},
		},
	}
}

// Instruction returns a prompt instruction for the LLM about when to use
// the python-interpreter tool.
func Instruction() string {
	return `When you need to perform calculations, data processing, algorithms, or any computational task, use the python-interpreter tool.

The python-interpreter runs Python code in a secure WebAssembly sandbox with:
- No network access
- Limited filesystem access
- 30 second timeout (configurable)
- 128 MB memory limit (configurable)

Common use cases:
- Mathematical calculations
- Data manipulation and analysis
- String processing and parsing
- Algorithm implementation
- JSON/data transformations
- Statistical operations

The interpreter supports standard Python libraries. Results include stdout, stderr, execution duration, and any errors.

To use it, provide Python code as a string to the python-interpreter tool.`
}
