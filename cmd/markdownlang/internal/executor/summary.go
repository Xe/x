// Package executor implements execution summary reporting for markdownlang.
package executor

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ExecutionSummary contains metrics and information about program execution.
type ExecutionSummary struct {
	// Program is the name of the program that was executed.
	Program string `json:"program"`

	// Success indicates whether execution completed successfully.
	Success bool `json:"success"`

	// Iterations is the number of iterations the agent loop performed.
	Iterations int `json:"iterations"`

	// Tokens contains token usage statistics.
	Tokens TokenSummary `json:"tokens"`

	// ToolsCalled is the number of tool calls made.
	ToolsCalled int `json:"tools_called"`

	// Duration is the total time taken for execution.
	Duration time.Duration `json:"duration"`

	// Error contains the error message if execution failed.
	Error string `json:"error,omitempty"`

	// AgentCalls contains information about agent-to-agent calls.
	AgentCalls *AgentCallsSummary `json:"agent_calls,omitempty"`

	// Model is the model that was used for execution.
	Model string `json:"model"`

	// StartTime is when execution started.
	StartTime time.Time `json:"start_time"`

	// EndTime is when execution ended.
	EndTime time.Time `json:"end_time"`
}

// TokenSummary contains token usage statistics.
type TokenSummary struct {
	// Total is the total number of tokens used.
	Total int `json:"total"`

	// Input is the number of tokens in the input.
	Input int `json:"input"`

	// Output is the number of tokens in the output.
	Output int `json:"output"`

	// Cost is the estimated cost in USD.
	Cost float64 `json:"cost,omitempty"`
}

// AgentCallsSummary contains information about agent-to-agent calls.
type AgentCallsSummary struct {
	// TotalCalls is the total number of agent calls made.
	TotalCalls int `json:"total_calls"`

	// CallsByAgent maps agent names to the number of times they were called.
	CallsByAgent map[string]int `json:"calls_by_agent"`

	// TotalDuration is the total time spent in agent calls.
	TotalDuration time.Duration `json:"total_duration"`

	// AverageDuration is the average duration of an agent call.
	AverageDuration time.Duration `json:"average_duration"`

	// TokensUsed is the total number of tokens used in agent calls.
	TokensUsed int `json:"tokens_used"`
}

// NewExecutionSummary creates an execution summary from metrics.
func NewExecutionSummary(programName string, metrics interface{}, success bool, errMsg string, model string, startTime, endTime time.Time) *ExecutionSummary {
	summary := &ExecutionSummary{
		Program:   programName,
		Success:   success,
		Model:     model,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
	}

	if errMsg != "" {
		summary.Error = errMsg
	}

	// Try to extract metrics from different types
	switch m := metrics.(type) {
	case map[string]interface{}:
		summary.extractFromMap(m)
	case interface{ Iterations() int }:
		summary.Iterations = m.Iterations()
	}

	return summary
}

// extractFromMap extracts metrics from a generic map.
func (s *ExecutionSummary) extractFromMap(m map[string]interface{}) {
	if v, ok := m["iterations"].(int); ok {
		s.Iterations = v
	}
	if v, ok := m["total_tokens"].(int); ok {
		s.Tokens.Total = v
	}
	if v, ok := m["input_tokens"].(int); ok {
		s.Tokens.Input = v
	}
	if v, ok := m["output_tokens"].(int); ok {
		s.Tokens.Output = v
	}
	if v, ok := m["tools_called"].(int); ok {
		s.ToolsCalled = v
	}
	if v, ok := m["duration"].(time.Duration); ok {
		s.Duration = v
	}
}

// OutputSummary writes the execution summary to stdout or stderr.
func OutputSummary(summary *ExecutionSummary) error {
	// Marshal to JSON with pretty printing
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal summary: %w", err)
	}

	// Write to stderr so it doesn't interfere with normal output
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "=== Execution Summary ===")
	fmt.Fprintln(os.Stderr, string(data))
	fmt.Fprintln(os.Stderr, "========================")
	fmt.Fprintln(os.Stderr, "")

	return nil
}

// CalculateCost estimates the cost in USD for token usage.
func CalculateCost(inputTokens, outputTokens int, model string) float64 {
	// Default pricing (GPT-4o pricing as of 2025)
	var inputCostPerMillion, outputCostPerMillion float64

	switch model {
	case "gpt-4o", "gpt-4o-turbo":
		inputCostPerMillion = 2.50
		outputCostPerMillion = 10.00
	case "gpt-4-turbo", "gpt-4-turbo-2024-04-09":
		inputCostPerMillion = 10.00
		outputCostPerMillion = 30.00
	case "gpt-4", "gpt-4-32k":
		inputCostPerMillion = 30.00
		outputCostPerMillion = 60.00
	case "gpt-3.5-turbo", "gpt-3.5-turbo-16k":
		inputCostPerMillion = 0.50
		outputCostPerMillion = 1.50
	default:
		// Default to conservative pricing
		inputCostPerMillion = 2.50
		outputCostPerMillion = 10.00
	}

	inputCost := float64(inputTokens) / 1_000_000 * inputCostPerMillion
	outputCost := float64(outputTokens) / 1_000_000 * outputCostPerMillion

	return inputCost + outputCost
}

