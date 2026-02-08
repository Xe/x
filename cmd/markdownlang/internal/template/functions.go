// Package template provides custom template functions for markdownlang.
package template

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
)

// FuncMap returns a map of custom template functions.
func FuncMap() template.FuncMap {
	return template.FuncMap{
		"upper":   strings.ToUpper,
		"lower":   strings.ToLower,
		"title":   strings.Title,
		"default": defaultFunc,
		"len":     lenFunc,
		"slice":   sliceFunc,
		"join":    strings.Join,
		"split":   strings.Split,
	}
}

// defaultFunc returns the default value if the given value is empty or zero.
// Usage: {{ .value | default "default_value" }}
// Note: In Go template pipelines, arguments are passed as (default, value)
func defaultFunc(defaultValue interface{}, value interface{}) interface{} {
	if value == nil {
		return defaultValue
	}

	switch v := value.(type) {
	case string:
		if v == "" {
			return defaultValue
		}
	case []string:
		if len(v) == 0 {
			return defaultValue
		}
	case []interface{}:
		if len(v) == 0 {
			return defaultValue
		}
	case int, int64, float64:
		// For numeric types, zero is a valid value
		// Only use default if it's actually zero
		if v == 0 || v == int64(0) || v == float64(0) {
			return defaultValue
		}
	case bool:
		// For booleans, false is a valid value
		// Only use default if it's actually false and we want to override
		// This is a bit tricky, so we'll be conservative
	}

	return value
}

// lenFunc returns the length of a string, slice, or map.
// Usage: {{ .items | len }}
func lenFunc(value interface{}) (int, error) {
	switch v := value.(type) {
	case string:
		return len(v), nil
	case []string:
		return len(v), nil
	case []interface{}:
		return len(v), nil
	case map[string]interface{}:
		return len(v), nil
	default:
		return 0, fmt.Errorf("len: unsupported type %T", value)
	}
}

// sliceFunc returns a slice of a string or slice.
// Usage: {{ .items | slice 0 5 }}
func sliceFunc(value interface{}, start, end int) (interface{}, error) {
	switch v := value.(type) {
	case string:
		runes := []rune(v)
		if start < 0 || end > len(runes) || start > end {
			return nil, fmt.Errorf("slice: invalid indices %d, %d for string of length %d", start, end, len(runes))
		}
		return string(runes[start:end]), nil
	case []string:
		if start < 0 || end > len(v) || start > end {
			return nil, fmt.Errorf("slice: invalid indices %d, %d for slice of length %d", start, end, len(v))
		}
		return v[start:end], nil
	case []interface{}:
		if start < 0 || end > len(v) || start > end {
			return nil, fmt.Errorf("slice: invalid indices %d, %d for slice of length %d", start, end, len(v))
		}
		return v[start:end], nil
	default:
		return nil, fmt.Errorf("slice: unsupported type %T", value)
	}
}

// escapeString escapes special characters to prevent injection attacks.
// This converts potentially dangerous characters to HTML entities.
func escapeString(s string) string {
	var buf bytes.Buffer
	template.HTMLEscape(&buf, []byte(s))
	return buf.String()
}

// sanitizeValue sanitizes a value to prevent injection attacks.
// For strings, it escapes HTML entities. For other types, it returns as-is.
func sanitizeValue(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return escapeString(v)
	case []string:
		result := make([]string, len(v))
		for i, s := range v {
			result[i] = escapeString(s)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = sanitizeValue(val)
		}
		return result
	default:
		return value
	}
}
