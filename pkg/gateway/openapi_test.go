package gateway

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestGenerateOpenAPI(t *testing.T) {
	t.Run("empty registry", func(t *testing.T) {
		registry := unit.NewRegistry()
		spec := GenerateOpenAPI(registry)

		var openapi map[string]any
		if err := json.Unmarshal(spec, &openapi); err != nil {
			t.Fatalf("failed to parse openapi spec: %v", err)
		}

		if openapi["openapi"] != OpenAPIVersion {
			t.Errorf("expected openapi version %s, got %v", OpenAPIVersion, openapi["openapi"])
		}

		info, ok := openapi["info"].(map[string]any)
		if !ok {
			t.Fatal("expected info object")
		}
		if info["title"] != APITitle {
			t.Errorf("expected title %s, got %v", APITitle, info["title"])
		}
		if info["version"] != APIVersion {
			t.Errorf("expected version %s, got %v", APIVersion, info["version"])
		}
	})

	t.Run("with commands and queries", func(t *testing.T) {
		registry := unit.NewRegistry()

		cmd := &mockOpenAPIUnit{
			name:        "model.pull",
			domain:      "model",
			description: "Pull a model from registry",
			inputSchema: unit.Schema{
				Type: "object",
				Properties: map[string]unit.Field{
					"source": {Name: "source", Schema: unit.Schema{Type: "string"}},
					"repo":   {Name: "repo", Schema: unit.Schema{Type: "string"}},
				},
				Required: []string{"source", "repo"},
			},
			outputSchema: unit.Schema{
				Type: "object",
				Properties: map[string]unit.Field{
					"model_id": {Name: "model_id", Schema: unit.Schema{Type: "string"}},
				},
			},
		}

		query := &mockOpenAPIUnit{
			name:        "model.list",
			domain:      "model",
			description: "List all models",
			inputSchema: unit.Schema{Type: "object"},
			outputSchema: unit.Schema{
				Type:  "array",
				Items: &unit.Schema{Type: "object"},
			},
		}

		_ = registry.RegisterCommand(cmd)
		_ = registry.RegisterQuery(query)

		spec := GenerateOpenAPI(registry)

		var openapi OpenAPISpec
		if err := json.Unmarshal(spec, &openapi); err != nil {
			t.Fatalf("failed to parse openapi spec: %v", err)
		}

		if _, exists := openapi.Paths["/api/v2/execute"]; !exists {
			t.Error("expected /api/v2/execute path")
		}
		if _, exists := openapi.Paths["/health"]; !exists {
			t.Error("expected /health path")
		}
	})

	t.Run("includes common schemas", func(t *testing.T) {
		registry := unit.NewRegistry()
		spec := GenerateOpenAPI(registry)

		var openapi OpenAPISpec
		if err := json.Unmarshal(spec, &openapi); err != nil {
			t.Fatalf("failed to parse openapi spec: %v", err)
		}

		expectedSchemas := []string{
			"ExecuteRequest",
			"RequestOptions",
			"ExecuteResponse",
			"ErrorInfo",
			"ResponseMeta",
			"Pagination",
			"ErrorResponse",
		}

		for _, name := range expectedSchemas {
			if _, exists := openapi.Components.Schemas[name]; !exists {
				t.Errorf("expected schema %s", name)
			}
		}
	})
}

func TestUnitToPath(t *testing.T) {
	tests := []struct {
		domain   string
		name     string
		expected string
	}{
		{"model", "model.pull", "/api/v2/model/pull"},
		{"inference", "inference.chat", "/api/v2/inference/chat"},
		{"device", "device.detect", "/api/v2/device/detect"},
		{"engine", "engine.start", "/api/v2/engine/start"},
	}

	for _, tt := range tests {
		t.Run(tt.domain+"."+tt.name, func(t *testing.T) {
			result := unitToPath(tt.domain, tt.name)
			if result != tt.expected {
				t.Errorf("unitToPath(%s, %s) = %s, expected %s", tt.domain, tt.name, result, tt.expected)
			}
		})
	}
}

