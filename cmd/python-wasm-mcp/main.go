package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"within.website/x"
	"within.website/x/internal"
	"within.website/x/llm/codeinterpreter/python"
)

var (
	bind   = flag.String("bind", "", "TCP host:port to bind HTTP to")
	apiKey = flag.String("api-key", "", "API key required for Authorization Bearer header")
)

type Input struct {
	Code string `json:"code" jsonschema:"The python code to execute"`
}

func Python(ctx context.Context, req *mcp.CallToolRequest, input Input) (*mcp.CallToolResult, *python.Result, error) {
	result, err := python.Run(ctx, nil, input.Code)
	if err != nil {
		return nil, nil, err
	}

	return nil, result, nil
}

func main() {
	internal.HandleStartup()

	srv := mcp.NewServer(&mcp.Implementation{Name: "python-wasm-mcp", Version: x.Version}, nil)
	mcp.AddTool(srv, &mcp.Tool{Name: "run", Description: "Run Python code"}, Python)

	switch *bind {
	case "":
		if err := srv.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			log.Fatal(err)
		}

	default:
		// Base MCP HTTP handler.
		inner := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
			return srv
		}, nil)

		// Optional bearer token authentication.
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if *apiKey != "" {
				if r.Header.Get("Authorization") != "Bearer "+*apiKey {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
			}
			inner.ServeHTTP(w, r)
		})

		log.Printf("MCP server listening on %s", *bind)
		if err := http.ListenAndServe(*bind, h); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}
}
