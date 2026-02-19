package logger

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestInit_Text(t *testing.T) {
	Reset()
	buf := &bytes.Buffer{}
	Init(Config{
		Level:  "info",
		Format: "text",
		Output: buf,
	})
	defer Reset()

	Info("test message", "key", "value")
	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("expected 'test message' in output, got: %s", output)
	}
}

func TestInit_JSON(t *testing.T) {
	Reset()
	buf := &bytes.Buffer{}
	Init(Config{
		Level:  "info",
		Format: "json",
		Output: buf,
	})
	defer Reset()

	Info("json message")
	output := buf.String()
	if !strings.Contains(output, "json message") {
		t.Errorf("expected 'json message' in output, got: %s", output)
	}
}

func TestInit_OnlyCalledOnce(t *testing.T) {
	Reset()
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	Init(Config{Level: "info", Format: "text", Output: buf1})
	Init(Config{Level: "info", Format: "text", Output: buf2}) // second call is no-op

	Info("only once")

	// Only buf1 should have received the log
	if buf1.Len() == 0 {
		t.Error("expected buf1 to have output")
	}
	if buf2.Len() != 0 {
		t.Error("expected buf2 to be empty (second Init is a no-op)")
	}

	Reset()
}

func TestDefault_BeforeInit(t *testing.T) {
	Reset()
	l := Default()
	if l == nil {
		t.Error("Default() should never return nil")
	}
}

func TestDefault_AfterInit(t *testing.T) {
	Reset()
	buf := &bytes.Buffer{}
	Init(Config{Level: "info", Format: "text", Output: buf})
	defer Reset()

	l := Default()
	if l == nil {
		t.Error("Default() should return non-nil logger after Init")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		wantInfo bool // true if result should be info or lower
	}{
		{"debug", false},
		{"info", false},
		{"warn", false},
		{"warning", false},
		{"error", false},
		{"", false},       // defaults to info
		{"invalid", false}, // defaults to info
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level := parseLevel(tt.input)
			_ = level // just ensure no panic
		})
	}
}

func TestWithContext_Empty(t *testing.T) {
	Reset()
	buf := &bytes.Buffer{}
	Init(Config{Level: "info", Format: "text", Output: buf})
	defer Reset()

	ctx := context.Background()
	l := WithContext(ctx)
	if l == nil {
		t.Error("WithContext should not return nil")
	}
}

func TestWithContext_WithValues(t *testing.T) {
	Reset()
	buf := &bytes.Buffer{}
	Init(Config{Level: "info", Format: "text", Output: buf})
	defer Reset()

	ctx := context.Background()
	ctx = SetRequestID(ctx, "req-123")
	ctx = SetTraceID(ctx, "trc-456")
	ctx = SetUnit(ctx, "model.list")

	l := WithContext(ctx)
	if l == nil {
		t.Error("WithContext should not return nil")
	}

	l.Info("context test")
	output := buf.String()
	if !strings.Contains(output, "req-123") {
		t.Errorf("expected request_id in output: %s", output)
	}
	if !strings.Contains(output, "trc-456") {
		t.Errorf("expected trace_id in output: %s", output)
	}
	if !strings.Contains(output, "model.list") {
		t.Errorf("expected unit in output: %s", output)
	}
}

func TestSetRequestID(t *testing.T) {
	ctx := context.Background()
	ctx = SetRequestID(ctx, "req-abc")
	if got := GetRequestID(ctx); got != "req-abc" {
		t.Errorf("GetRequestID() = %q, want 'req-abc'", got)
	}
}

func TestSetTraceID(t *testing.T) {
	ctx := context.Background()
	ctx = SetTraceID(ctx, "trc-xyz")
	if got := GetTraceID(ctx); got != "trc-xyz" {
		t.Errorf("GetTraceID() = %q, want 'trc-xyz'", got)
	}
}

func TestSetUnit(t *testing.T) {
	ctx := context.Background()
	ctx = SetUnit(ctx, "inference.chat")
	// SetUnit doesn't have a getter, just verify no panic
	_ = ctx
}

func TestGetRequestID_Missing(t *testing.T) {
	ctx := context.Background()
	if got := GetRequestID(ctx); got != "" {
		t.Errorf("GetRequestID() = %q, want empty string", got)
	}
}

func TestGetTraceID_Missing(t *testing.T) {
	ctx := context.Background()
	if got := GetTraceID(ctx); got != "" {
		t.Errorf("GetTraceID() = %q, want empty string", got)
	}
}

func TestLoggingFunctions(t *testing.T) {
	Reset()
	buf := &bytes.Buffer{}
	Init(Config{Level: "debug", Format: "text", Output: buf})
	defer Reset()

	// These should not panic
	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")

	output := buf.String()
	if !strings.Contains(output, "debug message") {
		t.Error("expected debug message in output")
	}
	if !strings.Contains(output, "info message") {
		t.Error("expected info message in output")
	}
}

func TestInit_WithAddSource(t *testing.T) {
	Reset()
	buf := &bytes.Buffer{}
	Init(Config{
		Level:     "info",
		Format:    "text",
		Output:    buf,
		AddSource: true,
	})
	defer Reset()

	Info("source message")
	// Just verify no panic
}
