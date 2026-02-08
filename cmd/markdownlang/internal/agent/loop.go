// Package agent implements the core agent loop for markdownlang programs.
//
// The agent loop is the heart of markdownlang - when an LLM calls a function,
// it blocks until that function's agent loop completes. This allows for
// composable, hierarchical agent systems where agents can call other agents.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"within.website/x/cmd/markdownlang/internal/parser"
)

const (
	// MaxIterations is the maximum number of iterations before giving up.
	MaxIterations = 10

	// DefaultModel is the default OpenAI model to use.
	DefaultModel = "gpt-5-pro"

	// DefaultTemperature is the default temperature for responses.
	DefaultTemperature = 0.7
)

// AgentLoop defines the interface for executing markdownlang programs.
//
// Execute runs the agent loop with the given input context and data,
// returning structured output that conforms to the program's output schema.
type AgentLoop interface {
	// Execute runs the agent loop with the provided input data.
	// The returned JSON will conform to the program's output schema.
	Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error)

	// Metrics returns runtime metrics from the last or current execution.
	Metrics() Metrics
}

// Program is a loaded markdownlang program ready for execution.
type Program struct {
	// parser.Program contains the parsed program metadata.
	parser.Program

	// client is the OpenAI client for making API calls.
	client openai.Client

	// apiKey is the OpenAI API key.
	apiKey string

	// baseURL is the OpenAI base URL.
	baseURL string

	// model is the OpenAI model to use for responses.
	model string

	// temperature is the temperature for responses.
	temperature float64

	// metrics tracks runtime metrics.
	metrics Metrics
}

// NewProgram creates a new Program from a parsed parser.Program.
func NewProgram(prog *parser.Program, opts ...ProgramOption) (*Program, error) {
	if prog == nil {
		return nil, errors.New("program is nil: nothing to execute")
	}

	p := &Program{
		Program:     *prog,
		client:      openai.NewClient(),
		model:       DefaultModel,
		temperature: DefaultTemperature,
		metrics:     Metrics{},
	}

	for _, opt := range opts {
		opt(p)
	}

	return p, nil
}

// ProgramOption configures a Program.
type ProgramOption func(*Program)

// WithClient sets a custom OpenAI client.
func WithClient(client openai.Client) ProgramOption {
	return func(p *Program) {
		p.client = client
	}
}

// WithModel sets the OpenAI model to use.
func WithModel(model string) ProgramOption {
	return func(p *Program) {
		p.model = model
	}
}

// WithTemperature sets the temperature for responses.
func WithTemperature(temp float64) ProgramOption {
	return func(p *Program) {
		p.temperature = temp
	}
}

// WithAPIKey sets the OpenAI API key.
func WithAPIKey(apiKey string) ProgramOption {
	return func(p *Program) {
		p.apiKey = apiKey
		p.createClient()
	}
}

// WithBaseURL sets the OpenAI base URL.
func WithBaseURL(baseURL string) ProgramOption {
	return func(p *Program) {
		p.baseURL = baseURL
		p.createClient()
	}
}

// createClient creates or recreates the OpenAI client with current settings.
func (p *Program) createClient() {
	opts := []option.RequestOption{}
	if p.apiKey != "" {
		opts = append(opts, option.WithAPIKey(p.apiKey))
	}
	if p.baseURL != "" {
		opts = append(opts, option.WithBaseURL(p.baseURL))
	}
	if len(opts) > 0 {
		p.client = openai.NewClient(opts...)
	}
}

// Execute runs the agent loop with the provided input data.
func (p *Program) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	startTime := time.Now()
	p.metrics = Metrics{} // Reset metrics

	// Validate input against schema
	if err := p.validateInput(input); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	slog.Info("starting agent loop",
		"program", p.Name,
		"model", p.model,
		"temperature", p.temperature)

	var lastError error
	var result json.RawMessage

	// Iterate until we get valid output or hit max iterations
	for iteration := 0; iteration < MaxIterations; iteration++ {
		p.metrics.Iterations++

		slog.Debug("agent loop iteration",
			"program", p.Name,
			"iteration", iteration+1,
			"max_iterations", MaxIterations)

		// Build the request
		req := p.buildRequest(ctx, input, iteration, lastError)

		// Call the OpenAI Responses API
		resp, err := p.client.Responses.New(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("OpenAI API call failed: %w", err)
		}

		// Update metrics
		p.updateMetrics(resp)

		// Extract and validate the output
		result, lastError = p.extractAndValidateOutput(resp)
		if lastError == nil {
			// Success!
			p.metrics.Duration = time.Since(startTime)
			slog.Info("agent loop completed",
				"program", p.Name,
				"iterations", p.metrics.Iterations,
				"duration", p.metrics.Duration,
				"total_tokens", p.metrics.TotalTokens)
			return result, nil
		}

		slog.Debug("output validation failed, will retry",
			"program", p.Name,
			"error", lastError,
			"iteration", iteration+1)
	}

	p.metrics.Duration = time.Since(startTime)
	return nil, fmt.Errorf("agent loop failed after %d iterations: %w", MaxIterations, lastError)
}

