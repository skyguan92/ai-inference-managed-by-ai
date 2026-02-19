package unit

import (
	"testing"
)

func TestSchema_Validate_UnknownType(t *testing.T) {
	schema := &Schema{Type: "unknown_type"}
	err := schema.Validate("some value")
	if err == nil {
		t.Error("expected error for unknown schema type")
	}
}

func TestSchema_Validate_NilInput(t *testing.T) {
	schemas := []*Schema{
		StringSchema(),
		NumberSchema(),
		BooleanSchema(),
		ArraySchema(StringSchema()),
		ObjectSchema(nil, nil),
	}

	for _, s := range schemas {
		err := s.Validate(nil)
		if err == nil {
			t.Errorf("expected error for nil input with %q schema", s.Type)
		}
	}
}

func TestSchema_Validate_ObjectWithReflectMap(t *testing.T) {
	// Test validateObject with a non-map[string]any that is a string-keyed map
	schema := ObjectSchema(map[string]Field{
		"name": {Name: "name", Schema: *StringSchema()},
	}, []string{"name"})

	// This uses reflect path since map[string]string is not map[string]any
	input := map[string]string{
		"name": "test",
	}
	err := schema.Validate(input)
	if err != nil {
		t.Errorf("expected no error for map[string]string input, got %v", err)
	}
}

func TestSchema_Validate_ObjectWithReflectMap_MissingRequired(t *testing.T) {
	schema := ObjectSchema(map[string]Field{
		"name": {Name: "name", Schema: *StringSchema()},
	}, []string{"name"})

	// This uses reflect path since map[string]string is not map[string]any
	input := map[string]string{
		"other": "value",
	}
	err := schema.Validate(input)
	if err == nil {
		t.Error("expected error for missing required field via reflect path")
	}
}

func TestSchema_Validate_ObjectWithNonMapInput(t *testing.T) {
	schema := ObjectSchema(nil, nil)
	// A struct is not a map, should fail
	err := schema.Validate(struct{ Name string }{Name: "test"})
	if err == nil {
		t.Error("expected error for struct input (not a map)")
	}
}

func TestSchema_Validate_ArrayWithItems(t *testing.T) {
	// Array without items schema
	schema := &Schema{Type: "array"}
	err := schema.Validate([]any{1, "two", true})
	if err != nil {
		t.Errorf("expected no error for array without items schema, got %v", err)
	}
}

func TestSchema_Validate_NumberAllIntegerTypes(t *testing.T) {
	schema := NumberSchema()
	min := float64(0)
	max := float64(1000)
	schema.Min = &min
	schema.Max = &max

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{"int8", int8(50), false},
		{"int16", int16(100), false},
		{"int32", int32(200), false},
		{"int64", int64(300), false},
		{"uint", uint(400), false},
		{"uint8", uint8(50), false},
		{"uint16", uint16(60), false},
		{"uint32", uint32(70), false},
		{"uint64", uint64(80), false},
		{"float32", float32(3.14), false},
		{"float64", float64(99.99), false},
		{"invalid string", "42", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := schema.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestSchema_Validate_NumberEnumValues(t *testing.T) {
	schema := NumberSchema()
	schema.Enum = []any{1, 2, 3}

	err := schema.Validate(1)
	if err != nil {
		t.Errorf("expected no error for enum value, got %v", err)
	}

	err = schema.Validate(4)
	if err == nil {
		t.Error("expected error for non-enum value")
	}
}

func TestSchema_Validate_InvalidPattern(t *testing.T) {
	schema := StringSchema()
	schema.Pattern = `[invalid`

	err := schema.Validate("hello")
	if err == nil {
		t.Error("expected error for invalid regex pattern")
	}
}

func TestStreamChunk(t *testing.T) {
	chunk := StreamChunk{
		Type:     "content",
		Data:     "Hello, world!",
		Metadata: map[string]any{"finish_reason": "stop"},
	}

	if chunk.Type != "content" {
		t.Errorf("Type = %q, want 'content'", chunk.Type)
	}
	if chunk.Data != "Hello, world!" {
		t.Errorf("Data = %v, want 'Hello, world!'", chunk.Data)
	}
	if chunk.Metadata == nil {
		t.Error("Metadata should not be nil")
	}
}

func TestStreamChunk_ErrorType(t *testing.T) {
	chunk := StreamChunk{
		Type: "error",
		Data: "something went wrong",
	}
	if chunk.Type != "error" {
		t.Errorf("Type = %q, want 'error'", chunk.Type)
	}
}

func TestStreamChunk_DoneType(t *testing.T) {
	chunk := StreamChunk{
		Type:     "done",
		Metadata: map[string]any{"total_tokens": 100},
	}
	if chunk.Type != "done" {
		t.Errorf("Type = %q, want 'done'", chunk.Type)
	}
}
