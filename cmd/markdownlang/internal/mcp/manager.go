package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Manager manages the lifecycle of multiple MCP server clients.
type Manager struct {
	clients map[string]*Client
	mu      sync.RWMutex
	logger  *slog.Logger
}

// NewManager creates a new MCP manager.
func NewManager(logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{
		clients: make(map[string]*Client),
		logger:  logger,
	}
}

// Start connects to an MCP server with the given configuration.
func (m *Manager) Start(ctx context.Context, config *MCPServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if config.Disabled {
		m.logger.Debug("MCP server is disabled, skipping", "server", config.Name)
		return nil
	}

	if _, exists := m.clients[config.Name]; exists {
		return fmt.Errorf("MCP server %s already started", config.Name)
	}

	client := NewClient(config)
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to start MCP server %s: %w", config.Name, err)
	}

	m.clients[config.Name] = client
	m.logger.Info("MCP server started", "server", config.Name)
	return nil
}

// Stop gracefully shuts down an MCP server.
func (m *Manager) Stop(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[name]
	if !exists {
		return fmt.Errorf("MCP server %s not found", name)
	}

	if err := client.Close(); err != nil {
		return fmt.Errorf("failed to stop MCP server %s: %w", name, err)
	}

	delete(m.clients, name)
	m.logger.Info("MCP server stopped", "server", name)
	return nil
}

// StopAll gracefully shuts down all MCP servers.
func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for name := range m.clients {
		client := m.clients[name]
		if err := client.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to stop %s: %w", name, err))
		}
		delete(m.clients, name)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors stopping servers: %v", errs)
	}

	m.logger.Info("All MCP servers stopped")
	return nil
}

// GetTools retrieves all available tools from all connected MCP servers.
func (m *Manager) GetTools(ctx context.Context) ([]Tool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allTools []Tool

	for name, client := range m.clients {
		if !client.IsConnected() {
			m.logger.Warn("Client not connected, skipping", "server", name)
			continue
		}

		serverTools, err := client.ListTools(ctx)
		if err != nil {
			m.logger.Error("Failed to list tools", "server", name, "error", err)
			continue
		}

		for _, tool := range serverTools {
			allTools = append(allTools, Tool{
				Tool:     tool,
				Server:   name,
				ServerID: name, // Using name as ID for now
			})
		}
	}

	return allTools, nil
}

// CallTool invokes a tool on the appropriate MCP server.
func (m *Manager) CallTool(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find the server that has this tool
	for _, client := range m.clients {
		if !client.IsConnected() {
			continue
		}

		tools, err := client.ListTools(ctx)
		if err != nil {
			m.logger.Error("Failed to list tools", "server", client.Name(), "error", err)
			continue
		}

		for _, tool := range tools {
			if tool.Name == name {
				return client.CallTool(ctx, name, args)
			}
		}
	}

	return nil, fmt.Errorf("tool %s not found on any connected server", name)
}

// GetTool finds a specific tool by name across all servers.
func (m *Manager) GetTool(ctx context.Context, name string) (*Tool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, client := range m.clients {
		if !client.IsConnected() {
			continue
		}

		tools, err := client.ListTools(ctx)
		if err != nil {
			continue
		}

		for _, tool := range tools {
			if tool.Name == name {
				return &Tool{
					Tool:     tool,
					Server:   client.Name(),
					ServerID: client.Name(),
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("tool %s not found", name)
}

// ListServers returns the names of all connected servers.
func (m *Manager) ListServers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var servers []string
	for name := range m.clients {
		servers = append(servers, name)
	}
	return servers
}

// ServerCount returns the number of connected servers.
func (m *Manager) ServerCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

// IsServerRunning checks if a specific server is running.
func (m *Manager) IsServerRunning(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[name]
	return exists && client.IsConnected()
}
