package inference

import (
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestNewRequestStartedEvent(t *testing.T) {
	before := time.Now()
	evt := NewRequestStartedEvent("req-1", "llama3", "chat")
	after := time.Now()

	if evt.Type() != EventTypeRequestStarted {
		t.Errorf("Type() = %q, want %q", evt.Type(), EventTypeRequestStarted)
	}
	if evt.Domain() != "inference" {
		t.Errorf("Domain() = %q, want 'inference'", evt.Domain())
	}
	if evt.Payload() == nil {
		t.Error("Payload() should not be nil")
	}
	if evt.CorrelationID() == "" {
		t.Error("CorrelationID() should not be empty")
	}
	ts := evt.Timestamp()
	if ts.Before(before) || ts.After(after) {
		t.Errorf("Timestamp() = %v, should be between %v and %v", ts, before, after)
	}

	payload, ok := evt.Payload().(map[string]any)
	if !ok {
		t.Fatal("Payload() should be map[string]any")
	}
	if payload["request_id"] != "req-1" {
		t.Errorf("payload[request_id] = %v, want 'req-1'", payload["request_id"])
	}
	if payload["model"] != "llama3" {
		t.Errorf("payload[model] = %v, want 'llama3'", payload["model"])
	}
	if payload["type"] != "chat" {
		t.Errorf("payload[type] = %v, want 'chat'", payload["type"])
	}

	// Verify Event interface implementation
	var _ unit.Event = evt
}

func TestNewRequestCompletedEvent(t *testing.T) {
	duration := 150 * time.Millisecond
	evt := NewRequestCompletedEvent("req-2", duration, 512)

	if evt.Type() != EventTypeRequestCompleted {
		t.Errorf("Type() = %q, want %q", evt.Type(), EventTypeRequestCompleted)
	}
	if evt.Domain() != "inference" {
		t.Errorf("Domain() = %q, want 'inference'", evt.Domain())
	}
	if evt.CorrelationID() == "" {
		t.Error("CorrelationID() should not be empty")
	}

	payload, ok := evt.Payload().(map[string]any)
	if !ok {
		t.Fatal("Payload() should be map[string]any")
	}
	if payload["request_id"] != "req-2" {
		t.Errorf("payload[request_id] = %v, want 'req-2'", payload["request_id"])
	}
	if payload["duration_ms"] != duration.Milliseconds() {
		t.Errorf("payload[duration_ms] = %v, want %d", payload["duration_ms"], duration.Milliseconds())
	}
	if payload["total_tokens"] != 512 {
		t.Errorf("payload[total_tokens] = %v, want 512", payload["total_tokens"])
	}

	var _ unit.Event = evt
}

func TestNewRequestFailedEvent(t *testing.T) {
	evt := NewRequestFailedEvent("req-3", "context deadline exceeded")

	if evt.Type() != EventTypeRequestFailed {
		t.Errorf("Type() = %q, want %q", evt.Type(), EventTypeRequestFailed)
	}
	if evt.Domain() != "inference" {
		t.Errorf("Domain() = %q, want 'inference'", evt.Domain())
	}
	if evt.CorrelationID() == "" {
		t.Error("CorrelationID() should not be empty")
	}
	if evt.Payload() == nil {
		t.Error("Payload() should not be nil")
	}
	if evt.Timestamp().IsZero() {
		t.Error("Timestamp() should not be zero")
	}

	payload, ok := evt.Payload().(map[string]any)
	if !ok {
		t.Fatal("Payload() should be map[string]any")
	}
	if payload["request_id"] != "req-3" {
		t.Errorf("payload[request_id] = %v, want 'req-3'", payload["request_id"])
	}
	if payload["error"] != "context deadline exceeded" {
		t.Errorf("payload[error] = %v, want 'context deadline exceeded'", payload["error"])
	}

	var _ unit.Event = evt
}

func TestEventTypeConstants(t *testing.T) {
	if EventTypeRequestStarted != "inference.request_started" {
		t.Errorf("EventTypeRequestStarted = %q, want 'inference.request_started'", EventTypeRequestStarted)
	}
	if EventTypeRequestCompleted != "inference.request_completed" {
		t.Errorf("EventTypeRequestCompleted = %q, want 'inference.request_completed'", EventTypeRequestCompleted)
	}
	if EventTypeRequestFailed != "inference.request_failed" {
		t.Errorf("EventTypeRequestFailed = %q, want 'inference.request_failed'", EventTypeRequestFailed)
	}
}

func TestRequestEvents_UniqueCorrelationIDs(t *testing.T) {
	evt1 := NewRequestStartedEvent("req-1", "model", "chat")
	evt2 := NewRequestStartedEvent("req-2", "model", "chat")

	if evt1.CorrelationID() == evt2.CorrelationID() {
		t.Error("different events should have different correlation IDs")
	}
}
