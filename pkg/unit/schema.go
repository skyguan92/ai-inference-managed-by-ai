package unit

import (
	"fmt"
	"reflect"
	"regexp"
)

func (s *Schema) Validate(input any) error {
	if input == nil {
		return fmt.Errorf("input is nil")
	}

	switch s.Type {
	case "string":
		return s.validateString(input)
	case "number":
		return s.validateNumber(input)
	case "boolean":
		return s.validateBoolean(input)
	case "array":
		return s.validateArray(input)
	case "object":
		return s.validateObject(input)
	default:
		return fmt.Errorf("unknown schema type: %s", s.Type)
	}
}

func (s *Schema) validateString(input any) error {
	str, ok := input.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", input)
	}

	if s.MinLength != nil && len(str) < *s.MinLength {
		return fmt.Errorf("string length %d is less than minimum %d", len(str), *s.MinLength)
	}

	if s.MaxLength != nil && len(str) > *s.MaxLength {
		return fmt.Errorf("string length %d exceeds maximum %d", len(str), *s.MaxLength)
	}

	if s.Pattern != "" {
		matched, err := regexp.MatchString(s.Pattern, str)
		if err != nil {
			return fmt.Errorf("invalid pattern %q: %w", s.Pattern, err)
		}
		if !matched {
			return fmt.Errorf("string %q does not match pattern %q", str, s.Pattern)
		}
	}

	if len(s.Enum) > 0 {
		return s.validateEnum(input)
	}

	return nil
}

func (s *Schema) validateNumber(input any) error {
	var value float64
	switch v := input.(type) {
	case int:
		value = float64(v)
	case int8:
		value = float64(v)
	case int16:
		value = float64(v)
	case int32:
		value = float64(v)
	case int64:
		value = float64(v)
	case uint:
		value = float64(v)
	case uint8:
		value = float64(v)
	case uint16:
		value = float64(v)
	case uint32:
		value = float64(v)
	case uint64:
		value = float64(v)
	case float32:
		value = float64(v)
	case float64:
		value = v
	default:
		return fmt.Errorf("expected number, got %T", input)
	}

	if s.Min != nil && value < *s.Min {
		return fmt.Errorf("value %v is less than minimum %v", value, *s.Min)
	}

	if s.Max != nil && value > *s.Max {
		return fmt.Errorf("value %v exceeds maximum %v", value, *s.Max)
	}

	if len(s.Enum) > 0 {
		return s.validateEnum(input)
	}

	return nil
}

func (s *Schema) validateBoolean(input any) error {
	if _, ok := input.(bool); !ok {
		return fmt.Errorf("expected boolean, got %T", input)
	}
	return nil
}

func (s *Schema) validateArray(input any) error {
	val := reflect.ValueOf(input)
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		return fmt.Errorf("expected array, got %T", input)
	}

	if s.Items == nil {
		return nil
	}

	for i := 0; i < val.Len(); i++ {
		item := val.Index(i).Interface()
		if err := s.Items.Validate(item); err != nil {
			return fmt.Errorf("array item %d: %w", i, err)
		}
	}

	return nil
}

func (s *Schema) validateObject(input any) error {
	obj, ok := input.(map[string]any)
	if !ok {
		val := reflect.ValueOf(input)
		if val.Kind() == reflect.Map && val.Type().Key().Kind() == reflect.String {
			obj = make(map[string]any)
			iter := val.MapRange()
			for iter.Next() {
				obj[iter.Key().String()] = iter.Value().Interface()
			}
		} else {
			return fmt.Errorf("expected object, got %T", input)
		}
	}

	for _, req := range s.Required {
		if _, exists := obj[req]; !exists {
			return fmt.Errorf("required field %q is missing", req)
		}
	}

	for name, field := range s.Properties {
		value, exists := obj[name]
		if !exists {
			continue
		}
		if err := field.Schema.Validate(value); err != nil {
			return fmt.Errorf("field %q: %w", name, err)
		}
	}

	return nil
}

func (s *Schema) validateEnum(input any) error {
	for _, enumValue := range s.Enum {
		if reflect.DeepEqual(input, enumValue) {
			return nil
		}
	}
	return fmt.Errorf("value %v is not one of allowed values %v", input, s.Enum)
}

func StringSchema() *Schema {
	return &Schema{Type: "string"}
}

func NumberSchema() *Schema {
	return &Schema{Type: "number"}
}

func BooleanSchema() *Schema {
	return &Schema{Type: "boolean"}
}

func ArraySchema(items *Schema) *Schema {
	return &Schema{
		Type:  "array",
		Items: items,
	}
}

func ObjectSchema(properties map[string]Field, required []string) *Schema {
	return &Schema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}
}

func NewField(name string, schema *Schema) Field {
	return Field{
		Name:   name,
		Schema: *schema,
	}
}
