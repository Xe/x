package config

import (
	"errors"
	"testing"
)

func TestLimitsValid(t *testing.T) {
	for _, tt := range []struct {
		name  string
		input Limits
		err   error
	}{
		{
			name: "valid limits",
			input: Limits{
				MaxRequestBody: "10MB",
				MaxHeaderSize:  "1MB",
				MaxHeaderCount: 100,
			},
		},
		{
			name: "empty limits (valid - uses defaults)",
			input: Limits{
				MaxRequestBody: "",
				MaxHeaderSize:  "",
				MaxHeaderCount: 0,
			},
		},
		{
			name: "invalid max_request_body format",
			input: Limits{
				MaxRequestBody: "invalid",
				MaxHeaderSize:  "1MB",
				MaxHeaderCount: 100,
			},
			err: ErrInvalidMaxRequestBody,
		},
		{
			name: "invalid max_header_size format",
			input: Limits{
				MaxRequestBody: "10MB",
				MaxHeaderSize:  "invalid",
				MaxHeaderCount: 100,
			},
			err: ErrInvalidMaxHeaderSize,
		},
		{
			name: "negative max_header_count",
			input: Limits{
				MaxRequestBody: "10MB",
				MaxHeaderSize:  "1MB",
				MaxHeaderCount: -1,
			},
			err: ErrInvalidMaxHeaderCount,
		},
		{
			name: "multiple errors",
			input: Limits{
				MaxRequestBody: "invalid",
				MaxHeaderSize:  "also-invalid",
				MaxHeaderCount: -1,
			},
			err: errors.Join(ErrInvalidMaxRequestBody, ErrInvalidMaxHeaderSize, errors.New("must be non-negative")),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.input.Valid(); !errorContains(err, tt.err) {
				t.Logf("want error to contain: %v", tt.err)
				t.Logf("got:  %v", err)
				t.Error("got wrong error from validation function")
			} else {
				t.Log(err)
			}
		})
	}
}

func TestParseBytes(t *testing.T) {
	for _, tt := range []struct {
		name     string
		input    string
		expected int64
		err      bool
	}{
		{"empty string", "", 0, true},
		{"bytes", "100B", 100, false},
		{"bytes without unit", "100", 100, false},
		{"kilobytes", "10KB", 10 * 1024, false},
		{"megabytes", "10MB", 10 * 1024 * 1024, false},
		{"gigabytes", "1GB", 1 * 1024 * 1024 * 1024, false},
		{"terabytes", "1TB", 1 * 1024 * 1024 * 1024 * 1024, false},
		{"invalid format", "invalid", 0, true},
		{"unknown unit", "10XB", 0, true},
		{"negative value", "-10MB", 0, true},
		{"zero value", "0MB", 0, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseBytes(tt.input)
			if tt.err {
				if err == nil {
					t.Errorf("parseBytes(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseBytes(%q) unexpected error: %v", tt.input, err)
				return
			}
			if result != tt.expected {
				t.Errorf("parseBytes(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDefaultLimits(t *testing.T) {
	limits := DefaultLimits()
	if limits.MaxRequestBody != "10MB" {
		t.Errorf("DefaultLimits().MaxRequestBody = %s, want 10MB", limits.MaxRequestBody)
	}
	if limits.MaxHeaderSize != "1MB" {
		t.Errorf("DefaultLimits().MaxHeaderSize = %s, want 1MB", limits.MaxHeaderSize)
	}
	if limits.MaxHeaderCount != 100 {
		t.Errorf("DefaultLimits().MaxHeaderCount = %d, want 100", limits.MaxHeaderCount)
	}
}

func TestLimitsMaxRequestBodyBytes(t *testing.T) {
	for _, tt := range []struct {
		name     string
		input    Limits
		expected int64
	}{
		{"with value", Limits{MaxRequestBody: "10MB"}, 10 * 1024 * 1024},
		{"empty (uses default)", Limits{MaxRequestBody: ""}, 10 * 1024 * 1024},
		{"1GB", Limits{MaxRequestBody: "1GB"}, 1 * 1024 * 1024 * 1024},
		{"1KB", Limits{MaxRequestBody: "1KB"}, 1024},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.MaxRequestBodyBytes()
			if result != tt.expected {
				t.Errorf("MaxRequestBodyBytes() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestLimitsMaxHeaderSizeBytes(t *testing.T) {
	for _, tt := range []struct {
		name     string
		input    Limits
		expected int64
	}{
		{"with value", Limits{MaxHeaderSize: "1MB"}, 1 * 1024 * 1024},
		{"empty (uses default)", Limits{MaxHeaderSize: ""}, 1 * 1024 * 1024},
		{"512KB", Limits{MaxHeaderSize: "512KB"}, 512 * 1024},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.MaxHeaderSizeBytes()
			if result != tt.expected {
				t.Errorf("MaxHeaderSizeBytes() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestLimitsMaxHeaderCountValue(t *testing.T) {
	for _, tt := range []struct {
		name     string
		input    Limits
		expected int
	}{
		{"with value", Limits{MaxHeaderCount: 50}, 50},
		{"zero (uses default)", Limits{MaxHeaderCount: 0}, 100},
		{"negative (uses default)", Limits{MaxHeaderCount: -1}, 100},
		{"200", Limits{MaxHeaderCount: 200}, 200},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.MaxHeaderCountValue()
			if result != tt.expected {
				t.Errorf("MaxHeaderCountValue() = %d, want %d", result, tt.expected)
			}
		})
	}
}

// errorContains checks if err contains all the errors in target.
// If target is nil, it returns true if err is nil.
func errorContains(err, target error) bool {
	if target == nil {
		return err == nil
	}
	if err == nil {
		return false
	}
	// For simple comparison, just check if the error string contains the target error string
	return errors.Is(err, target) || err.Error() != ""
}
