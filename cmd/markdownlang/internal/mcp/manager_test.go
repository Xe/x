package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestNewManager verifies that a new manager is created correctly.
func TestNewManager(t *testing.T) {
	mgr := NewManager(nil)

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	if mgr.clients == nil {
		t.Error("clients map is nil")
	}

	if mgr.ServerCount() != 0 {
		t.Errorf("expected 0 servers, got %d", mgr.ServerCount())
	}
}

// TestNewClient verifies that a new client is created correctly.
func TestNewClient(t *testing.T) {
	config := &MCPServerConfig{
		Name:    "test-server",
		Command: "echo",
		Args:    []string{"hello"},
	}

	client := NewClient(config)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.Name() != "test-server" {
		t.Errorf("expected name 'test-server', got '%s'", client.Name())
	}

	if client.IsConnected() {
		t.Error("new client should not be connected")
	}
}

// TestClientConnectInvalid verifies that connecting to an invalid command fails.
func TestClientConnectInvalid(t *testing.T) {
	config := &MCPServerConfig{
		Name:    "invalid-server",
		Command: "this-command-does-not-exist-12345",
	}

	client := NewClient(config)
	ctx := context.Background()
	err := client.Connect(ctx)

	if err == nil {
		t.Error("expected error connecting to invalid command, got nil")
	}
}

// TestDisabledServer verifies that disabled servers are not started.
func TestDisabledServer(t *testing.T) {
	mgr := NewManager(nil)
	config := &MCPServerConfig{
		Name:     "disabled-server",
		Command:  "echo",
		Disabled: true,
	}

	ctx := context.Background()
	err := mgr.Start(ctx, config)

	if err != nil {
		t.Errorf("expected no error starting disabled server, got %v", err)
	}

	if mgr.ServerCount() != 0 {
		t.Errorf("expected 0 servers for disabled config, got %d", mgr.ServerCount())
	}

	if mgr.IsServerRunning("disabled-server") {
		t.Error("disabled server should not be running")
	}
}

// TestManagerStartStop verifies basic start and stop operations.
// Note: This test requires a valid MCP server binary.
// To run integration tests, build and use a real MCP server like python-wasm-mcp.
func TestManagerStartStop(t *testing.T) {
	t.Skip("requires valid MCP server binary - use real MCP server for integration testing")
}

// TestManagerStartDuplicate verifies that starting a duplicate server fails.
// Note: This test requires a valid MCP server binary.
func TestManagerStartDuplicate(t *testing.T) {
	t.Skip("requires valid MCP server binary - use real MCP server for integration testing")
}

// TestManagerStopNonExistent verifies that stopping a non-existent server fails.
func TestManagerStopNonExistent(t *testing.T) {
	mgr := NewManager(nil)
	ctx := context.Background()

	err := mgr.Stop(ctx, "non-existent")
	if err == nil {
		t.Error("expected error stopping non-existent server, got nil")
	}
}

// TestManagerStopAll verifies that all servers are stopped.
// Note: This test requires valid MCP server binaries.
func TestManagerStopAll(t *testing.T) {
	t.Skip("requires valid MCP server binaries - use real MCP servers for integration testing")
}

// TestManagerListServers verifies that server names are listed correctly.
// Note: This test requires valid MCP server binaries.
func TestManagerListServers(t *testing.T) {
	t.Skip("requires valid MCP server binaries - use real MCP servers for integration testing")
}

// TestToolRegistry verifies that the tool registry works correctly.
func TestToolRegistry(t *testing.T) {
	reg := NewToolRegistry()

	if reg.Count() != 0 {
		t.Errorf("expected 0 tools, got %d", reg.Count())
	}

	// Register a tool
	tool := &Tool{
		Tool: &mcp.Tool{
			Name:        "test-tool",
			Description: "A test tool",
		},
		Server:   "test-server",
		ServerID: "test-server",
	}

	err := reg.Register(tool)
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	if reg.Count() != 1 {
		t.Errorf("expected 1 tool, got %d", reg.Count())
	}

	// Get the tool
	retrieved, exists := reg.Get("test-tool")
	if !exists {
		t.Error("tool not found after registration")
	}

	if retrieved.Tool.Name != "test-tool" {
		t.Errorf("expected tool name 'test-tool', got '%s'", retrieved.Tool.Name)
	}

	// List by server
	serverTools := reg.ListByServer("test-server")
	if len(serverTools) != 1 {
		t.Errorf("expected 1 tool from server, got %d", len(serverTools))
	}

	// Unregister
	reg.Unregister("test-tool")
	if reg.Count() != 0 {
		t.Errorf("expected 0 tools after unregister, got %d", reg.Count())
	}
}

// TestToolRegistryDuplicate verifies that duplicate tools are handled.
func TestToolRegistryDuplicate(t *testing.T) {
	reg := NewToolRegistry()

	tool1 := &Tool{
		Tool:     &mcp.Tool{Name: "dup-tool"},
		Server:   "server1",
		ServerID: "server1",
	}

	tool2 := &Tool{
		Tool:     &mcp.Tool{Name: "dup-tool"},
		Server:   "server2",
		ServerID: "server2",
	}

	// First registration should succeed
	err := reg.Register(tool1)
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Second registration with same name should fail
	err = reg.Register(tool2)
	if err == nil {
		t.Error("expected error registering duplicate tool, got nil")
	}
}

// TestToolString verifies the String representation of a tool.
func TestToolString(t *testing.T) {
	tool := Tool{
		Tool:   &mcp.Tool{Name: "my-tool"},
		Server: "my-server",
	}

	expected := "my-server:my-tool"
	if tool.String() != expected {
		t.Errorf("expected '%s', got '%s'", expected, tool.String())
	}
}

// TestManagerGetToolsContext verifies that GetTools respects context cancellation.
// Note: This test requires a valid MCP server binary.
func TestManagerGetToolsContext(t *testing.T) {
	t.Skip("requires valid MCP server binary - use real MCP server for integration testing")
}

// TestClientConnectTimeout verifies that connection attempts respect timeouts.
func TestClientConnectTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test in short mode")
	}

	config := &MCPServerConfig{
		Name:    "timeout-server",
		Command: "sleep",
		Args:    []string{"10"}, // Sleep for 10 seconds
	}

	client := NewClient(config)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := client.Connect(ctx)
	if err == nil {
		t.Error("expected timeout error, got nil")
		client.Close()
	}
}

// BenchmarkManagerGetTools benchmarks listing tools from multiple servers.
// Note: This benchmark requires a valid MCP server binary.
func BenchmarkManagerGetTools(b *testing.B) {
	b.Skip("requires valid MCP server binary - use real MCP server for integration testing")
}
