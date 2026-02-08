// Package agent implements runtime metrics for markdownlang program execution.
package agent

import (
	"fmt"
	"time"
)

// Metrics tracks runtime statistics for agent execution.
type Metrics struct {
	// Iterations is the number of iterations the agent loop performed.
	Iterations int

	// TotalTokens is the total number of tokens used across all API calls.
	TotalTokens int

	// InputTokens is the number of tokens in the input.
	InputTokens int

	// OutputTokens is the number of tokens in the output.
	OutputTokens int

	// ToolsCalled is the number of tool calls made.
	ToolsCalled int

	// Duration is the total time taken for execution.
	Duration time.Duration

	// ErrorCount is the number of errors encountered during execution.
	ErrorCount int

	// LastError is the last error encountered, if any.
	LastError error
}

// String returns a string representation of the metrics.
func (m Metrics) String() string {
	return fmt.Sprintf("Metrics{iterations=%d tokens=%d/%d/%d tools=%d duration=%s errors=%d}",
		m.Iterations, m.InputTokens, m.OutputTokens, m.TotalTokens, m.ToolsCalled, m.Duration, m.ErrorCount)
}

// Reset clears all metrics.
func (m *Metrics) Reset() {
	*m = Metrics{}
}

// TokenUsage returns a TokenUsage with the token counts.
func (m *Metrics) TokenUsage() TokenUsage {
	return TokenUsage{
		Total:  m.TotalTokens,
		Input:  m.InputTokens,
		Output: m.OutputTokens,
	}
}

// SuccessRate calculates the success rate as a percentage.
// Returns 0 if no iterations were run.
func (m *Metrics) SuccessRate() float64 {
	if m.Iterations == 0 {
		return 0
	}
	if m.ErrorCount == 0 {
		return 100.0
	}
	return float64(m.Iterations-m.ErrorCount) / float64(m.Iterations) * 100.0
}

// AverageTokensPerIteration returns the average number of tokens per iteration.
// Returns 0 if no iterations were run.
func (m *Metrics) AverageTokensPerIteration() float64 {
	if m.Iterations == 0 {
		return 0
	}
	return float64(m.TotalTokens) / float64(m.Iterations)
}

// TokenUsage represents token usage statistics.
type TokenUsage struct {
	// Total is the total number of tokens used.
	Total int

	// Input is the number of tokens in the input.
	Input int

	// Output is the number of tokens in the output.
	Output int
}

// Cost estimates the cost in USD for the token usage.
// This uses GPT-5 Pro pricing as a baseline.
// Actual costs may vary based on the model used.
func (t TokenUsage) Cost() float64 {
	// GPT-5 Pro pricing (as of 2025, approximate)
	const inputCostPerMillion = 2.50
	const outputCostPerMillion = 10.00

	inputCost := float64(t.Input) / 1_000_000 * inputCostPerMillion
	outputCost := float64(t.Output) / 1_000_000 * outputCostPerMillion

	return inputCost + outputCost
}

// String returns a string representation of the token usage.
func (t TokenUsage) String() string {
	return fmt.Sprintf("TokenUsage{Total: %d, Input: %d, Output: %d, Cost: $%.4f}",
		t.Total, t.Input, t.Output, t.Cost())
}

// MetricsSnapshot captures a snapshot of metrics at a point in time.
type MetricsSnapshot struct {
	Metrics
	Timestamp time.Time
}

// Snapshot creates a snapshot of the current metrics.
func (m *Metrics) Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		Metrics:   *m,
		Timestamp: time.Now(),
	}
}

// MetricsDelta calculates the difference between two metric snapshots.
type MetricsDelta struct {
	// Iterations is the change in iteration count.
	Iterations int

	// TotalTokens is the change in total token count.
	TotalTokens int

	// InputTokens is the change in input token count.
	InputTokens int

	// OutputTokens is the change in output token count.
	OutputTokens int

	// ToolsCalled is the change in tool call count.
	ToolsCalled int

	// Duration is the time elapsed between snapshots.
	Duration time.Duration
}

// Delta calculates the difference between two snapshots.
func Delta(from, to MetricsSnapshot) MetricsDelta {
	return MetricsDelta{
		Iterations:   to.Metrics.Iterations - from.Metrics.Iterations,
		TotalTokens:  to.Metrics.TotalTokens - from.Metrics.TotalTokens,
		InputTokens:  to.Metrics.InputTokens - from.Metrics.InputTokens,
		OutputTokens: to.Metrics.OutputTokens - from.Metrics.OutputTokens,
		ToolsCalled:  to.Metrics.ToolsCalled - from.Metrics.ToolsCalled,
		Duration:     to.Timestamp.Sub(from.Timestamp),
	}
}

// String returns a string representation of the delta.
func (d MetricsDelta) String() string {
	return fmt.Sprintf("MetricsDelta{Iterations: %d, Tokens: %d, Tools: %d, Duration: %s}",
		d.Iterations, d.TotalTokens, d.ToolsCalled, d.Duration)
}
