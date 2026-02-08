package parser

import (
	"encoding/json"
	"errors"
	"fmt"
)

// SchemaVersion represents the JSON Schema version being used.
const SchemaVersion = "https://json-schema.org/draft/2020-12/schema"

// JSONSchema represents a JSON Schema document structure.
type JSONSchema map[string]interface{}

// ValidateSchema checks if a JSON raw message conforms to JSON Schema Draft 2020-12.
func ValidateSchema(raw json.RawMessage) error {
	if len(raw) == 0 {
		return errors.New("empty schema: nothing to validate")
	}

	var schema JSONSchema
	if err := json.Unmarshal(raw, &schema); err != nil {
		return fmt.Errorf("schema is not valid JSON: %w", err)
	}

	// Check for $schema keyword
	if schemaVersion, ok := schema["$schema"].(string); ok {
		if schemaVersion != SchemaVersion {
			return fmt.Errorf("unsupported schema version: %s (we only support %s because standards matter)", schemaVersion, SchemaVersion)
		}
	}

	// Basic validation: must have at least a type or $ref
	if _, hasType := schema["type"]; !hasType {
		if _, hasRef := schema["$ref"]; !hasRef {
			return errors.New("schema has no 'type' or '$ref' field: what are you even describing?")
		}
	}

	// Validate type field if present
	if typeVal, ok := schema["type"].(string); ok {
		validTypes := map[string]bool{
			"null":    true,
			"boolean": true,
			"object":  true,
			"array":   true,
			"number":  true,
			"string":  true,
			"integer": true,
		}

		if !validTypes[typeVal] {
			return fmt.Errorf("invalid type '%s': that's not a real JSON Schema type and you know it", typeVal)
		}
	}

	// Validate object properties if present
	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		for propName, propSchema := range properties {
			if propMap, ok := propSchema.(map[string]interface{}); ok {
				// Recursively validate nested schemas
				propRaw, err := json.Marshal(propMap)
				if err != nil {
					return fmt.Errorf("property '%s' schema won't marshal: %w", propName, err)
				}
				if err := ValidateSchema(propRaw); err != nil {
					return fmt.Errorf("property '%s' is invalid: %w", propName, err)
				}
			}
		}
	}

	// Validate array items if present
	if items, ok := schema["items"]; ok {
		itemsRaw, err := json.Marshal(items)
		if err != nil {
			return fmt.Errorf("items schema won't marshal: %w", err)
		}
		if err := ValidateSchema(itemsRaw); err != nil {
			return fmt.Errorf("items schema is invalid: %w", err)
		}
	}

	return nil
}

// GetSchemaType extracts the type from a JSON schema.
func GetSchemaType(raw json.RawMessage) (string, error) {
	var schema JSONSchema
	if err := json.Unmarshal(raw, &schema); err != nil {
		return "", fmt.Errorf("failed to parse schema: %w", err)
	}

	if typeVal, ok := schema["type"].(string); ok {
		return typeVal, nil
	}

	return "", errors.New("schema has no type field")
}

// IsObjectSchema checks if a schema describes an object type.
func IsObjectSchema(raw json.RawMessage) bool {
	typ, err := GetSchemaType(raw)
	if err != nil {
		return false
	}
	return typ == "object"
}

// IsArraySchema checks if a schema describes an array type.
func IsArraySchema(raw json.RawMessage) bool {
	typ, err := GetSchemaType(raw)
	if err != nil {
		return false
	}
	return typ == "array"
}