// String returns a human-readable string representation of the summary.
func (s *ExecutionSummary) String() string {
	status := "SUCCESS"
	if !s.Success {
		status = "FAILED"
	}

	return fmt.Sprintf("ExecutionSummary{Program: %s, Status: %s, Iterations: %d, Tokens: %d/%d/%d, Tools: %d, Duration: %s}",
		s.Program, status, s.Iterations, s.Tokens.Input, s.Tokens.Output, s.Tokens.Total, s.ToolsCalled, s.Duration)
}

// ToMap converts the summary to a map for JSON marshaling.
func (s *ExecutionSummary) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"program":      s.Program,
		"success":      s.Success,
		"iterations":   s.Iterations,
		"tools_called": s.ToolsCalled,
		"duration":     s.Duration.String(),
		"model":        s.Model,
		"start_time":   s.StartTime.Format(time.RFC3339),
		"end_time":     s.EndTime.Format(time.RFC3339),
	}

	if s.Error != "" {
		result["error"] = s.Error
	}

	if s.Tokens.Total > 0 {
		result["tokens"] = map[string]interface{}{
			"total":  s.Tokens.Total,
			"input":  s.Tokens.Input,
			"output": s.Tokens.Output,
		}
	}

	if s.AgentCalls != nil {
		result["agent_calls"] = map[string]interface{}{
			"total_calls":      s.AgentCalls.TotalCalls,
			"calls_by_agent":   s.AgentCalls.CallsByAgent,
			"total_duration":   s.AgentCalls.TotalDuration.String(),
			"average_duration": s.AgentCalls.AverageDuration.String(),
			"tokens_used":      s.AgentCalls.TokensUsed,
		}
	}

	return result
}

// MergeAgentCalls merges agent call metrics into the summary.
func (s *ExecutionSummary) MergeAgentCalls(callsSummary *AgentCallsSummary) {
	s.AgentCalls = callsSummary
}

// GetSummaryForOutput returns a JSON string of the summary for output.
func (s *ExecutionSummary) GetSummaryForOutput() (string, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal summary: %w", err)
	}
	return string(data), nil
}

// WriteSummaryToFile writes the summary to a file.
func (s *ExecutionSummary) WriteSummaryToFile(path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal summary: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write summary file: %w", err)
	}

	return nil
}

// ParseSummaryFile reads a summary from a file.
func ParseSummaryFile(path string) (*ExecutionSummary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read summary file: %w", err)
	}

	var summary ExecutionSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, fmt.Errorf("failed to unmarshal summary: %w", err)
	}

	return &summary, nil
}

// CompareSummaries compares two summaries and returns the difference.
func CompareSummaries(before, after *ExecutionSummary) *SummaryDelta {
	return &SummaryDelta{
		Program:          after.Program,
		IterationsDelta:  after.Iterations - before.Iterations,
		TokensDelta:      after.Tokens.Total - before.Tokens.Total,
		ToolsCalledDelta: after.ToolsCalled - before.ToolsCalled,
		DurationDelta:    after.Duration - before.Duration,
	}
}

// SummaryDelta represents the difference between two execution summaries.
type SummaryDelta struct {
	Program          string
	IterationsDelta  int
	TokensDelta      int
	ToolsCalledDelta int
	DurationDelta    time.Duration
}

// String returns a string representation of the delta.
func (d *SummaryDelta) String() string {
	return fmt.Sprintf("SummaryDelta{Program: %s, Iterations: %+d, Tokens: %+d, Tools: %+d, Duration: %+v}",
		d.Program, d.IterationsDelta, d.TokensDelta, d.ToolsCalledDelta, d.DurationDelta)
}

// CreateAgentCallsSummary creates an agent calls summary from call metrics.
func CreateAgentCallsSummary(totalCalls int, callsByAgent map[string]int, totalDuration time.Duration, tokensUsed int) *AgentCallsSummary {
	avgDuration := time.Duration(0)
	if totalCalls > 0 {
		avgDuration = totalDuration / time.Duration(totalCalls)
	}

	return &AgentCallsSummary{
		TotalCalls:      totalCalls,
		CallsByAgent:    callsByAgent,
		TotalDuration:   totalDuration,
		AverageDuration: avgDuration,
		TokensUsed:      tokensUsed,
	}
}
