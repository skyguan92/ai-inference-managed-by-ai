package cli

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatOutput(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		format   OutputFormat
		contains string
	}{
		{
			name:     "json format",
			data:     map[string]string{"key": "value"},
			format:   OutputJSON,
			contains: `"key"`,
		},
		{
			name:     "yaml format",
			data:     map[string]string{"key": "value"},
			format:   OutputYAML,
			contains: "key: value",
		},
		{
			name:     "table format with map",
			data:     map[string]string{"name": "test", "value": "123"},
			format:   OutputTable,
			contains: "name",
		},
		{
			name:     "table format with nil",
			data:     nil,
			format:   OutputTable,
			contains: "",
		},
		{
			name:     "unknown format defaults to table",
			data:     map[string]string{"key": "value"},
			format:   OutputFormat("unknown"),
			contains: "key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := FormatOutput(tt.data, tt.format)
			assert.NoError(t, err)
			assert.Contains(t, output, tt.contains)
		})
	}
}

func TestFormatJSON(t *testing.T) {
	data := map[string]any{
		"name":  "test",
		"value": 123,
	}

	output, err := formatJSON(data)
	assert.NoError(t, err)
	assert.Contains(t, output, `"name"`)
	assert.Contains(t, output, `"test"`)
	assert.Contains(t, output, `"value"`)
	assert.Contains(t, output, `123`)
}

func TestFormatYAML(t *testing.T) {
	data := map[string]any{
		"name":  "test",
		"value": 123,
	}

	output, err := formatYAML(data)
	assert.NoError(t, err)
	assert.Contains(t, output, "name: test")
	assert.Contains(t, output, "value: 123")
}

