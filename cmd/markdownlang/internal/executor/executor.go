// Package executor implements the main execution logic for markdownlang programs.
package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/packages/param"
	"github.com/openai/openai-go/v2/shared"
	"within.website/x/cmd/markdownlang/internal/agent"
	"within.website/x/cmd/markdownlang/internal/config"
	"within.website/x/cmd/markdownlang/internal/mcp"
	"within.website/x/cmd/markdownlang/internal/parser"
	"within.website/x/cmd/markdownlang/internal/template"
	"within.website/x/llm/codeinterpreter/python"
)

// ExecutionMetrics tracks metrics during program execution.
type ExecutionMetrics struct {
	Iterations   int
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	ToolsCalled  int
}

// Executor handles program execution.
type Executor struct {
	client       openai.Client
	program      *parser.Program
	input        map[string]interface{}
	registry     *agent.Registry
	callManager  *agent.CallManager
	mcpManager   *mcp.Manager                 // MCP server manager
	toolHandlers map[string]agent.ToolHandler // tool name -> handler
	startTime    time.Time
	metrics      ExecutionMetrics
}

// New creates a new executor using the global config flags.
func New() (*Executor, error) {
	// Create OpenAI client
	client := openai.NewClient(
		option.WithAPIKey(*config.APIKey),
		option.WithBaseURL(*config.BaseURL),
	)

	// Parse input JSON
	var input map[string]interface{}
	if err := json.Unmarshal([]byte(*config.Input), &input); err != nil {
		return nil, fmt.Errorf("failed to parse input JSON: %w", err)
	}

	// Load and parse program
	program, err := parser.LoadProgram(*config.ProgramPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load program: %w", err)
	}

	// Validate input against schema
	if err := program.ValidateInput(input); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Create agent registry for imports
	programDir := filepath.Dir(*config.ProgramPath)
	registry := agent.NewRegistry(&agent.RegistryConfig{
		APIKey:  *config.APIKey,
		BaseURL: *config.BaseURL,
		Model:   *config.Model,
		BaseDir: programDir,
	})

	// Create call manager for agent-to-agent calls
	callManager := agent.NewCallManager(&agent.CallManagerConfig{
		Registry:      registry,
		EnableTracing: *config.Debug,
	})

	// Create MCP manager for MCP servers
	mcpManager := mcp.NewManager(nil)

	return &Executor{
		client:      client,
		program:     program,
		input:       input,
		registry:    registry,
		callManager: callManager,
		mcpManager:  mcpManager,
	}, nil
}

