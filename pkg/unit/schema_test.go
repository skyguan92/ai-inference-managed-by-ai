package unit

import (
	"testing"
)

func TestSchema_Validate_BasicTypes(t *testing.T) {
	tests := []struct {
		name    string
		schema  *Schema
		input   any
		wantErr bool
	}{
		{
			name:    "string type valid",
			schema:  StringSchema(),
			input:   "hello",
			wantErr: false,
		},
		{
			name:    "string type invalid - number",
			schema:  StringSchema(),
			input:   123,
			wantErr: true,
		},
		{
			name:    "number type valid - int",
			schema:  NumberSchema(),
			input:   42,
			wantErr: false,
		},
		{
			name:    "number type valid - float",
			schema:  NumberSchema(),
			input:   3.14,
			wantErr: false,
		},
		{
			name:    "number type invalid - string",
			schema:  NumberSchema(),
			input:   "42",
			wantErr: true,
		},
		{
			name:    "boolean type valid - true",
			schema:  BooleanSchema(),
			input:   true,
			wantErr: false,
		},
		{
			name:    "boolean type valid - false",
			schema:  BooleanSchema(),
			input:   false,
			wantErr: false,
		},
		{
			name:    "boolean type invalid",
			schema:  BooleanSchema(),
			input:   "true",
			wantErr: true,
		},
		{
			name:    "null input for string",
			schema:  StringSchema(),
			input:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchema_Validate_Required(t *testing.T) {
	schema := ObjectSchema(map[string]Field{
		"name":  {Name: "name", Schema: *StringSchema()},
		"email": {Name: "email", Schema: *StringSchema()},
	}, []string{"name"})

	tests := []struct {
		name    string
		input   any
		wantErr bool
		errMsg  string
	}{
		{
			name: "all required present",
			input: map[string]any{
				"name":  "John",
				"email": "john@example.com",
			},
			wantErr: false,
		},
		{
			name: "only required present",
			input: map[string]any{
				"name": "John",
			},
			wantErr: false,
		},
		{
			name:    "missing required field",
			input:   map[string]any{"email": "john@example.com"},
			wantErr: true,
			errMsg:  "required field \"name\" is missing",
		},
		{
			name:    "empty object",
			input:   map[string]any{},
			wantErr: true,
		},
		{
			name:    "not an object",
			input:   "not an object",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := schema.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.errMsg != "" && err != nil && err.Error() != tt.errMsg {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.errMsg)
			}
		})
	}
}

func TestSchema_Validate_Enum(t *testing.T) {
	schema := StringSchema()
	schema.Enum = []any{"ollama", "huggingface", "modelscope"}

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name:    "valid enum value",
			input:   "ollama",
			wantErr: false,
		},
		{
			name:    "invalid enum value",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "wrong type for enum",
			input:   123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := schema.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchema_Validate_MinMax(t *testing.T) {
	schema := NumberSchema()
	min := float64(0)
	max := float64(100)
	schema.Min = &min
	schema.Max = &max

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name:    "within range",
			input:   50,
			wantErr: false,
		},
		{
			name:    "at min",
			input:   0,
			wantErr: false,
		},
		{
			name:    "at max",
			input:   100,
			wantErr: false,
		},
		{
			name:    "below min",
			input:   -1,
			wantErr: true,
		},
		{
			name:    "above max",
			input:   101,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := schema.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchema_Validate_MinLengthMaxLength(t *testing.T) {
	schema := StringSchema()
	minLen := 3
	maxLen := 10
	schema.MinLength = &minLen
	schema.MaxLength = &maxLen

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name:    "valid length",
			input:   "hello",
			wantErr: false,
		},
		{
			name:    "at min length",
			input:   "abc",
			wantErr: false,
		},
		{
			name:    "at max length",
			input:   "abcdefghij",
			wantErr: false,
		},
		{
			name:    "too short",
			input:   "ab",
			wantErr: true,
		},
		{
			name:    "too long",
			input:   "abcdefghijk",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := schema.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchema_Validate_Array(t *testing.T) {
	itemSchema := NumberSchema()
	schema := ArraySchema(itemSchema)

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name:    "valid array",
			input:   []any{1, 2, 3},
			wantErr: false,
		},
		{
			name:    "empty array",
			input:   []any{},
			wantErr: false,
		},
		{
			name:    "invalid item type",
			input:   []any{1, "two", 3},
			wantErr: true,
		},
		{
			name:    "not an array",
			input:   "not an array",
			wantErr: true,
		},
		{
			name:    "array of ints",
			input:   []int{1, 2, 3},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := schema.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchema_Validate_NestedObject(t *testing.T) {
	addressSchema := ObjectSchema(map[string]Field{
		"city":    {Name: "city", Schema: *StringSchema()},
		"country": {Name: "country", Schema: *StringSchema()},
	}, []string{"city", "country"})

	personSchema := ObjectSchema(map[string]Field{
		"name":    {Name: "name", Schema: *StringSchema()},
		"address": {Name: "address", Schema: *addressSchema},
	}, []string{"name"})

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name: "valid nested object",
			input: map[string]any{
				"name": "John",
				"address": map[string]any{
					"city":    "Beijing",
					"country": "China",
				},
			},
			wantErr: false,
		},
		{
			name: "missing nested required field",
			input: map[string]any{
				"name": "John",
				"address": map[string]any{
					"city": "Beijing",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid nested type",
			input: map[string]any{
				"name":    "John",
				"address": "not an object",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := personSchema.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchemaHelpers(t *testing.T) {
	t.Run("StringSchema", func(t *testing.T) {
		s := StringSchema()
		if s.Type != "string" {
			t.Errorf("StringSchema().Type = %v, want string", s.Type)
		}
	})

	t.Run("NumberSchema", func(t *testing.T) {
		s := NumberSchema()
		if s.Type != "number" {
			t.Errorf("NumberSchema().Type = %v, want number", s.Type)
		}
	})

	t.Run("BooleanSchema", func(t *testing.T) {
		s := BooleanSchema()
		if s.Type != "boolean" {
			t.Errorf("BooleanSchema().Type = %v, want boolean", s.Type)
		}
	})

	t.Run("ArraySchema", func(t *testing.T) {
		items := StringSchema()
		s := ArraySchema(items)
		if s.Type != "array" {
			t.Errorf("ArraySchema().Type = %v, want array", s.Type)
		}
		if s.Items == nil || s.Items.Type != "string" {
			t.Errorf("ArraySchema().Items.Type = %v, want string", s.Items.Type)
		}
	})

	t.Run("ObjectSchema", func(t *testing.T) {
		props := map[string]Field{
			"name": {Name: "name", Schema: *StringSchema()},
		}
		required := []string{"name"}
		s := ObjectSchema(props, required)
		if s.Type != "object" {
			t.Errorf("ObjectSchema().Type = %v, want object", s.Type)
		}
		if len(s.Properties) != 1 {
			t.Errorf("ObjectSchema().Properties length = %v, want 1", len(s.Properties))
		}
		if len(s.Required) != 1 || s.Required[0] != "name" {
			t.Errorf("ObjectSchema().Required = %v, want [name]", s.Required)
		}
	})
}

func TestFieldHelper(t *testing.T) {
	field := NewField("name", StringSchema())
	if field.Name != "name" {
		t.Errorf("NewField().Name = %v, want name", field.Name)
	}
	if field.Schema.Type != "string" {
		t.Errorf("NewField().Schema.Type = %v, want string", field.Schema.Type)
	}
}

func TestSchema_Validate_Pattern(t *testing.T) {
	schema := StringSchema()
	schema.Pattern = `^[a-z]+$`

	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name:    "matches pattern",
			input:   "hello",
			wantErr: false,
		},
		{
			name:    "does not match pattern",
			input:   "Hello123",
			wantErr: true,
		},
		{
			name:    "empty string does not match plus pattern",
			input:   "",
			wantErr: true,
		},
		{
			name:    "single char matches",
			input:   "a",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := schema.Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
