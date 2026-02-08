// Package mcp tests MCP client functionality.
package mcp

import (
	"testing"
)

func TestDetectTransport(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     TransportType
	}{
		{
			name:     "HTTP URL",
			endpoint: "http://localhost:3000/mcp",
			want:     TransportSSE,
		},
		{
			name:     "HTTPS URL",
			endpoint: "https://example.com/mcp",
			want:     TransportSSE,
		},
		{
			name:     "WebSocket URL",
			endpoint: "ws://localhost:3000/mcp",
			want:     TransportSSE, // Falls back to SSE for now
		},
		{
			name:     "Secure WebSocket URL",
			endpoint: "wss://example.com/mcp",
			want:     TransportSSE, // Falls back to SSE for now
		},
		{
			name:     "Empty endpoint",
			endpoint: "",
			want:     TransportCommand,
		},
		{
			name:     "Invalid URL",
			endpoint: "://invalid",
			want:     TransportSSE, // Defaults to SSE on parse error
		},
		{
			name:     "URL with port",
			endpoint: "http://localhost:8080/sse",
			want:     TransportSSE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectTransport(tt.endpoint)
			if got != tt.want {
				t.Errorf("detectTransport(%q) = %v, want %v", tt.endpoint, got, tt.want)
			}
		})
	}
}

func TestTransportTypeString(t *testing.T) {
	tests := []struct {
		t    TransportType
		want string
	}{
		{TransportAuto, "auto"},
		{TransportSSE, "sse"},
		{TransportCommand, "command"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := string(tt.t)
			if got != tt.want {
				t.Errorf("TransportType = %q, want %q", got, tt.want)
			}
		})
	}
}
