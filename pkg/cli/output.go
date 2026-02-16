package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

type OutputFormat string

const (
	OutputTable OutputFormat = "table"
	OutputJSON  OutputFormat = "json"
	OutputYAML  OutputFormat = "yaml"
)

type OutputOptions struct {
	Format OutputFormat
	Quiet  bool
	Writer io.Writer
}

func NewOutputOptions() *OutputOptions {
	return &OutputOptions{
		Format: OutputTable,
		Quiet:  false,
		Writer: os.Stdout,
	}
}

func FormatOutput(data any, format OutputFormat) (string, error) {
	switch format {
	case OutputJSON:
		return formatJSON(data)
	case OutputYAML:
		return formatYAML(data)
	case OutputTable:
		return formatTable(data)
	default:
		return formatTable(data)
	}
}

func formatJSON(data any) (string, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal JSON: %w", err)
	}
	return string(b), nil
}

func formatYAML(data any) (string, error) {
	b, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal YAML: %w", err)
	}
	return string(b), nil
}

func formatTable(data any) (string, error) {
	if data == nil {
		return "", nil
	}

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "", nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		return formatSliceTable(data)
	case reflect.Map:
		return formatMapTable(data)
	case reflect.Struct:
		return formatStructTable(data)
	default:
		return fmt.Sprintf("%v", data), nil
	}
}

func formatSliceTable(data any) (string, error) {
	v := reflect.ValueOf(data)
	if v.Len() == 0 {
		return "No items", nil
	}

	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)

	first := v.Index(0).Interface()
	headers := getFields(first)

	fmt.Fprintln(w, strings.Join(headers, "\t"))
	fmt.Fprintln(w, strings.Join(makeSeparators(len(headers)), "\t"))

	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		values := getFieldValues(item, headers)
		fmt.Fprintln(w, strings.Join(values, "\t"))
	}

	w.Flush()
	return sb.String(), nil
}

func formatMapTable(data any) (string, error) {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Map {
		var sb strings.Builder
		w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)

		iter := v.MapRange()
		for iter.Next() {
			key := fmt.Sprintf("%v", iter.Key())
			value := formatValue(iter.Value().Interface())
			fmt.Fprintf(w, "%s\t%s\n", key, value)
		}

		w.Flush()
		return sb.String(), nil
	}
	return fmt.Sprintf("%v", data), nil
}

func formatStructTable(data any) (string, error) {
	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)

	headers := getFields(data)
	values := getFieldValues(data, headers)

	for i, h := range headers {
		fmt.Fprintf(w, "%s\t%s\n", h, values[i])
	}

	w.Flush()
	return sb.String(), nil
}

func getFields(data any) []string {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return []string{"value"}
	}

	t := v.Type()
	var fields []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name := field.Tag.Get("json")
		if name == "" || name == "-" {
			name = field.Name
		} else {
			if idx := strings.Index(name, ","); idx != -1 {
				name = name[:idx]
			}
		}
		fields = append(fields, name)
	}
	return fields
}

func getFieldValues(data any, fields []string) []string {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	values := make([]string, len(fields))
	if v.Kind() == reflect.Map {
		for i, field := range fields {
			fv := v.MapIndex(reflect.ValueOf(field))
			if fv.IsValid() {
				values[i] = formatValue(fv.Interface())
			} else {
				values[i] = ""
			}
		}
		return values
	}

	if v.Kind() != reflect.Struct {
		for i := range fields {
			if i == 0 {
				values[i] = formatValue(data)
			} else {
				values[i] = ""
			}
		}
		return values
	}

	t := v.Type()
	fieldMap := make(map[string]int)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name := field.Tag.Get("json")
		if name == "" || name == "-" {
			name = field.Name
		} else {
			if idx := strings.Index(name, ","); idx != -1 {
				name = name[:idx]
			}
		}
		fieldMap[name] = i
	}

	for i, field := range fields {
		if idx, ok := fieldMap[field]; ok {
			fv := v.Field(idx)
			values[i] = formatValue(fv.Interface())
		} else {
			values[i] = ""
		}
	}

	return values
}

func formatValue(v any) string {
	if v == nil {
		return ""
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return ""
		}
		v = rv.Elem().Interface()
	}

	switch val := v.(type) {
	case string:
		return val
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%.2f", val)
	case bool:
		return fmt.Sprintf("%t", val)
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	}
}

func makeSeparators(count int) []string {
	seps := make([]string, count)
	for i := range seps {
		seps[i] = strings.Repeat("-", 10)
	}
	return seps
}

func PrintOutput(data any, opts *OutputOptions) error {
	if opts.Quiet {
		return nil
	}

	output, err := FormatOutput(data, opts.Format)
	if err != nil {
		return err
	}

	fmt.Fprint(opts.Writer, output)
	return nil
}

func PrintError(err error, opts *OutputOptions) {
	if opts.Format == OutputJSON {
		data := map[string]any{
			"success": false,
			"error": map[string]string{
				"message": err.Error(),
			},
		}
		b, _ := json.MarshalIndent(data, "", "  ")
		fmt.Fprintln(os.Stderr, string(b))
	} else if opts.Format == OutputYAML {
		data := map[string]any{
			"success": false,
			"error": map[string]string{
				"message": err.Error(),
			},
		}
		b, _ := yaml.Marshal(data)
		fmt.Fprint(os.Stderr, string(b))
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

func PrintSuccess(message string, opts *OutputOptions) {
	if opts.Quiet {
		return
	}

	if opts.Format == OutputJSON {
		data := map[string]any{
			"success": true,
			"message": message,
		}
		b, _ := json.MarshalIndent(data, "", "  ")
		fmt.Fprintln(opts.Writer, string(b))
	} else if opts.Format == OutputYAML {
		data := map[string]any{
			"success": true,
			"message": message,
		}
		b, _ := yaml.Marshal(data)
		fmt.Fprint(opts.Writer, string(b))
	} else {
		fmt.Fprintln(opts.Writer, message)
	}
}
