// Command markdownlang executes markdownlang programs.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"within.website/x/cmd/markdownlang/internal/config"
	"within.website/x/cmd/markdownlang/internal/executor"
	"within.website/x/internal"
)

const (
	usage = `markdownlang - Execute markdownlang programs with LLMs

Usage:
  markdownlang -program <file.md> -input '{"key":"value"}' [-output result.json]

Flags:
  -program     Path to the markdownlang program (.md file) [required]
  -input       JSON input for the program (default: {})
  -output      Path to write output JSON (default: stdout)
  -model       OpenAI model to use (default: gpt-4o)
  -api-key     OpenAI API key (default: $OPENAI_API_KEY)
  -base-url    LLM base URL (default: $OPENAI_BASE_URL)
  -debug       Enable verbose debug logging
  -summary     Output JSON execution summary with metrics

Examples:
  # Run a program
  markdownlang -program fizzbuzz.md -input '{"start":1,"end":15}'

  # Run with debug output
  markdownlang -program myagent.md -input '{"url":"https://example.com"}' -debug

  # Save output to file
  markdownlang -program agent.md -input '{"data":[1,2,3]}' -output result.json

  # Run with execution summary
  markdownlang -program fizzbuzz.md -input '{"start":1,"end":10}' -summary

Documentation:
  https://github.com/Xe/x/tree/master/cmd/markdownlang`
)

func main() {
	internal.HandleStartup()

	// Show usage if no arguments provided
	if len(os.Args) == 1 {
		fmt.Println(usage)
		os.Exit(0)
	}

	// Validate flags
	if err := config.Validate(); err != nil {
		slog.Error("Invalid configuration", "error", err)
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	// Create executor
	ex, err := executor.New()
	if err != nil {
		slog.Error("Failed to create executor", "error", err)
		os.Exit(1)
	}

	// Execute program
	ctx := context.Background()
	result, err := ex.Execute(ctx)
	if err != nil {
		slog.Error("Execution failed", "error", err)

		// Output summary even on failure if requested
		if *config.Summary {
			summary := createErrorSummary(ex, err)
			if sumErr := executor.OutputSummary(summary); sumErr != nil {
				slog.Error("Failed to output summary", "error", sumErr)
			}
		}

		os.Exit(1)
	}

	// Output result
	if err := executor.OutputResult(result, *config.Output); err != nil {
		slog.Error("Failed to output result", "error", err)
		os.Exit(1)
	}

	// Output summary if requested
	if *config.Summary {
		summary := createSuccessSummary(ex)
		if err := executor.OutputSummary(summary); err != nil {
			slog.Error("Failed to output summary", "error", err)
		}
	}

	slog.Info("Execution completed successfully", "program", *config.ProgramPath)
}

// createSuccessSummary creates a success execution summary.
func createSuccessSummary(ex *executor.Executor) *executor.ExecutionSummary {
	metrics := ex.GetMetrics()
	startTime, _ := metrics["start_time"].(time.Time)
	endTime, _ := metrics["end_time"].(time.Time)

	summary := executor.NewExecutionSummary(
		*config.ProgramPath,
		metrics,
		true,
		"",
		*config.Model,
		startTime,
		endTime,
	)

	// Add agent calls summary if available
	if agentCalls, ok := metrics["agent_calls"].(map[string]interface{}); ok {
		if totalCalls, ok := agentCalls["total_calls"].(int); ok && totalCalls > 0 {
			callsByAgent := make(map[string]int)
			if byAgent, ok := agentCalls["calls_by_agent"].(map[string]int); ok {
				callsByAgent = byAgent
			}

			var totalDuration time.Duration
			if duration, ok := agentCalls["total_duration"].(time.Duration); ok {
				totalDuration = duration
			}

			tokensUsed := 0
			if tokens, ok := agentCalls["total_tokens"].(int); ok {
				tokensUsed = tokens
			}

			agentCallsSummary := executor.CreateAgentCallsSummary(
				totalCalls,
				callsByAgent,
				totalDuration,
				tokensUsed,
			)
			summary.MergeAgentCalls(agentCallsSummary)
		}
	}

	// Calculate token cost
	if summary.Tokens.Total > 0 {
		summary.Tokens.Cost = executor.CalculateCost(
			summary.Tokens.Input,
			summary.Tokens.Output,
			*config.Model,
		)
	}

	return summary
}

// createErrorSummary creates an error execution summary.
func createErrorSummary(ex *executor.Executor, execErr error) *executor.ExecutionSummary {
	metrics := ex.GetMetrics()
	startTime, _ := metrics["start_time"].(time.Time)
	endTime := time.Now()

	summary := executor.NewExecutionSummary(
		*config.ProgramPath,
		metrics,
		false,
		execErr.Error(),
		*config.Model,
		startTime,
		endTime,
	)

	// Calculate token cost even on error (if we have partial metrics)
	if summary.Tokens.Total > 0 {
		summary.Tokens.Cost = executor.CalculateCost(
			summary.Tokens.Input,
			summary.Tokens.Output,
			*config.Model,
		)
	}

	return summary
}