// Execute runs the program and returns the output.
func (e *Executor) Execute(ctx context.Context) (map[string]interface{}, error) {
	e.startTime = time.Now()

	// Initialize tool handlers with default tools
	e.toolHandlers = make(map[string]agent.ToolHandler)

	// Add Python interpreter as a default tool
	pythonSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"code": {
				"type": "string",
				"description": "Python code to execute"
			}
		},
		"required": ["code"]
	}`)

	e.toolHandlers["python"] = agent.NewStaticSchemaTool(pythonSchema, func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
		// Parse input to get the code
		var inputStruct struct {
			Code string `json:"code"`
		}
		if err := json.Unmarshal(input, &inputStruct); err != nil {
			return json.RawMessage(fmt.Sprintf(`{"error": "invalid input: %v"}`, err)), nil
		}

		if *config.Debug {
			fmt.Fprintf(os.Stderr, "Executing Python code:\n%s\n", inputStruct.Code)
		}

		// Run the Python code
		result, err := python.Run(ctx, nil, inputStruct.Code)
		if err != nil {
			return json.RawMessage(fmt.Sprintf(`{"error": "execution failed: %v", "stderr": %q}`, err, result.Stderr)), nil
		}

		// Return the result
		output := fmt.Sprintf(`{"stdout": %q, "stderr": %q}`, result.Stdout, result.Stderr)
		return json.RawMessage(output), nil
	})

	if *config.Debug {
		fmt.Fprintf(os.Stderr, "Added default tool: python\n")
	}

	// Start MCP servers if any
	if len(e.program.MCPServers) > 0 {
		if *config.Debug {
			fmt.Fprintf(os.Stderr, "Starting %d MCP servers...\n", len(e.program.MCPServers))
		}

		for i := range e.program.MCPServers {
			serverConfig := &e.program.MCPServers[i]
			if err := e.mcpManager.Start(ctx, serverConfig); err != nil {
				slog.Warn("failed to start MCP server", "server", serverConfig.Name, "error", err)
				continue
			}

			if *config.Debug {
				fmt.Fprintf(os.Stderr, "  Started MCP server: %s\n", serverConfig.Name)
			}
		}

		// Get tools from all MCP servers
		mcpTools, err := e.mcpManager.GetTools(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get MCP tools: %w", err)
		}

		if *config.Debug {
			fmt.Fprintf(os.Stderr, "Loaded %d MCP tools\n", len(mcpTools))
		}

		// Add MCP tools as tool handlers
		for _, tool := range mcpTools {
			toolName := tool.Tool.Name

			// Build JSON schema from the tool's input schema
			var schemaBytes json.RawMessage
			if tool.Tool.InputSchema != nil {
				if inputSchema, ok := tool.Tool.InputSchema.(map[string]any); ok {
					schemaBytes, _ = json.Marshal(inputSchema)
				}
			} else {
				// Default to empty object schema
				schemaBytes = json.RawMessage(`{"type": "object"}`)
			}

			// Create tool handler that calls the MCP manager
			e.toolHandlers[toolName] = agent.NewStaticSchemaTool(schemaBytes, func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
				// Parse input arguments
				var args map[string]any
				if err := json.Unmarshal(input, &args); err != nil {
					return json.RawMessage(fmt.Sprintf(`{"error": "invalid arguments: %v"}`, err)), nil
				}

				if *config.Debug {
					fmt.Fprintf(os.Stderr, "Calling MCP tool: %s\n", toolName)
				}

				// Call the MCP tool
				result, err := e.mcpManager.CallTool(ctx, toolName, args)
				if err != nil {
					slog.Error("MCP tool call failed", "tool", toolName, "error", err)
					return json.RawMessage(fmt.Sprintf(`{"error": "tool call failed: %v"}`, err)), nil
				}

				// Convert MCP result to JSON
				toolResult := mcp.NewToolCallResult(toolName, tool.Server, result)

				// Marshal the result back to JSON
				resultBytes, err := json.Marshal(toolResult)
				if err != nil {
					return json.RawMessage(fmt.Sprintf(`{"error": "failed to marshal result: %v"}`, err)), nil
				}

				return json.RawMessage(resultBytes), nil
			})

			if *config.Debug {
				fmt.Fprintf(os.Stderr, "  Added MCP tool: %s (from %s)\n", toolName, tool.Server)
			}
		}
	}

	// Load imports if any
	if len(e.program.Imports) > 0 {
		if *config.Debug {
			fmt.Fprintf(os.Stderr, "Loading %d imports...\n", len(e.program.Imports))
		}

		importHandlers, err := e.registry.CreateToolHandlers(ctx, e.program)
		if err != nil {
			return nil, fmt.Errorf("failed to load imports: %w", err)
		}

		// Merge import handlers into tool handlers
		for name, handler := range importHandlers {
			e.toolHandlers[name] = handler
		}

		if *config.Debug {
			fmt.Fprintf(os.Stderr, "Loaded %d agent tools\n", len(importHandlers))
			for name := range importHandlers {
				fmt.Fprintf(os.Stderr, "  - %s\n", name)
			}
		}
	}

	// Render template with input
	renderer := template.New()
	rendered, err := renderer.Render(e.program.Content, e.input)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	if *config.Debug {
		fmt.Fprintf(os.Stderr, "Rendered content:\n%s\n", rendered)
	}

	// Build system message
	systemMsg := fmt.Sprintf("You are a helpful AI assistant. Process the following request and output valid JSON matching the expected schema.\n\nOutput schema:\n%s", e.program.OutputSchema)

	// Execute agent loop
	result, err := e.executeAgentLoop(ctx, systemMsg, rendered)
	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	// Validate output against schema
	if err := e.program.ValidateOutput(result); err != nil {
		return nil, fmt.Errorf("output validation failed: %w", err)
	}

	// Stop all MCP servers
	if e.mcpManager.ServerCount() > 0 {
		if err := e.mcpManager.StopAll(ctx); err != nil {
			slog.Warn("failed to stop MCP servers", "error", err)
		}
	}

	return result, nil
}

// buildToolDefinitions converts tool handlers to OpenAI tool definitions.
func (e *Executor) buildToolDefinitions() []openai.ChatCompletionToolUnionParam {
	if len(e.toolHandlers) == 0 {
		return nil
	}

	tools := make([]openai.ChatCompletionToolUnionParam, 0, len(e.toolHandlers))

	for toolName, handler := range e.toolHandlers {
		// Get the schema from the handler
		schema := handler.Schema()

		// Parse the schema to get description
		var schemaMap map[string]interface{}
		if err := json.Unmarshal(schema, &schemaMap); err != nil {
			slog.Error("failed to parse tool schema", "tool", toolName, "error", err)
			continue
		}

		// Extract description if present
		description := ""
		if desc, ok := schemaMap["description"].(string); ok {
			description = desc
		}

		// Create function definition
		funcDef := shared.FunctionDefinitionParam{
			Name:       toolName,
			Parameters: shared.FunctionParameters(schemaMap),
		}

		// Set description if provided
		if description != "" {
			funcDef.Description = param.NewOpt(description)
		}

		// Create tool definition
		tool := openai.ChatCompletionFunctionTool(funcDef)
		tools = append(tools, tool)
	}

	return tools
}

// executeAgentLoop runs the main agent interaction loop with the LLM.
func (e *Executor) executeAgentLoop(ctx context.Context, systemMsg, userMsg string) (map[string]interface{}, error) {
	const maxIterations = 10

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemMsg),
		openai.UserMessage(userMsg),
	}

	for i := 0; i < maxIterations; i++ {
		e.metrics.Iterations = i + 1

		if *config.Debug {
			fmt.Fprintf(os.Stderr, "Iteration %d/%d\n", i+1, maxIterations)
		}

		// Create chat completion request
		responseFormat := shared.NewResponseFormatJSONObjectParam()

		// Use program's model if specified, otherwise use config flag
		model := *config.Model
		if e.program.Model != "" {
			model = e.program.Model
			if *config.Debug {
				fmt.Fprintf(os.Stderr, "Using program-specific model: %s\n", model)
			}
		}

		// Build tool definitions if we have tool handlers
		tools := e.buildToolDefinitions()

		params := openai.ChatCompletionNewParams{
			Messages: messages,
			Model:    openai.ChatModel(model),
			ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
				OfJSONObject: &responseFormat,
			},
		}

		// Add tools if available
		if len(tools) > 0 {
			params.Tools = tools
		}

		// Debug: Print the messages being sent to the AI
		if *config.Debug {
			fmt.Fprintf(os.Stderr, "\n--- Messages sent to AI ---\n")
			for i, msg := range messages {
				fmt.Fprintf(os.Stderr, "Message %d:\n", i)
				if msg.OfSystem != nil {
					content := "<array of content parts>"
					if msg.OfSystem.Content.OfString.Valid() {
						content = msg.OfSystem.Content.OfString.Value
					}
					fmt.Fprintf(os.Stderr, "  Role: system\n  Content: %s\n", content)
				} else if msg.OfUser != nil {
					content := "<array of content parts>"
					if msg.OfUser.Content.OfString.Valid() {
						content = msg.OfUser.Content.OfString.Value
					}
					fmt.Fprintf(os.Stderr, "  Role: user\n  Content: %s\n", content)
				} else if msg.OfAssistant != nil {
					content := "<array of content parts>"
					if msg.OfAssistant.Content.OfString.Valid() {
						content = msg.OfAssistant.Content.OfString.Value
					}
					fmt.Fprintf(os.Stderr, "  Role: assistant\n  Content: %s\n", content)
				}
			}
			fmt.Fprintf(os.Stderr, "--- End messages ---\n\n")
		}

		// Execute request
		completion, err := e.client.Chat.Completions.New(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("LLM request failed: %w", err)
		}

		// Track token usage
		e.metrics.InputTokens += int(completion.Usage.PromptTokens)
		e.metrics.OutputTokens += int(completion.Usage.CompletionTokens)
		e.metrics.TotalTokens += int(completion.Usage.TotalTokens)

		choice := completion.Choices[0]
		msg := choice.Message

		// Check if the LLM made tool calls
		if len(msg.ToolCalls) > 0 {
			if *config.Debug {
				fmt.Fprintf(os.Stderr, "LLM made %d tool calls\n", len(msg.ToolCalls))
			}

			// Build tool call params for the assistant message
			toolCallParams := make([]openai.ChatCompletionMessageToolCallUnionParam, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				toolCallParams[i] = tc.ToParam()
			}

			// Build content union for the assistant message
			contentUnion := openai.ChatCompletionAssistantMessageParamContentUnion{}
			if msg.Content != "" {
				contentUnion.OfString = param.NewOpt(msg.Content)
			}

			// Add the assistant message with tool calls to the conversation
			messages = append(messages, openai.ChatCompletionMessageParamUnion{
				OfAssistant: &openai.ChatCompletionAssistantMessageParam{
					Content:   contentUnion,
					ToolCalls: toolCallParams,
				},
			})

			// Execute each tool call
			for _, toolCall := range msg.ToolCalls {
				e.metrics.ToolsCalled++

				// Check if it's a function call
				if toolCall.Type != "function" {
					slog.Error("tool call is not a function call", "type", toolCall.Type)
					continue
				}

				toolName := toolCall.Function.Name
				arguments := toolCall.Function.Arguments

				if *config.Debug {
					fmt.Fprintf(os.Stderr, "Executing tool: %s\n", toolName)
					fmt.Fprintf(os.Stderr, "Arguments: %s\n", arguments)
				}

				// Execute the tool
				var result json.RawMessage
				var err error

				if handler, ok := e.toolHandlers[toolName]; ok {
					result, err = handler.Execute(ctx, json.RawMessage(arguments))
					if err != nil {
						slog.Error("tool execution failed", "tool", toolName, "error", err)
						result = json.RawMessage(fmt.Sprintf(`{"error": "%s"}`, err.Error()))
					}
				} else {
					slog.Error("tool not found", "tool", toolName)
					result = json.RawMessage(fmt.Sprintf(`{"error": "tool not found: %s"}`, toolName))
				}

				if *config.Debug {
					fmt.Fprintf(os.Stderr, "Tool result: %s\n", string(result))
				}

				// Build content union for the tool message
				resultContent := openai.ChatCompletionToolMessageParamContentUnion{
					OfString: param.NewOpt(string(result)),
				}

				// Add the tool result message to the conversation
				messages = append(messages, openai.ChatCompletionMessageParamUnion{
					OfTool: &openai.ChatCompletionToolMessageParam{
						Content:    resultContent,
						ToolCallID: toolCall.ID,
					},
				})
			}

			// Continue the loop to get the next response
			continue
		}

		// No tool calls, extract response content
		content := msg.Content
		if *config.Debug {
			fmt.Fprintf(os.Stderr, "LLM response:\n%s\n", content)
		}

		// Try to parse as JSON
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(content), &result); err != nil {
			// Not valid JSON, add feedback and retry
			messages = append(messages, openai.AssistantMessage(content))
			messages = append(messages, openai.UserMessage(fmt.Sprintf("Invalid JSON. Please output valid JSON matching the schema. Error: %v", err)))
			continue
		}

		// Validate against output schema
		if err := e.program.ValidateOutput(result); err != nil {
			messages = append(messages, openai.AssistantMessage(content))
			messages = append(messages, openai.UserMessage(fmt.Sprintf("Output doesn't match schema. Please fix. Error: %v", err)))
			continue
		}

		// Success! Return the result
		return result, nil
	}

	return nil, fmt.Errorf("exceeded maximum iterations (%d) without valid output", maxIterations)
}

// GetMetrics returns the current execution metrics.
func (e *Executor) GetMetrics() map[string]interface{} {
	toolsCalled := e.metrics.ToolsCalled

	// Add agent calls to the tool count if available
	if e.callManager != nil {
		callMetrics := e.callManager.GetCallMetrics()
		toolsCalled += callMetrics.TotalCalls
	}

	metrics := map[string]interface{}{
		"start_time":    e.startTime,
		"end_time":      time.Now(),
		"duration":      time.Since(e.startTime),
		"iterations":    e.metrics.Iterations,
		"input_tokens":  e.metrics.InputTokens,
		"output_tokens": e.metrics.OutputTokens,
		"total_tokens":  e.metrics.TotalTokens,
		"tools_called":  toolsCalled,
	}

	// Add call manager metrics if available
	if e.callManager != nil {
		callMetrics := e.callManager.GetCallMetrics()
		metrics["agent_calls"] = map[string]interface{}{
			"total_calls":             callMetrics.TotalCalls,
			"active_calls":            callMetrics.ActiveCalls,
			"calls_by_import":         callMetrics.CallsByImport,
			"calls_by_agent":          callMetrics.CallsByAgent,
			"total_duration":          callMetrics.TotalDuration,
			"average_duration":        callMetrics.AverageDuration,
			"total_tokens":            callMetrics.TotalTokens,
			"average_tokens_per_call": callMetrics.AverageTokensPerCall,
		}
	}

	return metrics
}

// OutputResult writes the result to the output file or stdout.
func OutputResult(result map[string]interface{}, outputPath string) error {
	// Marshal result with pretty printing
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	// Write to file or stdout
	if outputPath != "" {
		if err := os.WriteFile(outputPath, output, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
	} else {
		fmt.Println(string(output))
	}

	return nil
}
