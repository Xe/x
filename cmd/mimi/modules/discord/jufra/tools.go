package jufra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"connectrpc.com/connect"
	"within.website/x/llm/codeinterpreter/python"
	falin "within.website/x/migroserbices/falin/gen"
	"within.website/x/web/ollama"
)

var normalTools = []ollama.Function{
	{
		Name:        "code_interpreter",
		Description: "Run the given Python code",
		Parameters: ollama.Param{
			Type: "object",
			Properties: ollama.Properties{
				"code": {
					Type:        "string",
					Description: "The Python code to run",
				},
			},
			Required: []string{"code"},
		},
	},
	{
		Name:        "draw_image",
		Description: "Use Midjourney to draw an image from the given prompt",
		Parameters: ollama.Param{
			Type: "object",
			Properties: ollama.Properties{
				"prompt": {
					Type:        "string",
					Description: "The prompt to use",
				},
			},
			Required: []string{"prompt"},
		},
	},
	// {
	// 	Name:        "reply",
	// 	Description: "Reply to the message",
	// 	Parameters: ollama.Param{
	// 		Type: "object",
	// 		Properties: ollama.Properties{
	// 			"message": {
	// 				Type:        "string",
	// 				Description: "The message to send",
	// 			},
	// 		},
	// 		Required: []string{"message"},
	// 	},
	// },
}

type pythonCodeArgs struct {
	Code string `json:"code"`
}

func (pca *pythonCodeArgs) Valid() error {
	if pca.Code == "" {
		return errors.New("missing code parameter")
	}

	return nil
}

func (m *Module) runPythonCode(ctx context.Context, tc ollama.ToolCall) (*ollama.Message, error) {
	var args pythonCodeArgs
	if err := json.Unmarshal(tc.Arguments, &args); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	res, err := python.Run(ctx, nil, args.Code)
	if err != nil {
		return &ollama.Message{
			Role:    "tool",
			Content: jsonString(map[string]string{"error": err.Error(), "stdout": res.Stdout, "stderr": res.Stderr}),
		}, nil
	}

	slog.Info("python code ran", "res", res, "args", args)

	return &ollama.Message{
		Role:    "tool",
		Content: jsonString(res),
	}, nil
}

func (m *Module) getTools() []ollama.Tool {
	var result []ollama.Tool

	for _, tool := range normalTools {
		result = append(result, ollama.Tool{
			Type:     "function",
			Function: tool,
		})
	}

	return result
}

type drawImageArgs struct {
	Prompt string `json:"prompt"`
}

func (dia *drawImageArgs) Valid() error {
	if dia.Prompt == "" {
		return errors.New("missing prompt parameter")
	}

	return nil
}

func (m *Module) drawImage(ctx context.Context, tc ollama.ToolCall, channelID string) (*ollama.Message, error) {
	var args drawImageArgs
	if err := json.Unmarshal(tc.Arguments, &args); err != nil {
		return nil, err
	}

	go m.EventuallySendImage(channelID, args.Prompt)

	return &ollama.Message{
		Role: "tool",
		Content: jsonString(map[string]string{
			"instruction": "Rephrase this: I'm working on the image! It may take me a minute to think it up.",
		}),
	}, nil
}

func (m *Module) EventuallySendImage(channelID string, prompt string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := m.eventuallySendImage(ctx, channelID, prompt); err != nil {
		slog.Error("failed to send image", "err", err)
	}
}

func (m *Module) eventuallySendImage(ctx context.Context, channelID string, prompt string) error {
	tempDir, err := os.MkdirTemp("", "mimi-image-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	ir, err := m.falin.GenerateImage(ctx, connect.NewRequest(&falin.GenerateImageRequest{
		Prompt:    "an anime depiction of " + prompt,
		Model:     *falinModel,
		NumImages: 1,
	}))
	if err != nil {
		return fmt.Errorf("failed to predict: %w", err)
	}

	resp, err := http.Get(ir.Msg.Images[0].Url)
	if err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}
	defer resp.Body.Close()

	imgPath := filepath.Join(tempDir, "image.jpg")
	imgFile, err := os.Create(imgPath)
	if err != nil {
		return fmt.Errorf("failed to create image file: %w", err)
	}

	if _, err := io.Copy(imgFile, resp.Body); err != nil {
		return fmt.Errorf("failed to write image file: %w", err)
	}

	if _, err := imgFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek image file: %w", err)
	}

	msg, err := m.sess.ChannelFileSendWithMessage(channelID, "Here's the image!\n\n```"+prompt+"\n```", "image.jpg", imgFile)
	if err != nil {
		return fmt.Errorf("failed to send image: %w", err)
	}

	slog.Info("sent image", "msg", msg)

	return nil
}
