package parser

import (
	"encoding/json"
	"strings"
	"testing"

	"within.website/x/cmd/markdownlang/internal/mcp"
)

func TestParseProgram(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		want        *Program
		wantErr     bool
		errContains string
	}{
		{
			name: "valid markdown with front matter",
			source: `---
name: test_program
description: A test program
input:
  type: object
  properties:
    name:
      type: string
output:
  type: string
imports:
  - ./foo
  - strings
mcp_servers:
  - name: test_server
    command: test
    args: []
---

# Some content
`,
			want: &Program{
				Name:         "test_program",
				Description:  "A test program",
				InputSchema:  jsonSchema(`{"properties":{"name":{"type":"string"}},"type":"object"}`),
				OutputSchema: jsonSchema(`{"type":"string"}`),
				Imports:      []string{"./foo", "strings"},
				MCPServers: []mcp.MCPServerConfig{
					{Name: "test_server", Command: "test", Args: []string{}},
				},
			},
			wantErr: false,
		},
		{
			name: "missing opening delimiter",
			source: `name: test_program
description: A test program
`,
			wantErr:     true,
			errContains: "no opening --- delimiter",
		},
		{
			name: "missing closing delimiter",
			source: `---
name: test_program
description: A test program
`,
			wantErr:     true,
			errContains: "no closing --- delimiter",
		},
		{
			name: "empty front matter",
			source: `---
---
`,
			wantErr:     true,
			errContains: "front matter is empty",
		},
		{
			name: "invalid YAML",
			source: `---
name: [invalid
description: test
input:
  type: string
output:
  type: string
---
`,
			wantErr:     true,
			errContains: "your YAML is garbage",
		},
		{
			name: "missing required name",
			source: `---
description: test
input:
  type: string
output:
  type: string
---
`,
			wantErr:     true,
			errContains: "your program is nameless",
		},
		{
			name: "missing required description",
			source: `---
name: test
input:
  type: string
output:
  type: string
---
`,
			wantErr:     true,
			errContains: "no description",
		},
		{
			name: "missing input schema",
			source: `---
name: test
description: test
output:
  type: string
---
`,
			wantErr:     true,
			errContains: "this program takes no input",
		},
		{
			name: "missing output schema",
			source: `---
name: test
description: test
input:
  type: string
---
`,
			wantErr:     true,
			errContains: "this program produces no output",
		},
		{
			name: "empty import path",
			source: `---
name: test
description: test
input:
  type: string
output:
  type: string
imports:
  - ""
---
`,
			wantErr:     true,
			errContains: "empty import path",
		},
		{
			name: "import path with ..",
			source: `---
name: test
description: test
input:
  type: string
output:
  type: string
imports:
  - ../foo
---
`,
			wantErr:     true,
			errContains: "contains '..'",
		},
		{
			name: "valid relative import",
			source: `---
name: test
description: test
input:
  type: string
output:
  type: string
imports:
  - ./foo/bar
---
`,
			want: &Program{
				Name:         "test",
				Description:  "test",
				InputSchema:  jsonSchema(`{"type":"string"}`),
				OutputSchema: jsonSchema(`{"type":"string"}`),
				Imports:      []string{"./foo/bar"},
			},
			wantErr: false,
		},
		{
			name: "valid absolute import",
			source: `---
name: test
description: test
input:
  type: string
output:
  type: string
imports:
  - /usr/lib/foo
---
`,
			want: &Program{
				Name:         "test",
				Description:  "test",
				InputSchema:  jsonSchema(`{"type":"string"}`),
				OutputSchema: jsonSchema(`{"type":"string"}`),
				Imports:      []string{"/usr/lib/foo"},
			},
			wantErr: false,
		},
		{
			name: "valid stdlib import",
			source: `---
name: test
description: test
input:
  type: string
output:
  type: string
imports:
  - strings
  - fmt
  - github.com/foo/bar
---
`,
			want: &Program{
				Name:         "test",
				Description:  "test",
				InputSchema:  jsonSchema(`{"type":"string"}`),
				OutputSchema: jsonSchema(`{"type":"string"}`),
				Imports:      []string{"strings", "fmt", "github.com/foo/bar"},
			},
			wantErr: false,
		},
		{
			name: "valid nested object schema",
			source: `---
name: test
description: test
input:
  type: object
  properties:
    user:
      type: object
      properties:
        name:
          type: string
        age:
          type: integer
output:
  type: boolean
---
`,
			want: &Program{
				Name:         "test",
				Description:  "test",
				InputSchema:  jsonSchema(`{"properties":{"user":{"properties":{"age":{"type":"integer"},"name":{"type":"string"}},"type":"object"}},"type":"object"}`),
				OutputSchema: jsonSchema(`{"type":"boolean"}`),
			},
			wantErr: false,
		},
		{
			name: "valid array schema",
			source: `---
name: test
description: test
input:
  type: array
  items:
    type: string
output:
  type: number
---
`,
			want: &Program{
				Name:         "test",
				Description:  "test",
				InputSchema:  jsonSchema(`{"items":{"type":"string"},"type":"array"}`),
				OutputSchema: jsonSchema(`{"type":"number"}`),
			},
			wantErr: false,
		},
		{
			name: "mcp servers with args",
			source: `---
name: test
description: test
input:
  type: string
output:
  type: string
mcp_servers:
  - name: server1
    command: binary1
    args:
      - --port
      - "8080"
  - name: server2
    command: /usr/bin/binary2
    args: []
---
`,
			want: &Program{
				Name:         "test",
				Description:  "test",
				InputSchema:  jsonSchema(`{"type":"string"}`),
				OutputSchema: jsonSchema(`{"type":"string"}`),
				MCPServers: []mcp.MCPServerConfig{
					{Name: "server1", Command: "binary1", Args: []string{"--port", "8080"}},
					{Name: "server2", Command: "/usr/bin/binary2", Args: []string{}},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseProgram(tt.source)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseProgram() expected error containing %q, got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
					t.Errorf("ParseProgram() error = %q, want error containing %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseProgram() unexpected error: %v", err)
				return
			}

			if got.Name != tt.want.Name {
				t.Errorf("ParseProgram().Name = %q, want %q", got.Name, tt.want.Name)
			}

			if got.Description != tt.want.Description {
				t.Errorf("ParseProgram().Description = %q, want %q", got.Description, tt.want.Description)
			}

			if !jsonEqual(got.InputSchema, tt.want.InputSchema) {
				t.Errorf("ParseProgram().InputSchema = %s, want %s", got.InputSchema, tt.want.InputSchema)
			}

			if !jsonEqual(got.OutputSchema, tt.want.OutputSchema) {
				t.Errorf("ParseProgram().OutputSchema = %s, want %s", got.OutputSchema, tt.want.OutputSchema)
			}

			if len(got.Imports) != len(tt.want.Imports) {
				t.Errorf("ParseProgram().Imports length = %d, want %d", len(got.Imports), len(tt.want.Imports))
			} else {
				for i, imp := range got.Imports {
					if imp != tt.want.Imports[i] {
						t.Errorf("ParseProgram().Imports[%d] = %q, want %q", i, imp, tt.want.Imports[i])
					}
				}
			}

			if len(got.MCPServers) != len(tt.want.MCPServers) {
				t.Errorf("ParseProgram().MCPServers length = %d, want %d", len(got.MCPServers), len(tt.want.MCPServers))
			} else {
				for i, srv := range got.MCPServers {
					if srv.Name != tt.want.MCPServers[i].Name {
						t.Errorf("ParseProgram().MCPServers[%d].Name = %q, want %q", i, srv.Name, tt.want.MCPServers[i].Name)
					}
					if srv.Command != tt.want.MCPServers[i].Command {
						t.Errorf("ParseProgram().MCPServers[%d].Command = %q, want %q", i, srv.Command, tt.want.MCPServers[i].Command)
					}
				}
			}
		})
	}
}

