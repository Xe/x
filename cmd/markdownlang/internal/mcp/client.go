// Package mcp provides MCP client management for markdownlang.
// It handles starting MCP servers, managing their lifecycle, and routing tool calls.
package mcp

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TransportType specifies the transport protocol for MCP connections.
type TransportType string

const (
	// TransportAuto auto-detects the transport type based on URL scheme.
	TransportAuto TransportType = "auto"
	// TransportSSE uses Server-Sent Events (HTTP streaming).
	TransportSSE TransportType = "sse"
	// TransportCommand uses stdio-based command transport.
	TransportCommand TransportType = "command"
)

// MCPServerConfig defines the configuration for starting an MCP server.
type MCPServerConfig struct {
	// Name is the identifier for this server
	Name string `yaml:"name" json:"name"`

	// Command is the executable to run for command-based servers
	Command string `yaml:"command,omitempty" json:"command,omitempty"`

	// Args are the command-line arguments
	Args []string `yaml:"args,omitempty" json:"args,omitempty"`

	// Env contains environment variables for the command
	Env map[string]string `yaml:"env,omitempty" json:"env,omitempty"`

	// URL is the endpoint for HTTP-based MCP servers (SSE, WebSocket, etc.)
	URL string `yaml:"url,omitempty" json:"url,omitempty"`

	// Transport specifies the transport type. If "auto", it will be detected
	// from the URL scheme or default to SSE for HTTP URLs.
	// Defaults to "auto".
	Transport TransportType `yaml:"transport,omitempty" json:"transport,omitempty"`

	// Disabled indicates the server should not be started.
	Disabled bool `yaml:"disabled,omitempty" json:"disabled,omitempty"`
}

// Client wraps an MCP client connection to a single server.
type Client struct {
	config  *MCPServerConfig
	client  *mcp.Client
	session *mcp.ClientSession
	cancel  context.CancelFunc
	mu      sync.RWMutex
}

// NewClient creates a new MCP client from the given configuration.
func NewClient(config *MCPServerConfig) *Client {
	return &Client{
		config: config,
	}
}

// detectTransport determines the appropriate transport type based on URL scheme.
// Returns SSE for http:// and https:// URLs by default.
// Can be extended to support ws://, wss:// for WebSocket, etc.
func detectTransport(endpoint string) TransportType {
	if endpoint == "" {
		return TransportCommand
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		// If URL parsing fails, default to SSE
		return TransportSSE
	}

	scheme := strings.ToLower(u.Scheme)

	// Auto-detect based on URL scheme
	switch scheme {
	case "ws", "wss":
		// WebSocket transport (not yet implemented, reserved for future)
		return TransportSSE // Fall back to SSE for now
	case "http", "https":
		// Default to SSE for HTTP endpoints
		return TransportSSE
	default:
		// Unknown scheme, try SSE as default
		return TransportSSE
	}
}

// Connect starts the MCP server and establishes a connection.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.config.Disabled {
		return fmt.Errorf("server %s is disabled", c.config.Name)
	}

	// Determine the transport type
	transportType := c.config.Transport
	if transportType == "" || transportType == TransportAuto {
		// Auto-detect based on URL
		if c.config.URL != "" {
			transportType = detectTransport(c.config.URL)
		} else if c.config.Command != "" {
			transportType = TransportCommand
		} else {
			return fmt.Errorf("must specify either URL or Command for server %s", c.config.Name)
		}
	}

	// Create the appropriate transport
	var transport mcp.Transport
	impl := &mcp.Implementation{
		Name:    "markdownlang",
		Version: "1.0.0",
	}
	c.client = mcp.NewClient(impl, nil)

	// Create cancelable context for this connection
	clientCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	switch transportType {
	case TransportSSE:
		if c.config.URL == "" {
			cancel()
			return fmt.Errorf("SSE transport requires URL for server %s", c.config.Name)
		}
		transport = &mcp.SSEClientTransport{
			Endpoint: c.config.URL,
		}

	case TransportCommand:
		if c.config.Command == "" {
			cancel()
			return fmt.Errorf("command transport requires Command for server %s", c.config.Name)
		}
		cmd := exec.CommandContext(ctx, c.config.Command, c.config.Args...)
		if len(c.config.Env) > 0 {
			env := cmd.Env
			for k, v := range c.config.Env {
				env = append(env, fmt.Sprintf("%s=%s", k, v))
			}
			cmd.Env = env
		}
		transport = &mcp.CommandTransport{
			Command: cmd,
		}

	default:
		cancel()
		return fmt.Errorf("unsupported transport type %q for server %s", transportType, c.config.Name)
	}

	// Connect to server
	session, err := c.client.Connect(clientCtx, transport, nil)
	if err != nil {
		cancel()
		return fmt.Errorf("failed to connect to MCP server %s using %s transport: %w", c.config.Name, transportType, err)
	}

	c.session = session
	return nil
}

// Close gracefully shuts down the MCP client connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}

	if c.session != nil {
		if err := c.session.Close(); err != nil {
			return fmt.Errorf("failed to close session for %s: %w", c.config.Name, err)
		}
		c.session = nil
	}

	c.client = nil
	return nil
}

// ListTools retrieves all available tools from the MCP server.
func (c *Client) ListTools(ctx context.Context) ([]*mcp.Tool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.session == nil {
		return nil, fmt.Errorf("client %s not connected", c.config.Name)
	}

	// List tools with pagination
	var allTools []*mcp.Tool
	cursor := ""

	for {
		result, err := c.session.ListTools(ctx, &mcp.ListToolsParams{
			Cursor: cursor,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list tools from %s: %w", c.config.Name, err)
		}

		allTools = append(allTools, result.Tools...)

		// No more results if NextCursor is empty
		if result.NextCursor == "" {
			break
		}
		cursor = result.NextCursor
	}

	return allTools, nil
}

// CallTool invokes a tool on the MCP server with the given arguments.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.session == nil {
		return nil, fmt.Errorf("client %s not connected", c.config.Name)
	}

	params := &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	}

	result, err := c.session.CallTool(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("tool call %s failed on %s: %w", name, c.config.Name, err)
	}

	return result, nil
}

// Name returns the server name for this client.
func (c *Client) Name() string {
	return c.config.Name
}

// IsConnected returns whether the client is currently connected.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.session != nil
}
