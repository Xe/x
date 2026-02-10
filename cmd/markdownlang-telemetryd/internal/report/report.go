// Package report implements telemetry report validation for markdownlang-telemetryd.
package report

import (
	"fmt"
)

// Report contains telemetry data received from markdownlang clients.
// This mirrors the telemetry.Report structure from the markdownlang package.
type Report struct {
	// System information
	OS        string `json:"os"`                  // Operating system
	Arch      string `json:"arch"`                // Architecture (amd64, arm64, etc.)
	GoVersion string `json:"go_version"`          // Go runtime version
	NumCPU    int    `json:"num_cpu"`             // Number of CPU cores
	Hostname  string `json:"hostname,omitempty"`  // System hostname
	UnameAll  string `json:"uname_all,omitempty"` // Full uname -a output

	// Git configuration
	GitUserName  string `json:"git_user_name,omitempty"`  // git config user.name
	GitUserEmail string `json:"git_user_email,omitempty"` // git config user.email

	// Program information
	Version       string `json:"version"`                  // Program version
	Program       string `json:"program"`                  // Program name (markdownlang)
	ProgramSHA256 string `json:"program_sha256,omitempty"` // SHA256 hash of program file

	// Execution metrics
	DurationMs    int64    `json:"duration_ms"`              // Execution time in milliseconds
	ToolCallCount int      `json:"tool_call_count"`          // Number of tool calls made
	ToolsUsed     []string `json:"tools_used"`               // Names of tools used
	MCPServers    []string `json:"mcp_servers,omitempty"`    // Names of MCP servers used
	MCPToolsUsed  []string `json:"mcp_tools_used,omitempty"` // MCP tools that were called

	// Model information
	ModelProviderURL string `json:"model_provider_url,omitempty"` // Base URL of LLM provider
	ModelName        string `json:"model_name,omitempty"`         // Model name (e.g., gpt-4o)

	// Environment details
	Shell      string `json:"shell,omitempty"`       // Shell from SHELL env var
	Term       string `json:"term,omitempty"`        // Terminal from TERM env var
	Timezone   string `json:"timezone,omitempty"`    // Local timezone name
	WorkingDir string `json:"working_dir,omitempty"` // Last 2 parts of working directory

	// Timestamp
	Timestamp int64 `json:"timestamp"` // Unix timestamp
}

// Validate checks that all required fields are present and valid.
// Returns a descriptive error if validation fails, nil otherwise.
func (r *Report) Validate() error {
	// Check OS is not empty
	if r.OS == "" {
		return fmt.Errorf("validation failed: OS field is required")
	}

	// Check Arch is not empty
	if r.Arch == "" {
		return fmt.Errorf("validation failed: Arch field is required")
	}

	// Check Version is not empty
	if r.Version == "" {
		return fmt.Errorf("validation failed: Version field is required")
	}

	// Check Program is "markdownlang"
	if r.Program != "markdownlang" {
		return fmt.Errorf("validation failed: Program must be 'markdownlang', got '%s'", r.Program)
	}

	// Check Timestamp is non-zero
	if r.Timestamp == 0 {
		return fmt.Errorf("validation failed: Timestamp field is required and must be non-zero")
	}

	// Check GitUserEmail is not empty
	if r.GitUserEmail == "" {
		return fmt.Errorf("validation failed: GitUserEmail field is required")
	}

	// Check GitUserName is not empty
	if r.GitUserName == "" {
		return fmt.Errorf("validation failed: GitUserName field is required")
	}

	return nil
}
