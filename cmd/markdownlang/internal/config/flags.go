// Package config defines command-line flags for markdownlang.
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

var (
	// ProgramPath is the path to the markdownlang program (.md file)
	ProgramPath = flag.String("program", "", "Path to the markdownlang program (.md file)")

	// Input is the JSON input string for the program
	Input = flag.String("input", "{}", "JSON input for the program (default: {})")

	// Output is the path to write the output JSON file
	Output = flag.String("output", "", "Path to write the output JSON file (default: stdout)")

	// Model is the OpenAI model to use
	Model = flag.String("model", "gpt-4o", "OpenAI model to use")

	// APIKey is the OpenAI API key
	APIKey = flag.String("api-key", os.Getenv("OPENAI_API_KEY"), "OpenAI API key (default: $OPENAI_API_KEY)")

	// BaseURL is the LLM base URL
	BaseURL = flag.String("base-url", os.Getenv("OPENAI_BASE_URL"), "LLM base URL (default: $OPENAI_BASE_URL)")

	// Debug enables verbose logging
	Debug = flag.Bool("debug", false, "Enable verbose debug logging")

	// Summary enables JSON execution summary output
	Summary = flag.Bool("summary", false, "Output JSON execution summary with metrics")

	// Agree enables agreement acceptance mode
	Agree = flag.Bool("agree", false, "Accept the trans rights agreement")
)

// Validate validates the flag configuration.
func Validate() error {
	// In agreement mode, we don't need a program path
	if *Agree {
		return nil
	}

	// Check if program path is provided
	if *ProgramPath == "" {
		return fmt.Errorf("program path is required (use -program flag)")
	}

	// Check if program file exists
	if _, err := os.Stat(*ProgramPath); os.IsNotExist(err) {
		return fmt.Errorf("program file does not exist: %s", *ProgramPath)
	}

	// Validate that input is valid JSON
	var inputJSON interface{}
	if err := json.Unmarshal([]byte(*Input), &inputJSON); err != nil {
		return fmt.Errorf("invalid JSON input: %w", err)
	}

	return nil
}
