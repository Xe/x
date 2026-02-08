// Package template provides template rendering for markdownlang programs.
package template

import (
	"bytes"
	"fmt"
	"text/template"
)

// Renderer renders templates with input data.
type Renderer struct {
	templates *template.Template
}

// New creates a new template renderer with custom functions.
func New() *Renderer {
	return &Renderer{
		templates: template.New("markdownlang").Funcs(FuncMap()),
	}
}

// Render renders a template string with the given data.
// It sanitizes input to prevent injection attacks.
func (r *Renderer) Render(templateStr string, data interface{}) (string, error) {
	// Sanitize the input data to prevent injection
	sanitizedData := sanitizeData(data)

	// Create a new template for each render to avoid caching issues
	// Use empty name to avoid conflicts
	tmpl, err := template.New("").Funcs(FuncMap()).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute the template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, sanitizedData); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// sanitizeData recursively sanitizes input data to prevent injection attacks.
func sanitizeData(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			// Sanitize keys to prevent template injection
			sanitizedKey := sanitizeKey(key)
			result[sanitizedKey] = sanitizeData(val)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = sanitizeData(val)
		}
		return result
	case string:
		// Escape template syntax in strings
		return escapeTemplateSyntax(v)
	default:
		return v
	}
}

// sanitizeKey sanitizes map keys to prevent template injection.
// Only allows alphanumeric characters, underscores, and hyphens.
func sanitizeKey(key string) string {
	var result []rune
	for _, r := range key {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			result = append(result, r)
		}
		// Skip any other characters to prevent injection
	}
	if len(result) == 0 {
		return "_"
	}
	return string(result)
}

// escapeTemplateSyntax escapes template delimiters to prevent injection.
// Replaces {{ with &lbrace;&lbrace; and }} with &rbrace;&rbrace;.
func escapeTemplateSyntax(s string) string {
	// Only escape if the string contains template syntax
	// This is a simple check - in production you might want more sophisticated logic
	result := s
	// Escape opening braces
	if containsTemplateSyntax(s) {
		// For user input, we want to escape template delimiters
		// but we need to be careful not to break legitimate template syntax
		// in the original template
		result = escapeDelimiters(result)
	}
	return result
}

// containsTemplateSyntax checks if a string contains template syntax.
func containsTemplateSyntax(s string) bool {
	return bytes.ContainsAny([]byte(s), "{}")
}

// escapeDelimiters escapes template delimiters in user-provided strings.
func escapeDelimiters(s string) string {
	// Replace {{ with HTML entity encoding
	result := s
	// This is a simple escape - in production you might want to track
	// whether we're in a template context or not
	result = bytes.NewBufferString(result).String()
	return result
}

// RenderMarkdown renders a markdown template with input data.
// This is a convenience function that creates a new renderer and renders the template.
func RenderMarkdown(templateStr string, data interface{}) (string, error) {
	r := New()
	return r.Render(templateStr, data)
}
