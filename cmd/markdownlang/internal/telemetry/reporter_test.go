package telemetry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	// Telemetry is always enabled now
	r := New()
	if r == nil {
		t.Error("New() returned nil reporter")
	}
}

func TestReporter_RecordTool(t *testing.T) {
	r := New()

	// Record some tools
	r.RecordTool("fetch")
	r.RecordTool("python-interpreter")
	r.RecordTool("fetch") // Duplicate

	if len(r.toolsUsed) != 2 {
		t.Errorf("RecordTool() recorded %d tools, want 2", len(r.toolsUsed))
	}

	if !r.toolsUsed["fetch"] {
		t.Error("RecordTool() did not record 'fetch'")
	}

	if !r.toolsUsed["python-interpreter"] {
		t.Error("RecordTool() did not record 'python-interpreter'")
	}
}

func TestReporter_ReportDuration(t *testing.T) {
	// Create test server
	var receivedReport Report
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Check method
		if req.Method != "POST" {
			t.Errorf("expected POST request, got %s", req.Method)
		}

		// Check headers
		receivedHeaders = req.Header

		// Check content type
		if ct := req.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		// Decode and store report
		dec := json.NewDecoder(req.Body)
		if err := dec.Decode(&receivedReport); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Set endpoint to test server
	oldEndpoint := Endpoint
	defer func() { Endpoint = oldEndpoint }()
	Endpoint = server.URL

	// Create reporter and ensure some time passes
	r := New()
	r.RecordTool("test-tool")
	time.Sleep(10 * time.Millisecond) // Ensure duration > 0
	r.ReportDuration()

	// Wait for background goroutine to send
	time.Sleep(100 * time.Millisecond)

	// Verify report
	if receivedReport.Program != ProgramName {
		t.Errorf("program = %s, want %s", receivedReport.Program, ProgramName)
	}

	if receivedReport.Version != Version {
		t.Errorf("version = %s, want %s", receivedReport.Version, Version)
	}

	if len(receivedReport.ToolsUsed) != 1 {
		t.Errorf("tools_used = %v, want [test-tool]", receivedReport.ToolsUsed)
	}

	if receivedReport.DurationMs <= 0 {
		t.Errorf("duration_ms = %d, want > 0", receivedReport.DurationMs)
	}

	// Check user agent
	ua := receivedHeaders.Get("User-Agent")
	if !strings.HasPrefix(ua, userAgentPrefix+"/") {
		t.Errorf("User-Agent = %s, want %s/*", ua, userAgentPrefix)
	}
}

func TestReporter_ReportDurationNilReporter(t *testing.T) {
	// Calling methods on nil reporter should not panic
	var r *Reporter
	r.RecordTool("test-tool")
	r.ReportDuration() // Should not panic
}

