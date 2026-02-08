// Command markdownlang executes markdownlang programs.
package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"within.website/x/cmd/markdownlang/internal/agreement"
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
  -agree       Accept the trans rights agreement (first-time setup only)

Examples:
  # Run a program
  markdownlang -program fizzbuzz.md -input '{"start":1,"end":15}'

  # Run with debug output
  markdownlang -program myagent.md -input '{"url":"https://example.com"}' -debug

  # Save output to file
  markdownlang -program agent.md -input '{"data":[1,2,3]}' -output result.json

  # Run with execution summary
  markdownlang -program fizzbuzz.md -input '{"start":1,"end":10}' -summary

Agreement:
  Before first use, you must accept the trans rights agreement by running:
  markdownlang -agree
  Then type the following phrase exactly:
  "I hereby agree to not harm transgender people and largely leave them alone so they can live their life in peace."

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

	// Handle agreement flag
	if *config.Agree {
		if err := handleAgreement(); err != nil {
			slog.Error("Agreement failed", "error", err)
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("Agreement accepted. You can now use markdownlang.")
		os.Exit(0)
	}

	// Check for agreement before running any program
	if err := agreement.Check(); err != nil {
		fmt.Fprintln(os.Stderr, "\n"+err.Error())
		fmt.Fprintln(os.Stderr, "\nTo accept the agreement, run: markdownlang -agree")
		fmt.Fprintln(os.Stderr)
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

// handleAgreement handles the agreement acceptance process.
func handleAgreement() error {
	fmt.Println("markdownlang Trans Rights Agreement")
	fmt.Println()
	fmt.Println("Before using markdownlang, you must agree to the following:")
	fmt.Println()

	// Get the required phrase for this session
	requiredPhrase, _, err := agreement.GetOrCreateRequiredPhrase()
	if err != nil {
		return fmt.Errorf("failed to get required phrase: %w", err)
	}

	fmt.Println(requiredPhrase)
	fmt.Println()
	fmt.Print("Type the above phrase exactly to accept: ")

	// Read user input
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return fmt.Errorf("failed to read input")
	}

	phrase := scanner.Text()

	// Validate the phrase and get the index
	phraseIndex, err := agreement.ValidatePhrase(phrase)
	if err != nil {
		return err
	}

	// Accept the agreement with the phrase index
	if err := agreement.Accept(phraseIndex); err != nil {
		return fmt.Errorf("failed to save agreement: %w", err)
	}

	return nil
}
