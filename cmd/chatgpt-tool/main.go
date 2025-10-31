package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"within.website/x/internal"
)

var (
	openAIAPIBase = flag.String("openai-base-url", "http://zohar:11434/v1", "OpenAI API base URL")
	openAIAPIKey  = flag.String("openai-api-key", "", "OpenAI API key")
	openAIModel   = flag.String("openai-model", "gpt-oss:120b", "OpenAI model")
)

func main() {
	internal.HandleStartup()

	client := openai.NewClient(
		option.WithAPIKey(*openAIAPIKey),
		option.WithBaseURL(*openAIAPIBase),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Read question from command line arguments or prompt user
	var question string
	if flag.NArg() > 0 {
		question = strings.Join(flag.Args(), " ")
	} else {
		fmt.Print("Enter your question: ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			question = scanner.Text()
		}
	}

	print("> ")
	println(question)

	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("Answer user questions using the python tool to help."),
			openai.UserMessage(question),
		},
		Tools: []openai.ChatCompletionToolUnionParam{
			openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
				Name:        "get_weather",
				Description: openai.String("Get weather at the given location"),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]string{
							"type": "string",
						},
					},
					"required": []string{"location"},
				},
			}),
			openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
				Name:        "python",
				Description: openai.String("Run Python code and return the result"),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]any{
						"code": map[string]string{
							"type": "string",
						},
					},
					"required": []string{"code"},
				},
			}),
		},
		Seed:  openai.Int(0),
		Model: openai.ChatModel(*openAIModel),
	}

	// Make initial chat completion request
	completion, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		panic(err)
	}

	toolCalls := completion.Choices[0].Message.ToolCalls

	// Return early if there are no tool calls
	if len(toolCalls) == 0 {
		fmt.Println(completion.Choices[0].Message.Content)
		return
	}

	// If there is a was a function call, continue the conversation
	params.Messages = append(params.Messages, completion.Choices[0].Message.ToParam())
	for _, toolCall := range toolCalls {
		if toolCall.Function.Name == "get_weather" {
			// Extract the location from the function call arguments
			var args map[string]interface{}
			err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
			if err != nil {
				panic(err)
			}
			location := args["location"].(string)

			// Simulate getting weather data
			weatherData := getWeather(location)

			// Print the weather data
			fmt.Printf("Weather in %s: %s\n", location, weatherData)

			params.Messages = append(params.Messages, openai.ToolMessage(weatherData, toolCall.ID))
		} else if toolCall.Function.Name == "run_python" {
			// Extract the code from the function call arguments
			var args map[string]interface{}
			err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
			if err != nil {
				panic(err)
			}
			code := args["code"].(string)

			result, err := Python(ctx, PythonInput{Code: code})
			if err != nil {
				panic(err)
			}

			// Print the result
			fmt.Printf("Python result: %s\n", result)

			var buf bytes.Buffer
			if json.NewEncoder(&buf).Encode(result); err != nil {
				panic(err)
			}

			params.Messages = append(params.Messages, openai.ToolMessage(buf.String(), toolCall.ID))
		}
	}

	completion, err = client.Chat.Completions.New(ctx, params)
	if err != nil {
		panic(err)
	}

	println(completion.Choices[0].Message.Content)
}

// Mock function to simulate weather data retrieval
func getWeather(location string) string {
	// In a real implementation, this function would call a weather API
	return "Sunny, 25°C"
}

// Function to run Python code
func runPython(code string) string {
	// Create the command to run Python with the provided code
	cmd := exec.Command("python3", "-c", code)

	// Capture stdout and stderr
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()
	if err != nil {
		// If there was an error, return the error message
		return fmt.Sprintf("Error: %s, stderr: %s", err.Error(), stderr.String())
	}

	// Return the output, trimming any trailing whitespace
	return strings.TrimSpace(out.String())
}