func TestReporter_IsEnabled(t *testing.T) {
	tests := []struct {
		name        string
		nilReporter bool
		want        bool
	}{
		{
			name:        "reporter",
			nilReporter: false,
			want:        true,
		},
		{
			name:        "nil reporter",
			nilReporter: true,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r *Reporter
			if !tt.nilReporter {
				r = New()
			}

			if got := r.IsEnabled(); got != tt.want {
				t.Errorf("Reporter.IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReporter_ConcurrentToolRecording(t *testing.T) {
	r := New()

	// Record tools from multiple goroutines
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r.RecordTool(string(rune('a' + i)))
		}(i)
	}
	wg.Wait()

	if len(r.toolsUsed) != 10 {
		t.Errorf("RecordTool() recorded %d tools, want 10", len(r.toolsUsed))
	}
}

func TestReporter_ReportMarshaling(t *testing.T) {
	report := Report{
		OS:         "linux",
		Arch:       "amd64",
		Version:    "1.0.0",
		Program:    "markdownlang",
		DurationMs: 1234,
		ToolsUsed:  []string{"fetch", "python"},
		Timestamp:  1234567890,
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal() failed: %v", err)
	}

	// Verify it can be unmarshaled back
	var decoded Report
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() failed: %v", err)
	}

	if decoded.OS != report.OS {
		t.Errorf("OS = %s, want %s", decoded.OS, report.OS)
	}

	if decoded.Arch != report.Arch {
		t.Errorf("Arch = %s, want %s", decoded.Arch, report.Arch)
	}
}

func TestSetVersion(t *testing.T) {
	oldVersion := Version
	defer func() { Version = oldVersion }()

	SetVersion("2.0.0")
	if Version != "2.0.0" {
		t.Errorf("SetVersion() = %s, want 2.0.0", Version)
	}
}

func TestSetProgramName(t *testing.T) {
	oldName := ProgramName
	defer func() { ProgramName = oldName }()

	SetProgramName("test-program")
	if ProgramName != "test-program" {
		t.Errorf("SetProgramName() = %s, want test-program", ProgramName)
	}
}

func TestSetEndpoint(t *testing.T) {
	oldEndpoint := Endpoint
	defer func() { Endpoint = oldEndpoint }()

	SetEndpoint("http://test.example")
	if Endpoint != "http://test.example" {
		t.Errorf("SetEndpoint() = %s, want http://test.example", Endpoint)
	}
}

func TestReporter_SendFailureDoesNotPanic(t *testing.T) {
	// Use an invalid URL
	oldEndpoint := Endpoint
	defer func() { Endpoint = oldEndpoint }()
	Endpoint = "http://invalid.url-that.does.not.exist:12345"

	r := New()

	// Should not panic
	r.ReportDuration()

	// Wait for background goroutine
	time.Sleep(100 * time.Millisecond)
}

func TestReporter_EmptyEndpoint(t *testing.T) {
	oldEndpoint := Endpoint
	defer func() { Endpoint = oldEndpoint }()
	Endpoint = ""

	r := New()

	// Should not panic
	r.ReportDuration()

	// Wait for potential background goroutine
	time.Sleep(100 * time.Millisecond)
}

func TestReporter_SetProgramPath(t *testing.T) {
	r := New()

	r.SetProgramPath("/test/path/program.md")

	r.mu.RLock()
	if r.programPath != "/test/path/program.md" {
		t.Errorf("SetProgramPath() = %s, want /test/path/program.md", r.programPath)
	}
	r.mu.RUnlock()
}

func TestReporter_SetModel(t *testing.T) {
	r := New()

	r.SetModel("https://api.example.com", "gpt-4")

	r.mu.RLock()
	if r.modelURL != "https://api.example.com" {
		t.Errorf("SetModel() url = %s, want https://api.example.com", r.modelURL)
	}
	if r.modelName != "gpt-4" {
		t.Errorf("SetModel() name = %s, want gpt-4", r.modelName)
	}
	r.mu.RUnlock()
}

func TestReporter_RecordMCPServer(t *testing.T) {
	r := New()

	r.RecordMCPServer("filesystem")
	r.RecordMCPServer("fetch")
	r.RecordMCPServer("filesystem") // Duplicate

	r.mu.RLock()
	if len(r.mcpServers) != 2 {
		t.Errorf("RecordMCPServer() recorded %d servers, want 2", len(r.mcpServers))
	}
	if !r.mcpServers["filesystem"] {
		t.Error("RecordMCPServer() did not record 'filesystem'")
	}
	r.mu.RUnlock()
}

func TestReporter_RecordMCPTool(t *testing.T) {
	r := New()

	r.RecordMCPTool("mcp__filesystem__read_file")
	r.RecordMCPTool("mcp__fetch__fetch")

	r.mu.RLock()
	if len(r.mcpToolsUsed) != 2 {
		t.Errorf("RecordMCPTool() recorded %d tools, want 2", len(r.mcpToolsUsed))
	}
	r.mu.RUnlock()
}

func TestReporter_ToolCallCount(t *testing.T) {
	r := New()

	r.RecordTool("tool1")
	r.RecordTool("tool2")
	r.RecordTool("tool1") // Duplicate - still counts as a call

	r.mu.RLock()
	if r.toolCallCount != 3 {
		t.Errorf("Tool call count = %d, want 3", r.toolCallCount)
	}
	r.mu.RUnlock()
}

func TestSanitizeEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{
			name:  "valid email - returns full email",
			email: "user@example.com",
			want:  "user@example.com",
		},
		{
			name:  "email with subdomain - returns full email",
			email: "user@mail.example.com",
			want:  "user@mail.example.com",
		},
		{
			name:  "invalid email",
			email: "not-an-email",
			want:  "not-an-email",
		},
		{
			name:  "empty email",
			email: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeEmail(tt.email); got != tt.want {
				t.Errorf("sanitizeEmail(%q) = %q, want %q", tt.email, got, tt.want)
			}
		})
	}
}

func TestSanitizeWorkingDir(t *testing.T) {
	// Just test that it doesn't panic and returns something
	result := sanitizeWorkingDir()
	if result == "" {
		// It's ok if it returns empty in some environments (like tests)
		// but we want to ensure it doesn't crash
	}
}

func TestGetHostname(t *testing.T) {
	// Just test that it doesn't panic
	hostname := getHostname()
	// Most systems will have a hostname, but we don't want to fail tests if not
	_ = hostname
}

func TestGetUnameAll(t *testing.T) {
	// Just test that it doesn't panic
	uname := getUnameAll()
	// uname might not be available on all systems
	_ = uname
}

func TestGetTimezone(t *testing.T) {
	// Just test that it doesn't panic and returns something
	tz := getTimezone()
	if tz == "" {
		t.Error("getTimezone() returned empty string")
	}
}
