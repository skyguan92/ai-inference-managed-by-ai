package service

import (
	"testing"
	"time"
)

func TestGetString(t *testing.T) {
	m := map[string]any{
		"key":     "value",
		"num":     42,
		"missing": nil,
	}

	if got := getString(m, "key"); got != "value" {
		t.Errorf("getString(key) = %q, want %q", got, "value")
	}
	if got := getString(m, "num"); got != "" {
		t.Errorf("getString(num) = %q, want empty string (wrong type)", got)
	}
	if got := getString(m, "nonexistent"); got != "" {
		t.Errorf("getString(nonexistent) = %q, want empty string", got)
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  int
	}{
		{"int", int(42), 42},
		{"int32", int32(100), 100},
		{"int64", int64(200), 200},
		{"float64", float64(3.7), 3},
		{"string", "42", 0},
		{"missing", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := map[string]any{}
			if tt.value != nil {
				m["k"] = tt.value
			}
			if got := getInt(m, "k"); got != tt.want {
				t.Errorf("getInt(%v) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}

func TestGetInt64(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  int64
	}{
		{"int", int(42), 42},
		{"int32", int32(100), 100},
		{"int64", int64(9999999999), 9999999999},
		{"float64", float64(3.7), 3},
		{"string", "42", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := map[string]any{"k": tt.value}
			if got := getInt64(m, "k"); got != tt.want {
				t.Errorf("getInt64(%v) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}

	// Missing key
	if got := getInt64(map[string]any{}, "missing"); got != 0 {
		t.Errorf("getInt64 missing key = %d, want 0", got)
	}
}

func TestGetUint64(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  uint64
	}{
		{"uint64", uint64(1000), 1000},
		{"int64", int64(2000), 2000},
		{"float64", float64(3000.7), 3000},
		{"int", int(4000), 4000},
		{"string", "42", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := map[string]any{"k": tt.value}
			if got := getUint64(m, "k"); got != tt.want {
				t.Errorf("getUint64(%v) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}

	// Missing key
	if got := getUint64(map[string]any{}, "missing"); got != 0 {
		t.Errorf("getUint64 missing key = %d, want 0", got)
	}
}

func TestGetFloat64(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  float64
	}{
		{"float64", float64(3.14), 3.14},
		{"float32", float32(2.5), float64(float32(2.5))},
		{"int", int(10), 10.0},
		{"int64", int64(20), 20.0},
		{"string", "3.14", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := map[string]any{"k": tt.value}
			if got := getFloat64(m, "k"); got != tt.want {
				t.Errorf("getFloat64(%v) = %f, want %f", tt.value, got, tt.want)
			}
		})
	}

	// Missing key
	if got := getFloat64(map[string]any{}, "missing"); got != 0.0 {
		t.Errorf("getFloat64 missing key = %f, want 0.0", got)
	}
}

func TestGetBool(t *testing.T) {
	m := map[string]any{
		"true":    true,
		"false":   false,
		"string":  "true",
		"integer": 1,
	}

	if got := getBool(m, "true"); !got {
		t.Error("getBool(true) = false, want true")
	}
	if got := getBool(m, "false"); got {
		t.Error("getBool(false) = true, want false")
	}
	if got := getBool(m, "string"); got {
		t.Error("getBool(string) = true, want false (wrong type)")
	}
	if got := getBool(m, "nonexistent"); got {
		t.Error("getBool(nonexistent) = true, want false")
	}
}

func TestGetStringSlice(t *testing.T) {
	t.Run("[]string type", func(t *testing.T) {
		m := map[string]any{"k": []string{"a", "b", "c"}}
		got := getStringSlice(m, "k")
		if len(got) != 3 {
			t.Errorf("len = %d, want 3", len(got))
		}
		if got[0] != "a" || got[1] != "b" || got[2] != "c" {
			t.Errorf("values = %v, want [a b c]", got)
		}
	})

	t.Run("[]any with strings type", func(t *testing.T) {
		m := map[string]any{"k": []any{"x", "y"}}
		got := getStringSlice(m, "k")
		if len(got) != 2 {
			t.Errorf("len = %d, want 2", len(got))
		}
		if got[0] != "x" || got[1] != "y" {
			t.Errorf("values = %v, want [x y]", got)
		}
	})

	t.Run("[]any with mixed types", func(t *testing.T) {
		m := map[string]any{"k": []any{"str", 42, true}}
		got := getStringSlice(m, "k")
		// Only the string elements should be included
		if len(got) != 1 {
			t.Errorf("len = %d, want 1 (only string elements)", len(got))
		}
		if got[0] != "str" {
			t.Errorf("values = %v, want [str]", got)
		}
	})

	t.Run("wrong type", func(t *testing.T) {
		m := map[string]any{"k": "not-a-slice"}
		got := getStringSlice(m, "k")
		if got != nil {
			t.Errorf("expected nil for wrong type, got %v", got)
		}
	})

	t.Run("missing key", func(t *testing.T) {
		m := map[string]any{}
		got := getStringSlice(m, "missing")
		if got != nil {
			t.Errorf("expected nil for missing key, got %v", got)
		}
	})
}

func TestBaseEvent(t *testing.T) {
	e := &BaseEvent{
		eventType: "test.event",
		domain:    "test",
		payload:   map[string]any{"key": "value"},
	}

	if e.Type() != "test.event" {
		t.Errorf("Type() = %q, want 'test.event'", e.Type())
	}
	if e.Domain() != "test" {
		t.Errorf("Domain() = %q, want 'test'", e.Domain())
	}
	if e.Payload() == nil {
		t.Error("Payload() should not be nil")
	}
	if e.CorrelationID() != "" {
		t.Errorf("CorrelationID() = %q, want empty string", e.CorrelationID())
	}

	ts := e.Timestamp()
	if ts.IsZero() {
		t.Error("Timestamp() should not be zero")
	}
	// Timestamp should be close to now
	if time.Since(ts) > time.Second {
		t.Errorf("Timestamp() = %v, should be very recent", ts)
	}
}