func TestHTTPMethodForCommand(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"model.create", "post"},
		{"model.pull", "post"},
		{"model.list", "get"},
		{"model.get", "get"},
		{"model.delete", "post"},
		{"engine.start", "post"},
		{"engine.stop", "post"},
		{"inference.chat", "post"},
		{"inference.embed", "post"},
		{"device.detect", "get"},
		{"device.info", "get"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := httpMethodForCommand(tt.name)
			if result != tt.expected {
				t.Errorf("httpMethodForCommand(%s) = %s, expected %s", tt.name, result, tt.expected)
			}
		})
	}
}

func TestSanitizeOperationID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"model.pull", "model_pull"},
		{"inference.chat", "inference_chat"},
		{"simple", "simple"},
		{"multi.dot.name", "multi_dot_name"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeOperationID(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeOperationID(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSchemaToOpenAPI(t *testing.T) {
	t.Run("basic schema", func(t *testing.T) {
		s := unit.Schema{
			Type:        "object",
			Description: "Test schema",
			Required:    []string{"name"},
		}

		result := schemaToOpenAPI(s)

		if result.Type != "object" {
			t.Errorf("expected type object, got %s", result.Type)
		}
		if result.Description != "Test schema" {
			t.Errorf("expected description, got %s", result.Description)
		}
		if len(result.Required) != 1 || result.Required[0] != "name" {
			t.Errorf("expected required [name], got %v", result.Required)
		}
	})

	t.Run("schema with properties", func(t *testing.T) {
		s := unit.Schema{
			Type: "object",
			Properties: map[string]unit.Field{
				"name": {
					Name: "name",
					Schema: unit.Schema{
						Type:        "string",
						Description: "The name",
					},
				},
				"count": {
					Name: "count",
					Schema: unit.Schema{
						Type: "integer",
					},
				},
			},
		}

		result := schemaToOpenAPI(s)

		if len(result.Properties) != 2 {
			t.Errorf("expected 2 properties, got %d", len(result.Properties))
		}
		if result.Properties["name"].Type != "string" {
			t.Error("expected name property to be string")
		}
	})

	t.Run("array schema", func(t *testing.T) {
		s := unit.Schema{
			Type:  "array",
			Items: &unit.Schema{Type: "string"},
		}

		result := schemaToOpenAPI(s)

		if result.Type != "array" {
			t.Errorf("expected type array, got %s", result.Type)
		}
		if result.Items == nil || result.Items.Type != "string" {
			t.Error("expected items type string")
		}
	})

	t.Run("schema with enum", func(t *testing.T) {
		s := unit.Schema{
			Type: "string",
			Enum: []any{"one", "two", "three"},
		}

		result := schemaToOpenAPI(s)

		if len(result.Enum) != 3 {
			t.Errorf("expected 3 enum values, got %d", len(result.Enum))
		}
	})
}

func TestIndexOf(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected int
	}{
		{"model.pull", ".", 5},
		{"noseparator", ".", -1},
		{"a.b.c", ".", 1},
		{"", ".", -1},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			result := indexOf(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("indexOf(%s, %s) = %d, expected %d", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

type mockOpenAPIUnit struct {
	name         string
	domain       string
	description  string
	inputSchema  unit.Schema
	outputSchema unit.Schema
}

func (m *mockOpenAPIUnit) Name() string              { return m.name }
func (m *mockOpenAPIUnit) Domain() string            { return m.domain }
func (m *mockOpenAPIUnit) Description() string       { return m.description }
func (m *mockOpenAPIUnit) InputSchema() unit.Schema  { return m.inputSchema }
func (m *mockOpenAPIUnit) OutputSchema() unit.Schema { return m.outputSchema }
func (m *mockOpenAPIUnit) Examples() []unit.Example  { return nil }
func (m *mockOpenAPIUnit) Execute(ctx context.Context, input any) (any, error) {
	return map[string]any{"success": true}, nil
}
