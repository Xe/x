package mcp

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Tool represents an MCP tool with metadata about its source server.
type Tool struct {
	Tool     *mcp.Tool
	Server   string // Human-readable server name
	ServerID string // Unique server identifier
}

// String returns a string representation of the tool.
func (t Tool) String() string {
	return fmt.Sprintf("%s:%s", t.Server, t.Tool.Name)
}

// ToolSchema represents the JSON schema for a tool's input.
type ToolSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]any         `json:"inputSchema,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ToSchema converts a Tool to a ToolSchema for JSON serialization.
func (t Tool) ToSchema() ToolSchema {
	schema := ToolSchema{
		Name:        t.Tool.Name,
		Description: t.Tool.Description,
		Metadata: map[string]interface{}{
			"server":   t.Server,
			"serverId": t.ServerID,
		},
	}

	if t.Tool.InputSchema != nil {
		// InputSchema is already any, convert to map[string]any
		if inputSchema, ok := t.Tool.InputSchema.(map[string]any); ok {
			schema.InputSchema = inputSchema
		}
	}

	return schema
}

// ToolRegistry tracks available tools and routes calls to the appropriate server.
type ToolRegistry struct {
	tools map[string]*Tool // tool name -> tool
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*Tool),
	}
}

// Register adds a tool to the registry.
func (r *ToolRegistry) Register(tool *Tool) error {
	if tool == nil || tool.Tool == nil {
		return fmt.Errorf("cannot register nil tool")
	}

	name := tool.Tool.Name
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	// Check for duplicate tool names (tools with same name from different servers)
	if existing, exists := r.tools[name]; exists {
		// If the same tool name exists from a different server, we need to disambiguate
		// For now, we'll allow it but prefer the first registered tool
		// In a future version, we might want to namespace tools or return an error
		return fmt.Errorf("tool %s already registered from server %s", name, existing.Server)
	}

	r.tools[name] = tool
	return nil
}

// Unregister removes a tool from the registry.
func (r *ToolRegistry) Unregister(name string) {
	delete(r.tools, name)
}

// Get retrieves a tool by name.
func (r *ToolRegistry) Get(name string) (*Tool, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

// List returns all registered tools.
func (r *ToolRegistry) List() []*Tool {
	tools := make([]*Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ListByServer returns all tools from a specific server.
func (r *ToolRegistry) ListByServer(serverID string) []*Tool {
	var tools []*Tool
	for _, tool := range r.tools {
		if tool.ServerID == serverID {
			tools = append(tools, tool)
		}
	}
	return tools
}

// Clear removes all tools from the registry.
func (r *ToolRegistry) Clear() {
	r.tools = make(map[string]*Tool)
}

// Count returns the number of registered tools.
func (r *ToolRegistry) Count() int {
	return len(r.tools)
}

// ToSchemas converts all registered tools to ToolSchema format.
func (r *ToolRegistry) ToSchemas() []ToolSchema {
	schemas := make([]ToolSchema, 0, len(r.tools))
	for _, tool := range r.tools {
		schemas = append(schemas, tool.ToSchema())
	}
	return schemas
}

// ToolCallResult represents the result of a tool call.
type ToolCallResult struct {
	Content []any       `json:"content"`
	IsError bool        `json:"isError,omitempty"`
	Meta    *ToolMeta   `json:"meta,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ToolMeta contains metadata about a tool call.
type ToolMeta struct {
	ToolName string `json:"toolName"`
	Server   string `json:"server"`
	Success  bool   `json:"success"`
}

// NewToolCallResult creates a new tool call result from an MCP CallToolResult.
func NewToolCallResult(toolName, server string, result *mcp.CallToolResult) *ToolCallResult {
	content := make([]any, len(result.Content))
	for i, c := range result.Content {
		// Convert content based on type
		switch v := c.(type) {
		case *mcp.TextContent:
			content[i] = map[string]any{
				"type": "text",
				"text": v.Text,
			}
		case *mcp.ImageContent:
			content[i] = map[string]any{
				"type":     "image",
				"data":     v.Data,
				"mimeType": v.MIMEType, // Fixed: MIMEType not MimeType
			}
		case *mcp.EmbeddedResource:
			content[i] = map[string]any{
				"type": "embedded_resource",
			}
			if v.Resource != nil {
				content[i] = map[string]any{
					"type": "embedded_resource",
					"uri":  v.Resource.URI,
				}
			}
		default:
			content[i] = map[string]any{
				"type": "unknown",
				"data": fmt.Sprintf("%v", v),
			}
		}
	}

	return &ToolCallResult{
		Content: content,
		IsError: result.IsError,
		Meta: &ToolMeta{
			ToolName: toolName,
			Server:   server,
			Success:  !result.IsError,
		},
	}
}
