// Package mcp provides MCP client management for markdownlang.
// It handles starting MCP servers, managing their lifecycle, and routing tool calls.
package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPServerConfig defines the configuration for starting an MCP server.
type MCPServerConfig struct {
	Name    string            `yaml:"name" json:"name"`
	Command string            `yaml:"command" json:"command"`
	Args    []string          `yaml:"args,omitempty" json:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	// Disabled indicates the server should not be started.
	Disabled bool `yaml:"disabled,omitempty" json:"disabled,omitempty"`
}

// Client wraps an MCP client connection to a single server.
type Client struct {
	config    *MCPServerConfig
	client    *mcp.Client
	session   *mcp.ClientSession
	transport *mcp.CommandTransport
	cancel    context.CancelFunc
	mu        sync.RWMutex
}

// NewClient creates a new MCP client from the given configuration.
func NewClient(config *MCPServerConfig) *Client {
	return &Client{
		config: config,
	}
}

// Connect starts the MCP server and establishes a connection.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.config.Disabled {
		return fmt.Errorf("server %s is disabled", c.config.Name)
	}

	// Create command with environment variables
	cmd := exec.CommandContext(ctx, c.config.Command, c.config.Args...)
	if len(c.config.Env) > 0 {
		env := cmd.Env
		for k, v := range c.config.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	// Create transport
	c.transport = &mcp.CommandTransport{
		Command: cmd,
	}

	// Create client
	impl := &mcp.Implementation{
		Name:    "markdownlang",
		Version: "1.0.0",
	}
	c.client = mcp.NewClient(impl, nil)

	// Create cancelable context for this connection
	clientCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	// Connect to server
	session, err := c.client.Connect(clientCtx, c.transport, nil)
	if err != nil {
		cancel()
		return fmt.Errorf("failed to connect to MCP server %s: %w", c.config.Name, err)
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
	c.transport = nil
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
