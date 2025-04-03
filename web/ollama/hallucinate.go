//go:build ignore

package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/swaggest/jsonschema-go"
	"within.website/x/valid"
	"within.website/x/web"
)

// Hallucinate prompts the model to hallucinate a "valid" JSON response to the given input.
func Hallucinate[T valid.Interface](ctx context.Context, c *Client, opts HallucinateOpts) (*T, error) {
	reflector := jsonschema.Reflector{}

	schema, err := reflector.Reflect(new(T))
	if err != nil {
		return nil, err
	}

	inp := &CompleteRequest{
		Model:     opts.Model,
		Messages:  opts.Messages,
		Format:    p("json"),
		Stream:    true,
		KeepAlive: (9999 * time.Minute).String(),
	}
	tries := 0
	for tries <= 5 {
		tries++

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		buf := &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(inp); err != nil {
			return nil, fmt.Errorf("ollama: error encoding request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", buf)
		if err != nil {
			return nil, fmt.Errorf("ollama: error creating request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("ollama: error making request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, web.NewError(http.StatusOK, resp)
		}

		whitespaceCount := 0

		dec := json.NewDecoder(resp.Body)
		buf = bytes.NewBuffer(nil)

		for {
			var cr CompleteResponse
			err := dec.Decode(&cr)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					return nil, fmt.Errorf("ollama: error decoding response: %w", err)
				} else {
					break
				}
			}

			//slog.Debug("got response", "response", cr.Message.Content)

			if _, err := fmt.Fprint(buf, cr.Message.Content); err != nil {
				return nil, fmt.Errorf("ollama: error writing response to buffer: %w", err)
			}

			for _, r := range cr.Message.Content {
				if r == '\n' {
					whitespaceCount++
				}
			}

			if whitespaceCount > 10 {
				cancel()
			}

			//slog.Debug("buffer is now", "buffer", buf.String())

			var result T
			if err := json.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(&result); err != nil {
				//slog.Debug("error decoding response", "err", err)
				continue
			}

			if err := result.Valid(); err != nil {
				slog.Debug("error validating response", "err", err)
				continue
			}

			//slog.Debug("got valid response", "response", result)
			cancel()

			return &result, nil
		}
	}

	return nil, fmt.Errorf("ollama: failed to hallucinate a valid response after 5 tries")
}
