// Package telemetry implements usage telemetry for markdownlang.
// Telemetry is always enabled.
package telemetry

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	// Default telemetry endpoint
	defaultEndpoint = "https://telemetry.markdownlang.techaro.lol/ingest"

	// HTTP timeout for telemetry requests
	defaultTimeout = 5 * time.Second

	// User agent prefix
	userAgentPrefix = "markdownlang"
)

var (
	// Version is set by the main program
	Version = "dev"

	// ProgramName is set by the main program
	ProgramName = "markdownlang"

	// Endpoint can be overridden for testing
	Endpoint = defaultEndpoint
)

// Report contains telemetry data to send.
// We collect various system and execution metrics to understand usage patterns.
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
	GitUserEmail string `json:"git_user_email,omitempty"` // git config user.email (full email)

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

// Reporter handles sending telemetry data.
type Reporter struct {
	httpClient    *http.Client
	startTime     time.Time
	toolsUsed     map[string]bool
	toolCallCount int
	mcpServers    map[string]bool
	mcpToolsUsed  map[string]bool
	programPath   string
	modelURL      string
	modelName     string
	mu            sync.RWMutex
}

// New creates a new telemetry reporter.
// Telemetry is always enabled.
func New() *Reporter {
	return &Reporter{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		startTime:     time.Now(),
		toolsUsed:     make(map[string]bool),
		toolCallCount: 0,
		mcpServers:    make(map[string]bool),
		mcpToolsUsed:  make(map[string]bool),
	}
}

// RecordTool records that a tool was used during execution.
func (r *Reporter) RecordTool(toolName string) {
	if r != nil {
		r.mu.Lock()
		r.toolsUsed[toolName] = true
		r.toolCallCount++
		r.mu.Unlock()
	}
}

// SetProgramPath sets the path to the program file being executed.
func (r *Reporter) SetProgramPath(path string) {
	if r != nil {
		r.mu.Lock()
		r.programPath = path
		r.mu.Unlock()
	}
}

// SetModel sets the model URL and name.
func (r *Reporter) SetModel(url, name string) {
	if r != nil {
		r.mu.Lock()
		r.modelURL = url
		r.modelName = name
		r.mu.Unlock()
	}
}

// RecordMCPServer records that an MCP server was configured.
func (r *Reporter) RecordMCPServer(serverName string) {
	if r != nil {
		r.mu.Lock()
		r.mcpServers[serverName] = true
		r.mu.Unlock()
	}
}

// RecordMCPTool records that an MCP tool was called.
func (r *Reporter) RecordMCPTool(toolName string) {
	if r != nil {
		r.mu.Lock()
		r.mcpToolsUsed[toolName] = true
		r.mu.Unlock()
	}
}

// ReportDuration sends a telemetry report with the execution duration.
// Errors are logged but do not affect program execution.
func (r *Reporter) ReportDuration() {
	if r == nil {
		return
	}

	duration := time.Since(r.startTime)

	// Collect all the creepy data
	report := r.collectReportData(duration)

	// Send in background - don't block execution
	go r.send(report)
}

// collectReportData gathers all telemetry information.
func (r *Reporter) collectReportData(duration time.Duration) Report {
	// Get locked data
	r.mu.RLock()
	programPath := r.programPath
	modelURL := r.modelURL
	modelName := r.modelName
	toolCallCount := r.toolCallCount

	// Build tool lists
	tools := make([]string, 0, len(r.toolsUsed))
	for tool := range r.toolsUsed {
		tools = append(tools, tool)
	}

	mcpServers := make([]string, 0, len(r.mcpServers))
	for server := range r.mcpServers {
		mcpServers = append(mcpServers, server)
	}

	mcpTools := make([]string, 0, len(r.mcpToolsUsed))
	for tool := range r.mcpToolsUsed {
		mcpTools = append(mcpTools, tool)
	}
	r.mu.RUnlock()

	// Collect system information
	report := Report{
		OS:               runtime.GOOS,
		Arch:             runtime.GOARCH,
		GoVersion:        runtime.Version(),
		NumCPU:           runtime.NumCPU(),
		UnameAll:         getUnameAll(),
		GitUserName:      getGitConfig("user.name"),
		GitUserEmail:     sanitizeEmail(getGitConfig("user.email")),
		Version:          Version,
		Program:          ProgramName,
		ProgramSHA256:    hashFile(programPath),
		DurationMs:       duration.Milliseconds(),
		ToolCallCount:    toolCallCount,
		ToolsUsed:        tools,
		MCPServers:       mcpServers,
		MCPToolsUsed:     mcpTools,
		ModelProviderURL: modelURL,
		ModelName:        modelName,
		Shell:            os.Getenv("SHELL"),
		Term:             os.Getenv("TERM"),
		Timezone:         getTimezone(),
		WorkingDir:       sanitizeWorkingDir(),
		Hostname:         getHostname(),
		Timestamp:        time.Now().Unix(),
	}

	return report
}

// getUnameAll runs uname -a and returns its output.
func getUnameAll() string {
	out, err := exec.Command("uname", "-a").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// getGitConfig returns a git config value.
// Exits with a vague error message if git config is not available.
func getGitConfig(key string) string {
	out, err := exec.Command("git", "config", "--get", key).Output()
	if err != nil {
		slog.Error("improper environment setup", "error", err)
		os.Exit(1)
	}
	return strings.TrimSpace(string(out))
}

// sanitizeEmail returns the full email address.
func sanitizeEmail(email string) string {
	return email // Return full email, no sanitization
}

// hashFile computes the SHA256 hash of a file.
func hashFile(path string) string {
	if path == "" {
		return ""
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// getTimezone returns the local timezone name.
func getTimezone() string {
	// Get timezone name, prefer offset over name
	name, offset := time.Now().Zone()
	if offset != 0 {
		return fmt.Sprintf("UTC%+d", offset/3600)
	}
	return name
}

// sanitizeWorkingDir returns the last 2 parts of the working directory.
func sanitizeWorkingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	parts := strings.Split(wd, string(os.PathSeparator))
	if len(parts) > 2 {
		return strings.Join(parts[len(parts)-2:], string(os.PathSeparator))
	}
	return wd
}

// getHostname returns the system hostname.
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return ""
	}
	return hostname
}

// send sends a telemetry report to the endpoint.
// Errors are logged but do not affect program execution.
func (r *Reporter) send(report Report) {
	if Endpoint == "" {
		return
	}

	// Marshal report
	body, err := json.Marshal(report)
	if err != nil {
		slog.Debug("failed to marshal telemetry report", "error", err)
		return
	}

	// Create request
	req, err := http.NewRequest("POST", Endpoint, bytes.NewReader(body))
	if err != nil {
		slog.Debug("failed to create telemetry request", "error", err)
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", userAgentPrefix, Version))

	// Send request (with timeout)
	resp, err := r.httpClient.Do(req)
	if err != nil {
		slog.Debug("telemetry request failed", "error", err)
		return
	}
	defer resp.Body.Close()

	// We don't care about the response, just that it was sent
	// Log at debug level only
	slog.Debug("telemetry sent", "status", resp.StatusCode)
}

// IsEnabled returns whether telemetry is enabled (always true).
func (r *Reporter) IsEnabled() bool {
	return r != nil
}

// SetVersion sets the version string for telemetry reports.
func SetVersion(version string) {
	Version = version
}

// SetProgramName sets the program name for telemetry reports.
func SetProgramName(name string) {
	ProgramName = name
}

// SetEndpoint sets the telemetry endpoint (for testing).
func SetEndpoint(endpoint string) {
	Endpoint = endpoint
}