func TestFormatTable(t *testing.T) {
	t.Run("slice of maps", func(t *testing.T) {
		type testItem struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}
		data := []testItem{
			{Name: "item1", Count: 10},
			{Name: "item2", Count: 20},
		}

		output, err := formatTable(data)
		assert.NoError(t, err)
		assert.Contains(t, output, "name")
		assert.Contains(t, output, "item1")
		assert.Contains(t, output, "item2")
	})

	t.Run("single map", func(t *testing.T) {
		data := map[string]any{
			"key1": "value1",
			"key2": "value2",
		}

		output, err := formatTable(data)
		assert.NoError(t, err)
		assert.Contains(t, output, "key1")
		assert.Contains(t, output, "value1")
	})

	t.Run("nil data", func(t *testing.T) {
		output, err := formatTable(nil)
		assert.NoError(t, err)
		assert.Empty(t, output)
	})

	t.Run("empty slice", func(t *testing.T) {
		data := []map[string]string{}
		output, err := formatTable(data)
		assert.NoError(t, err)
		assert.Contains(t, output, "No items")
	})

	t.Run("primitive value", func(t *testing.T) {
		output, err := formatTable("simple string")
		assert.NoError(t, err)
		assert.Contains(t, output, "simple string")
	})

	t.Run("slice of primitives", func(t *testing.T) {
		data := []string{"item1", "item2", "item3"}
		output, err := formatTable(data)
		assert.NoError(t, err)
		assert.Contains(t, output, "value")
	})

	t.Run("struct", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}
		data := TestStruct{Name: "test", Value: 42}

		output, err := formatTable(data)
		assert.NoError(t, err)
		assert.Contains(t, output, "name")
		assert.Contains(t, output, "test")
	})

	t.Run("pointer to nil", func(t *testing.T) {
		var data *map[string]string
		output, err := formatTable(data)
		assert.NoError(t, err)
		assert.Empty(t, output)
	})

	t.Run("pointer to struct", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
		}
		data := &TestStruct{Name: "test"}

		output, err := formatTable(data)
		assert.NoError(t, err)
		assert.Contains(t, output, "name")
	})
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"string", "hello", "hello"},
		{"int", 42, "42"},
		{"float", 3.14, "3.14"},
		{"bool", true, "true"},
		{"nil", nil, ""},
		{"slice", []string{"a", "b"}, `["a","b"]`},
		{"map", map[string]string{"k": "v"}, `{"k":"v"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatValue_Pointer(t *testing.T) {
	val := "hello"
	ptr := &val
	result := formatValue(ptr)
	assert.Equal(t, "hello", result)

	var nilPtr *string
	result = formatValue(nilPtr)
	assert.Empty(t, result)
}

func TestPrintOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := &OutputOptions{
		Format: OutputJSON,
		Quiet:  false,
		Writer: buf,
	}

	data := map[string]string{"test": "value"}
	err := PrintOutput(data, opts)
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "test")
}

func TestPrintOutputQuiet(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := &OutputOptions{
		Format: OutputJSON,
		Quiet:  true,
		Writer: buf,
	}

	data := map[string]string{"test": "value"}
	err := PrintOutput(data, opts)
	assert.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestPrintError(t *testing.T) {
	t.Run("json format", func(t *testing.T) {
		opts := &OutputOptions{
			Format: OutputJSON,
		}
		PrintError(errors.New("test error"), opts)
	})

	t.Run("yaml format", func(t *testing.T) {
		opts := &OutputOptions{
			Format: OutputYAML,
		}
		PrintError(errors.New("test error"), opts)
	})

	t.Run("table format", func(t *testing.T) {
		opts := &OutputOptions{
			Format: OutputTable,
		}
		PrintError(errors.New("test error"), opts)
	})
}

func TestPrintSuccess(t *testing.T) {
	t.Run("json format", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := &OutputOptions{
			Format: OutputJSON,
			Quiet:  false,
			Writer: buf,
		}

		PrintSuccess("operation completed", opts)
		output := buf.String()
		assert.Contains(t, output, "success")
		assert.Contains(t, output, "operation completed")
	})

	t.Run("yaml format", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := &OutputOptions{
			Format: OutputYAML,
			Quiet:  false,
			Writer: buf,
		}

		PrintSuccess("operation completed", opts)
		assert.Contains(t, buf.String(), "operation completed")
	})

	t.Run("table format", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := &OutputOptions{
			Format: OutputTable,
			Quiet:  false,
			Writer: buf,
		}

		PrintSuccess("operation completed", opts)
		assert.Contains(t, buf.String(), "operation completed")
	})

	t.Run("quiet mode", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := &OutputOptions{
			Format: OutputTable,
			Quiet:  true,
			Writer: buf,
		}

		PrintSuccess("operation completed", opts)
		assert.Empty(t, buf.String())
	})
}

func TestNewOutputOptions(t *testing.T) {
	opts := NewOutputOptions()
	assert.Equal(t, OutputTable, opts.Format)
	assert.False(t, opts.Quiet)
	assert.NotNil(t, opts.Writer)
}

func TestMakeSeparators(t *testing.T) {
	seps := makeSeparators(3)
	assert.Len(t, seps, 3)
	for _, sep := range seps {
		assert.Contains(t, sep, "-")
	}
}

func TestGetFieldValues(t *testing.T) {
	t.Run("map input", func(t *testing.T) {
		data := map[string]any{"name": "test", "value": 123}
		fields := []string{"name", "value"}
		values := getFieldValues(data, fields)

		assert.Equal(t, "test", values[0])
		assert.Equal(t, "123", values[1])
	})

	t.Run("non-struct non-map", func(t *testing.T) {
		data := "simple string"
		fields := []string{"value"}
		values := getFieldValues(data, fields)

		assert.Equal(t, "simple string", values[0])
	})
}

func TestGetFields(t *testing.T) {
	t.Run("struct with json tags", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}
		data := TestStruct{Name: "test", Value: 42}

		fields := getFields(data)
		assert.Contains(t, fields, "name")
		assert.Contains(t, fields, "value")
	})

	t.Run("struct without json tags", func(t *testing.T) {
		type TestStruct struct {
			Name  string
			Value int
		}
		data := TestStruct{Name: "test", Value: 42}

		fields := getFields(data)
		assert.Contains(t, fields, "Name")
		assert.Contains(t, fields, "Value")
	})

	t.Run("non-struct", func(t *testing.T) {
		fields := getFields("simple string")
		assert.Len(t, fields, 1)
		assert.Equal(t, "value", fields[0])
	})

	t.Run("pointer to struct", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
		}
		data := &TestStruct{Name: "test"}

		fields := getFields(data)
		assert.Contains(t, fields, "name")
	})

	t.Run("struct with json omitempty", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name,omitempty"`
			Value int    `json:"value,omitempty"`
		}
		data := TestStruct{Name: "test", Value: 42}

		fields := getFields(data)
		assert.Contains(t, fields, "name")
		assert.Contains(t, fields, "value")
	})

	t.Run("struct with unexported fields", func(t *testing.T) {
		type TestStruct struct {
			Name  string
			value int
		}
		data := TestStruct{Name: "test", value: 42}

		fields := getFields(data)
		assert.Contains(t, fields, "Name")
		assert.Len(t, fields, 1)
	})
}

