package config

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidMaxRequestBody = errors.New("invalid max_request_body value")
	ErrInvalidMaxHeaderSize  = errors.New("invalid max_header_size value")
	ErrInvalidMaxHeaderCount = errors.New("invalid max_header_count value")
)

// Limits holds request size limits for a domain to prevent DoS attacks.
type Limits struct {
	// MaxRequestBody is the maximum request body size (e.g., "10MB", "1GB").
	// If empty, defaults to 10MB.
	MaxRequestBody string `hcl:"max_request_body,optional"`

	// MaxHeaderSize is the maximum size of headers in bytes (e.g., "1MB").
	// If empty, defaults to 1MB.
	MaxHeaderSize string `hcl:"max_header_size,optional"`

	// MaxHeaderCount is the maximum number of headers allowed.
	// If empty or zero, defaults to 100.
	MaxHeaderCount int `hcl:"max_header_count,optional"`
}

// DefaultLimits returns the default limits for a domain.
func DefaultLimits() Limits {
	return Limits{
		MaxRequestBody: "10MB",
		MaxHeaderSize:  "1MB",
		MaxHeaderCount: 100,
	}
}

// MaxRequestBodyBytes returns the max request body size in bytes.
func (l Limits) MaxRequestBodyBytes() int64 {
	if l.MaxRequestBody == "" {
		return 10 * 1024 * 1024 // 10MB default
	}
	bytes, err := parseBytes(l.MaxRequestBody)
	if err != nil {
		return 10 * 1024 * 1024 // 10MB default on error
	}
	return bytes
}

// MaxHeaderSizeBytes returns the max header size in bytes.
func (l Limits) MaxHeaderSizeBytes() int64 {
	if l.MaxHeaderSize == "" {
		return 1 * 1024 * 1024 // 1MB default
	}
	bytes, err := parseBytes(l.MaxHeaderSize)
	if err != nil {
		return 1 * 1024 * 1024 // 1MB default on error
	}
	return bytes
}

// MaxHeaderCountValue returns the max header count.
func (l Limits) MaxHeaderCountValue() int {
	if l.MaxHeaderCount <= 0 {
		return 100 // default
	}
	return l.MaxHeaderCount
}

// Valid validates the limits configuration.
func (l Limits) Valid() error {
	var errs []error

	if l.MaxRequestBody != "" {
		if _, err := parseBytes(l.MaxRequestBody); err != nil {
			errs = append(errs, fmt.Errorf("%w %q: %w", ErrInvalidMaxRequestBody, l.MaxRequestBody, err))
		}
	}

	if l.MaxHeaderSize != "" {
		if _, err := parseBytes(l.MaxHeaderSize); err != nil {
			errs = append(errs, fmt.Errorf("%w %q: %w", ErrInvalidMaxHeaderSize, l.MaxHeaderSize, err))
		}
	}

	if l.MaxHeaderCount < 0 {
		errs = append(errs, fmt.Errorf("%w: must be non-negative", ErrInvalidMaxHeaderCount))
	}

	if len(errs) != 0 {
		return errors.Join(errs...)
	}

	return nil
}

// parseBytes parses a byte size string like "10MB" into bytes.
func parseBytes(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	var num int64
	var unit string
	n, err := fmt.Sscanf(s, "%d%s", &num, &unit)

	// Handle the cases:
	// - n=0: nothing was parsed (error)
	// - n=1: only number was parsed, EOF error is expected (valid, no unit)
	// - n=2: both number and unit were parsed (valid)
	// - n=2 with error: error (shouldn't happen but handle it)
	// - n=1 with non-EOF error: error

	if n == 0 {
		return 0, fmt.Errorf("invalid format: %w", err)
	}

	if n == 1 {
		// Check if it's an EOF error (expected when no unit)
		if err != nil && err.Error() != "unexpected EOF" && err.Error() != "EOF" {
			return 0, fmt.Errorf("invalid format: %w", err)
		}
		// No unit provided, treat as bytes
		unit = ""
	}

	if n == 2 && err != nil {
		return 0, fmt.Errorf("invalid format: %w", err)
	}

	if num <= 0 {
		return 0, fmt.Errorf("size must be positive")
	}

	switch unit {
	case "B", "":
		return num, nil
	case "KB":
		return num * 1024, nil
	case "MB":
		return num * 1024 * 1024, nil
	case "GB":
		return num * 1024 * 1024 * 1024, nil
	case "TB":
		return num * 1024 * 1024 * 1024 * 1024, nil
	default:
		return 0, fmt.Errorf("unknown unit: %s (use B, KB, MB, GB, or TB)", unit)
	}
}