// Metrics returns the runtime metrics from the last execution.
func (p *Program) Metrics() Metrics {
	return p.metrics
}

// buildRequest constructs the ResponseNewParams for an iteration.
func (p *Program) buildRequest(ctx context.Context, input json.RawMessage, iteration int, lastError error) responses.ResponseNewParams {
	// Build the system message with description and interpolated input
	systemMsg := p.buildSystemMessage(input, iteration, lastError)

	// Input can be either a string or structured - use string representation
	inputStr := string(input)

	params := responses.ResponseNewParams{
		Model:        p.model,
		Temperature:  param.NewOpt(p.temperature),
		Instructions: param.NewOpt(systemMsg),
		Input:        responses.ResponseNewParamsInputUnion{OfString: param.NewOpt(inputStr)},
	}

	return params
}

// buildSystemMessage creates the system message with interpolated input.
func (p *Program) buildSystemMessage(input json.RawMessage, iteration int, lastError error) string {
	msg := fmt.Sprintf("# %s\n\n", p.Name)

	if p.Description != "" {
		msg += fmt.Sprintf("%s\n\n", p.Description)
	}

	// Add input data if available
	if len(input) > 0 {
		msg += "## Input\n\n"
		msg += "```json\n"
		msg += string(input)
		msg += "\n```\n\n"
	}

	// Add error feedback from previous iteration
	if iteration > 0 && lastError != nil {
		msg += fmt.Sprintf("## Previous Error\n\n")
		msg += fmt.Sprintf("Your previous output did not match the required schema: %s\n\n", lastError.Error())
		msg += "Please try again, ensuring your output conforms to the output schema.\n\n"
	}

	msg += "## Instructions\n\n"
	msg += "You must respond with valid JSON that matches the output schema exactly. "
	msg += "Do not include any text outside the JSON structure.\n"

	return msg
}

// validateInput validates input data against the input schema.
func (p *Program) validateInput(input json.RawMessage) error {
	// Empty input is allowed - many programs don't require input
	// Basic JSON validation
	var data interface{}
	if err := json.Unmarshal(input, &data); err != nil {
		return fmt.Errorf("input is not valid JSON: %w", err)
	}

	// TODO: Full JSON Schema validation against p.InputSchema
	// For now, we just check that it's valid JSON

	return nil
}

// extractAndValidateOutput extracts the output from a Response and validates it.
func (p *Program) extractAndValidateOutput(resp *responses.Response) (json.RawMessage, error) {
	if len(resp.Output) == 0 {
		return nil, errors.New("response has no output: the model gave us nothing")
	}

	// Get the last output item (should be the final response)
	lastOutput := resp.Output[len(resp.Output)-1]

	// Extract the content based on the output type
	var result json.RawMessage
	switch lastOutput.Type {
	case "message":
		if len(lastOutput.Content) == 0 {
			return nil, errors.New("message has no content")
		}
		// Get the first content item
		content := lastOutput.Content[0]
		// Check content type
		switch content.Type {
		case "output_text":
			result = json.RawMessage(content.Text)
		case "refusal":
			return nil, errors.New("model refused to generate output: " + content.Refusal)
		default:
			return nil, fmt.Errorf("unsupported content type: %s", content.Type)
		}
	default:
		return nil, fmt.Errorf("unsupported output type: %s", lastOutput.Type)
	}

	// Validate the output against the schema
	if err := p.validateOutput(result); err != nil {
		return nil, err
	}

	return result, nil
}

// validateOutput validates output data against the output schema.
func (p *Program) validateOutput(output json.RawMessage) error {
	if len(output) == 0 {
		return errors.New("output is empty: that's not valid JSON")
	}

	// Basic JSON validation
	var data interface{}
	if err := json.Unmarshal(output, &data); err != nil {
		return fmt.Errorf("output is not valid JSON: %w", err)
	}

	// TODO: Full JSON Schema validation against p.OutputSchema
	// For now, we just check that it's valid JSON

	return nil
}

// updateMetrics updates metrics from the OpenAI response.
func (p *Program) updateMetrics(resp *responses.Response) {
	// Count tokens - ResponseUsage is a struct, not a pointer
	p.metrics.TotalTokens += int(resp.Usage.TotalTokens)
	p.metrics.InputTokens += int(resp.Usage.InputTokens)
	p.metrics.OutputTokens += int(resp.Usage.OutputTokens)

	// Count tool calls
	for _, item := range resp.Output {
		// Check if this is a function call
		if item.Type == "function_call" || item.Type == "mcp_call" {
			p.metrics.ToolsCalled++
		}
	}
}
