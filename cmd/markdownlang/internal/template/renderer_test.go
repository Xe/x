package template

import (
	"testing"
)

func TestRenderer_Render_SimpleVariable(t *testing.T) {
	r := New()
	data := map[string]interface{}{
		"name": "Xe",
	}

	result, err := r.Render("Hello {{ .name }}", data)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	expected := "Hello Xe"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestRenderer_Render_NestedField(t *testing.T) {
	r := New()
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "Xe",
		},
	}

	result, err := r.Render("Hello {{ .user.name }}", data)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	expected := "Hello Xe"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestRenderer_Render_If(t *testing.T) {
	r := New()
	data := map[string]interface{}{
		"debug": true,
		"count": 42,
	}

	result, err := r.Render("{{ if .debug }}Debug mode: {{ .count }}{{ end }}", data)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	expected := "Debug mode: 42"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestRenderer_Render_Range(t *testing.T) {
	r := New()
	data := map[string]interface{}{
		"items": []string{"foo", "bar", "baz"},
	}

	result, err := r.Render("{{ range .items }}{{ . }} {{ end }}", data)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	expected := "foo bar baz "
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestRenderer_Render_Upper(t *testing.T) {
	r := New()
	data := map[string]interface{}{
		"text": "hello",
	}

	result, err := r.Render("{{ .text | upper }}", data)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	expected := "HELLO"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestRenderer_Render_Lower(t *testing.T) {
	r := New()
	data := map[string]interface{}{
		"text": "HELLO",
	}

	result, err := r.Render("{{ .text | lower }}", data)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	expected := "hello"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestRenderer_Render_Title(t *testing.T) {
	r := New()
	data := map[string]interface{}{
		"text": "hello world",
	}

	result, err := r.Render("{{ .text | title }}", data)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	expected := "Hello World"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestRenderer_Render_Default(t *testing.T) {
	r := New()
	// Test with nil value (should use default)
	data1 := map[string]interface{}{
		"value": nil,
	}

	result1, err := r.Render("{{ .value | default \"default\" }}", data1)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	expected1 := "default"
	if result1 != expected1 {
		t.Errorf("expected %q, got %q", expected1, result1)
	}

	// Test with present value
	data2 := map[string]interface{}{
		"value": "present",
	}

	result2, err := r.Render("{{ .value | default \"default\" }}", data2)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	expected2 := "present"
	if result2 != expected2 {
		t.Errorf("expected %q, got %q", expected2, result2)
	}
}

func TestRenderer_Render_Len(t *testing.T) {
	r := New()
	data := map[string]interface{}{
		"items": []string{"foo", "bar", "baz"},
	}

	result, err := r.Render("{{ .items | len }}", data)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	expected := "3"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestRenderer_Render_Slice(t *testing.T) {
	r := New()
	data := map[string]interface{}{
		"items": []string{"foo", "bar", "baz", "qux"},
		"start": 0,
		"end":   2,
	}

	result, err := r.Render("{{ range $slice := slice .items .start .end }}{{ $slice }} {{ end }}", data)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	expected := "foo bar "
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestRenderer_Render_Join(t *testing.T) {
	r := New()
	data := map[string]interface{}{
		"items":     []string{"foo", "bar", "baz"},
		"separator": ",",
	}

	result, err := r.Render("{{ join .items .separator }}", data)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	expected := "foo,bar,baz"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestRenderer_Render_Split(t *testing.T) {
	r := New()
	data := map[string]interface{}{
		"text": "foo,bar,baz",
		"sep":  ",",
	}

	result, err := r.Render("{{ range $item := split .text .sep }}{{ $item }} {{ end }}", data)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	expected := "foo bar baz "
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestRenderer_Render_MissingField(t *testing.T) {
	r := New()
	data := map[string]interface{}{
		"existing": "value",
	}

	// Go templates don't error on missing fields by default
	// They output "<no value>"
	result, err := r.Render("{{ .missing }}", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Missing field results in "<no value>"
	if result != "<no value>" {
		t.Errorf("expected '<no value>' for missing field, got %q", result)
	}

	// Test with default value for missing field
	// When field doesn't exist, our default function gets nil and returns the default
	result2, err := r.Render("{{ .missing | default \"default_value\" }}", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Since the field doesn't exist (is nil), default kicks in
	expected2 := "default_value"
	if result2 != expected2 {
		t.Errorf("expected %q, got %q", expected2, result2)
	}

	// Test with field that exists but is nil
	data3 := map[string]interface{}{
		"value": nil,
	}
	result3, err := r.Render("{{ .value | default \"default_value\" }}", data3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected3 := "default_value"
	if result3 != expected3 {
		t.Errorf("expected %q, got %q", expected3, result3)
	}
}

func TestRenderer_Render_ComplexTemplate(t *testing.T) {
	r := New()
	data := map[string]interface{}{
		"debug": true,
		"files": []string{"file1.txt", "file2.txt"},
	}

	result, err := r.Render("{{ if .debug }}Processing {{ .files | len }} files{{ end }}", data)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	expected := "Processing 2 files"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFuncMap_DefaultFunc(t *testing.T) {
	fm := FuncMap()

	defaultFn, ok := fm["default"].(func(interface{}, interface{}) interface{})
	if !ok {
		t.Fatal("default function not found or wrong signature")
	}

	// Note: In Go template pipelines, the default value comes first
	// Test with present value
	result1 := defaultFn("default", "present")
	if result1 != "present" {
		t.Errorf("expected 'present', got %v", result1)
	}

	// Test with empty string
	result2 := defaultFn("default", "")
	if result2 != "default" {
		t.Errorf("expected 'default', got %v", result2)
	}

	// Test with nil
	result3 := defaultFn("default", nil)
	if result3 != "default" {
		t.Errorf("expected 'default', got %v", result3)
	}
}

func TestFuncMap_LenFunc(t *testing.T) {
	fm := FuncMap()

	lenFn, ok := fm["len"].(func(interface{}) (int, error))
	if !ok {
		t.Fatal("len function not found or wrong signature")
	}

	// Test with string
	result1, err := lenFn("hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result1 != 5 {
		t.Errorf("expected 5, got %d", result1)
	}

	// Test with slice
	result2, err := lenFn([]string{"foo", "bar"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2 != 2 {
		t.Errorf("expected 2, got %d", result2)
	}

	// Test with unsupported type
	_, err = lenFn(42)
	if err == nil {
		t.Error("expected error for unsupported type, got nil")
	}
}

func TestFuncMap_SliceFunc(t *testing.T) {
	fm := FuncMap()

	sliceFn, ok := fm["slice"].(func(interface{}, int, int) (interface{}, error))
	if !ok {
		t.Fatal("slice function not found or wrong signature")
	}

	// Test with string
	result1, err := sliceFn("hello", 0, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result1 != "he" {
		t.Errorf("expected 'he', got %v", result1)
	}

	// Test with slice
	result2, err := sliceFn([]string{"foo", "bar", "baz"}, 0, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2.([]string)[1] != "bar" {
		t.Errorf("expected 'bar', got %v", result2)
	}

	// Test with invalid indices
	_, err = sliceFn("hello", 0, 10)
	if err == nil {
		t.Error("expected error for invalid indices, got nil")
	}
}

func TestSanitizeKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid key", "valid_key", "valid_key"},
		{"key with special chars", "key<script>", "keyscript"},
		{"key with spaces", "key with spaces", "keywithspaces"},
		{"empty key", "", "_"},
		{"key with template syntax", "{{key}}", "key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeKey(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestRenderMarkdown(t *testing.T) {
	data := map[string]interface{}{
		"name":  "Xe",
		"count": 42,
	}

	result, err := RenderMarkdown("Hello {{ .name }}, count: {{ .count }}", data)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	expected := "Hello Xe, count: 42"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
