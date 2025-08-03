package logging

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestLogHTTPRequest(t *testing.T) {
	tests := []struct {
		name         string
		setupRequest func() *http.Request
		status       int
		bytesWritten int64
		duration     time.Duration
		wantContains []string
	}{
		{
			name: "basic GET request",
			setupRequest: func() *http.Request {
				req := &http.Request{
					Method:     "GET",
					RequestURI: "/api/users",
					Proto:      "HTTP/1.1",
					Host:       "example.com",
					RemoteAddr: "192.168.1.1:54321",
					Header:     make(http.Header),
				}
				req.Header.Set("User-Agent", "Mozilla/5.0 (Test)")
				req.Header.Set("Referer", "https://example.com/")
				return req
			},
			status:       200,
			bytesWritten: 1024,
			duration:     150 * time.Millisecond,
			wantContains: []string{
				"example.com",
				"192.168.1.1",
				"- -", // no basic auth user
				"\"GET /api/users HTTP/1.1\"",
				"200 1024",
				"\"https://example.com/\"",
				"\"Mozilla/5.0 (Test)\"",
				"150", // duration in milliseconds
			},
		},
		{
			name: "POST request with basic auth",
			setupRequest: func() *http.Request {
				req := &http.Request{
					Method:     "POST",
					RequestURI: "/api/data",
					Proto:      "HTTP/1.1",
					Host:       "api.example.com",
					RemoteAddr: "10.0.0.1:12345",
					Header:     make(http.Header),
				}
				// Simulate basic auth
				req.SetBasicAuth("testuser", "password")
				req.Header.Set("User-Agent", "curl/7.68.0")
				return req
			},
			status:       201,
			bytesWritten: 512,
			duration:     75 * time.Millisecond,
			wantContains: []string{
				"api.example.com",
				"10.0.0.1",
				"- testuser", // basic auth user
				"\"POST /api/data HTTP/1.1\"",
				"201 512",
				"\"-\"", // no referer
				"\"curl/7.68.0\"",
				"75",
			},
		},
		{
			name: "request with empty/missing fields",
			setupRequest: func() *http.Request {
				req := &http.Request{
					Method: "", // empty method
					Proto:  "", // empty protocol
					Host:   "", // empty host
					Header: make(http.Header),
				}
				req.URL = &url.URL{Path: "/test", RawQuery: "param=value"}
				return req
			},
			status:       404,
			bytesWritten: 0,
			duration:     5 * time.Millisecond,
			wantContains: []string{
				"- -",                                // empty host becomes "-"
				"- -",                                // no remote addr and no auth user
				"\"GET /test?param=value HTTP/1.0\"", // defaults
				"404 0",
				"\"-\" \"-\"", // no referer or user-agent
				"5",
			},
		},
		{
			name: "request with query parameters",
			setupRequest: func() *http.Request {
				req := &http.Request{
					Method:     "GET",
					RequestURI: "/search?q=golang&page=2",
					Proto:      "HTTP/2.0",
					Host:       "search.example.com",
					RemoteAddr: "[::1]:8080",
					Header:     make(http.Header),
				}
				req.Header.Set("User-Agent", "Go-http-client/2.0")
				req.Header.Set("Referer", "https://search.example.com/")
				return req
			},
			status:       200,
			bytesWritten: 2048,
			duration:     250 * time.Millisecond,
			wantContains: []string{
				"search.example.com",
				"::1",
				"\"GET /search?q=golang&page=2 HTTP/2.0\"",
				"200 2048",
				"\"https://search.example.com/\"",
				"\"Go-http-client/2.0\"",
				"250",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			req := tt.setupRequest()

			LogHTTPRequest(&buf, req, tt.status, tt.bytesWritten, tt.duration)

			output := buf.String()
			t.Logf("Output: %s", output)

			// Check that the output contains all expected substrings
			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("Expected output to contain %q, but it didn't. Full output: %s", want, output)
				}
			}

			// Verify the output ends with a newline
			if !strings.HasSuffix(output, "\n") {
				t.Error("Expected output to end with newline")
			}

			// Verify timestamp format is present (basic check)
			if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
				t.Error("Expected output to contain timestamp in brackets")
			}
		})
	}
}

func TestLogHTTPRequestFormat(t *testing.T) {
	// Test the exact format matches expected Apache/nginx combined log format
	req := &http.Request{
		Method:     "GET",
		RequestURI: "/test",
		Proto:      "HTTP/1.1",
		Host:       "example.com",
		RemoteAddr: "127.0.0.1:12345",
		Header:     make(http.Header),
	}
	req.Header.Set("User-Agent", "TestAgent/1.0")
	req.Header.Set("Referer", "https://example.com/home")

	var buf bytes.Buffer
	LogHTTPRequest(&buf, req, 200, 1024, 100*time.Millisecond)

	output := strings.TrimSpace(buf.String())
	parts := strings.Fields(output)

	// The format should be: vhost remoteaddr - remoteuser [timestamp] "method uri protocol" status bytes "referer" "user-agent" duration
	// But quotes and brackets make this tricky to split by fields, so we'll do basic checks

	if len(parts) < 10 {
		t.Errorf("Expected at least 10 space-separated parts, got %d: %v", len(parts), parts)
	}

	// Check specific positions we can verify
	if parts[0] != "example.com" {
		t.Errorf("Expected vhost 'example.com', got %q", parts[0])
	}

	if parts[1] != "127.0.0.1" {
		t.Errorf("Expected remote addr '127.0.0.1', got %q", parts[1])
	}

	if parts[2] != "-" {
		t.Errorf("Expected '-' for empty field, got %q", parts[2])
	}

	if parts[3] != "-" {
		t.Errorf("Expected '-' for remote user, got %q", parts[3])
	}

	// Check that status and bytes are present (they'll be towards the end)
	foundStatus := false
	foundBytes := false
	for _, part := range parts {
		if part == "200" {
			foundStatus = true
		}
		if part == "1024" {
			foundBytes = true
		}
	}

	if !foundStatus {
		t.Error("Expected to find status code '200' in output")
	}

	if !foundBytes {
		t.Error("Expected to find bytes '1024' in output")
	}
}

func TestLogHTTPRequestDurationUnits(t *testing.T) {
	// Test different duration values to ensure millisecond conversion is correct
	testCases := []struct {
		duration time.Duration
		expected string
	}{
		{1 * time.Millisecond, "1"},
		{100 * time.Millisecond, "100"},
		{1 * time.Second, "1000"},
		{1500 * time.Millisecond, "1500"},
		{500 * time.Microsecond, "0"}, // Less than 1ms should be 0
	}

	for _, tc := range testCases {
		t.Run(tc.duration.String(), func(t *testing.T) {
			req := &http.Request{
				Method:     "GET",
				RequestURI: "/test",
				Proto:      "HTTP/1.1",
				Host:       "test.com",
				Header:     make(http.Header),
			}

			var buf bytes.Buffer
			LogHTTPRequest(&buf, req, 200, 0, tc.duration)

			output := buf.String()
			if !strings.HasSuffix(strings.TrimSpace(output), tc.expected) {
				t.Errorf("Expected output to end with %q, got: %s", tc.expected, output)
			}
		})
	}
}