func TestValidateSchema(t *testing.T) {
	tests := []struct {
		name        string
		schema      string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid string schema",
			schema:  `{"type":"string"}`,
			wantErr: false,
		},
		{
			name:    "valid object schema",
			schema:  `{"type":"object","properties":{"name":{"type":"string"}}}`,
			wantErr: false,
		},
		{
			name:    "valid array schema",
			schema:  `{"type":"array","items":{"type":"string"}}`,
			wantErr: false,
		},
		{
			name:    "valid with schema version",
			schema:  `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"string"}`,
			wantErr: false,
		},
		{
			name:        "invalid schema version",
			schema:      `{"$schema":"http://wrong-version","type":"string"}`,
			wantErr:     true,
			errContains: "unsupported schema version",
		},
		{
			name:        "invalid type",
			schema:      `{"type":"not_a_real_type"}`,
			wantErr:     true,
			errContains: "invalid type",
		},
		{
			name:        "no type or ref",
			schema:      `{"description":"useless"}`,
			wantErr:     true,
			errContains: "no 'type' or '$ref'",
		},
		{
			name:        "invalid JSON",
			schema:      `{not valid json}`,
			wantErr:     true,
			errContains: "not valid JSON",
		},
		{
			name:        "empty schema",
			schema:      ``,
			wantErr:     true,
			errContains: "empty schema",
		},
		{
			name:    "valid $ref",
			schema:  `{"$ref":"#/definitions/thing"}`,
			wantErr: false,
		},
		{
			name:    "nested valid schemas",
			schema:  `{"type":"object","properties":{"user":{"type":"object","properties":{"name":{"type":"string"}}}}}`,
			wantErr: false,
		},
		{
			name:        "nested invalid schema",
			schema:      `{"type":"object","properties":{"user":{"type":"fake_type"}}}`,
			wantErr:     true,
			errContains: "invalid type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSchema(json.RawMessage(tt.schema))
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSchema() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
				t.Errorf("ValidateSchema() error = %q, want containing %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestGetSchemaType(t *testing.T) {
	tests := []struct {
		name    string
		schema  string
		want    string
		wantErr bool
	}{
		{
			name:    "string type",
			schema:  `{"type":"string"}`,
			want:    "string",
			wantErr: false,
		},
		{
			name:    "object type",
			schema:  `{"type":"object"}`,
			want:    "object",
			wantErr: false,
		},
		{
			name:    "no type field",
			schema:  `{"$ref":"#/definitions/foo"}`,
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			schema:  `{invalid}`,
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSchemaType(json.RawMessage(tt.schema))
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSchemaType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetSchemaType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsObjectSchema(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		want   bool
	}{
		{
			name:   "object schema",
			schema: `{"type":"object"}`,
			want:   true,
		},
		{
			name:   "string schema",
			schema: `{"type":"string"}`,
			want:   false,
		},
		{
			name:   "invalid schema",
			schema: `{invalid}`,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsObjectSchema(json.RawMessage(tt.schema)); got != tt.want {
				t.Errorf("IsObjectSchema() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsArraySchema(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		want   bool
	}{
		{
			name:   "array schema",
			schema: `{"type":"array"}`,
			want:   true,
		},
		{
			name:   "string schema",
			schema: `{"type":"string"}`,
			want:   false,
		},
		{
			name:   "invalid schema",
			schema: `{invalid}`,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsArraySchema(json.RawMessage(tt.schema)); got != tt.want {
				t.Errorf("IsArraySchema() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to compare JSON raw messages
func jsonEqual(a, b json.RawMessage) bool {
	var ja, jb interface{}
	if err := json.Unmarshal(a, &ja); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &jb); err != nil {
		return false
	}
	return jsonEqualReflect(ja, jb)
}

func jsonEqualReflect(a, b interface{}) bool {
	switch va := a.(type) {
	case map[string]interface{}:
		vb, ok := b.(map[string]interface{})
		if !ok || len(va) != len(vb) {
			return false
		}
		for key := range va {
			if !jsonEqualReflect(va[key], vb[key]) {
				return false
			}
		}
		return true
	case []interface{}:
		vb, ok := b.([]interface{})
		if !ok || len(va) != len(vb) {
			return false
		}
		for i := range va {
			if !jsonEqualReflect(va[i], vb[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

// Helper function to create JSON raw message
func jsonSchema(s string) json.RawMessage {
	return json.RawMessage(s)
}