func TestFormatJSON_Error(t *testing.T) {
	ch := make(chan int)
	_, err := formatJSON(ch)
	assert.Error(t, err)
}

func TestFormatTable_Array(t *testing.T) {
	data := [3]string{"item1", "item2", "item3"}
	output, err := formatTable(data)
	assert.NoError(t, err)
	assert.Contains(t, output, "value")
}

func TestFormatSliceTable_EmptySlice(t *testing.T) {
	data := []int{}
	output, err := formatSliceTable(data)
	assert.NoError(t, err)
	assert.Contains(t, output, "No items")
}

func TestFormatMapTable_NonMap(t *testing.T) {
	output, err := formatMapTable("not a map")
	assert.NoError(t, err)
	assert.Contains(t, output, "not a map")
}

func TestGetFieldValues_Struct(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	data := TestStruct{Name: "test", Value: 42}
	fields := []string{"name", "value"}
	values := getFieldValues(data, fields)

	assert.Equal(t, "test", values[0])
	assert.Equal(t, "42", values[1])
}

func TestGetFieldValues_MissingField(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
	}
	data := TestStruct{Name: "test"}
	fields := []string{"name", "missing"}
	values := getFieldValues(data, fields)

	assert.Equal(t, "test", values[0])
	assert.Equal(t, "", values[1])
}

func TestGetFieldValues_MapWithMissingKey(t *testing.T) {
	data := map[string]any{"name": "test"}
	fields := []string{"name", "missing"}
	values := getFieldValues(data, fields)

	assert.Equal(t, "test", values[0])
	assert.Equal(t, "", values[1])
}

func TestFormatValue_Float32(t *testing.T) {
	result := formatValue(float32(3.14159))
	assert.Contains(t, result, "3.14")
}

func TestFormatValue_Float64(t *testing.T) {
	result := formatValue(3.14159)
	assert.Contains(t, result, "3.14")
}

func TestFormatValue_IntTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"int8", int8(8), "8"},
		{"int16", int16(16), "16"},
		{"int32", int32(32), "32"},
		{"int64", int64(64), "64"},
		{"uint", uint(1), "1"},
		{"uint8", uint8(8), "8"},
		{"uint16", uint16(16), "16"},
		{"uint32", uint32(32), "32"},
		{"uint64", uint64(64), "64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatValue_ComplexType(t *testing.T) {
	type ComplexStruct struct {
		Field1 string
		Field2 int
	}
	result := formatValue(ComplexStruct{Field1: "test", Field2: 42})
	assert.Contains(t, result, "Field1")
}

func TestFormatStructTable(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	data := TestStruct{Name: "test", Value: 42}

	output, err := formatStructTable(data)
	assert.NoError(t, err)
	assert.Contains(t, output, "name")
	assert.Contains(t, output, "test")
}
