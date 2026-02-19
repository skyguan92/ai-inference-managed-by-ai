package engine

import (
	"testing"
	"time"
)

func TestNewStartedEvent(t *testing.T) {
	engine := createTestEngine("ollama", EngineTypeOllama)
	engine.Status = EngineStatusRunning

	before := time.Now()
	evt := NewStartedEvent(engine, "proc-1234")
	after := time.Now()

	if evt.Type() != EventTypeStarted {
		t.Errorf("Type() = %q, want %q", evt.Type(), EventTypeStarted)
	}
	if evt.Domain() != "engine" {
		t.Errorf("Domain() = %q, want 'engine'", evt.Domain())
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
	if payload["engine_id"] != engine.ID {
		t.Errorf("payload[engine_id] = %v, want %q", payload["engine_id"], engine.ID)
	}
	if payload["name"] != "ollama" {
		t.Errorf("payload[name] = %v, want 'ollama'", payload["name"])
	}
	if payload["process_id"] != "proc-1234" {
		t.Errorf("payload[process_id] = %v, want 'proc-1234'", payload["process_id"])
	}
}

func TestNewStoppedEvent(t *testing.T) {
	engine := createTestEngine("vllm", EngineTypeVLLM)
	evt := NewStoppedEvent(engine, "user requested")

	if evt.Type() != EventTypeStopped {
		t.Errorf("Type() = %q, want %q", evt.Type(), EventTypeStopped)
	}
	if evt.Domain() != "engine" {
		t.Errorf("Domain() = %q, want 'engine'", evt.Domain())
	}
	if evt.CorrelationID() == "" {
		t.Error("CorrelationID() should not be empty")
	}
	if evt.Timestamp().IsZero() {
		t.Error("Timestamp() should not be zero")
	}

	payload, ok := evt.Payload().(map[string]any)
	if !ok {
		t.Fatal("Payload() should be map[string]any")
	}
	if payload["reason"] != "user requested" {
		t.Errorf("payload[reason] = %v, want 'user requested'", payload["reason"])
	}
	if payload["name"] != "vllm" {
		t.Errorf("payload[name] = %v, want 'vllm'", payload["name"])
	}
}

func TestNewErrorEvent(t *testing.T) {
	engine := createTestEngine("ollama", EngineTypeOllama)
	evt := NewErrorEvent(engine, "out of memory", "OOM_ERROR")

	if evt.Type() != EventTypeError {
		t.Errorf("Type() = %q, want %q", evt.Type(), EventTypeError)
	}
	if evt.Domain() != "engine" {
		t.Errorf("Domain() = %q, want 'engine'", evt.Domain())
	}
	if evt.CorrelationID() == "" {
		t.Error("CorrelationID() should not be empty")
	}
	if evt.Payload() == nil {
		t.Error("Payload() should not be nil")
	}

	payload, ok := evt.Payload().(map[string]any)
	if !ok {
		t.Fatal("Payload() should be map[string]any")
	}
	if payload["error"] != "out of memory" {
		t.Errorf("payload[error] = %v, want 'out of memory'", payload["error"])
	}
	if payload["error_code"] != "OOM_ERROR" {
		t.Errorf("payload[error_code] = %v, want 'OOM_ERROR'", payload["error_code"])
	}
}

func TestNewHealthChangedEvent(t *testing.T) {
	engine := createTestEngine("ollama", EngineTypeOllama)
	details := map[string]any{
		"cpu_usage": 85.5,
		"issue":     "high load",
	}
	evt := NewHealthChangedEvent(engine, EngineStatusRunning, EngineStatusError, details)

	if evt.Type() != EventTypeHealthChanged {
		t.Errorf("Type() = %q, want %q", evt.Type(), EventTypeHealthChanged)
	}
	if evt.Domain() != "engine" {
		t.Errorf("Domain() = %q, want 'engine'", evt.Domain())
	}
	if evt.CorrelationID() == "" {
		t.Error("CorrelationID() should not be empty")
	}

	payload, ok := evt.Payload().(map[string]any)
	if !ok {
		t.Fatal("Payload() should be map[string]any")
	}
	if payload["old_status"] != string(EngineStatusRunning) {
		t.Errorf("payload[old_status] = %v, want %q", payload["old_status"], EngineStatusRunning)
	}
	if payload["new_status"] != string(EngineStatusError) {
		t.Errorf("payload[new_status] = %v, want %q", payload["new_status"], EngineStatusError)
	}
	// Details should be merged into payload
	if payload["cpu_usage"] != 85.5 {
		t.Errorf("payload[cpu_usage] = %v, want 85.5", payload["cpu_usage"])
	}
	if payload["issue"] != "high load" {
		t.Errorf("payload[issue] = %v, want 'high load'", payload["issue"])
	}
}

func TestNewHealthChangedEvent_NilDetails(t *testing.T) {
	engine := createTestEngine("ollama", EngineTypeOllama)
	// nil details should not panic
	evt := NewHealthChangedEvent(engine, EngineStatusStopped, EngineStatusRunning, nil)

	if evt == nil {
		t.Fatal("NewHealthChangedEvent should not return nil")
	}
	if evt.Type() != EventTypeHealthChanged {
		t.Errorf("Type() = %q, want %q", evt.Type(), EventTypeHealthChanged)
	}
}

func TestEngineEventTypeConstants(t *testing.T) {
	if EventTypeStarted != "engine.started" {
		t.Errorf("EventTypeStarted = %q, want 'engine.started'", EventTypeStarted)
	}
	if EventTypeStopped != "engine.stopped" {
		t.Errorf("EventTypeStopped = %q, want 'engine.stopped'", EventTypeStopped)
	}
	if EventTypeError != "engine.error" {
		t.Errorf("EventTypeError = %q, want 'engine.error'", EventTypeError)
	}
	if EventTypeHealthChanged != "engine.health_changed" {
		t.Errorf("EventTypeHealthChanged = %q, want 'engine.health_changed'", EventTypeHealthChanged)
	}
}

func TestEngineEvents_UniqueCorrelationIDs(t *testing.T) {
	engine := createTestEngine("ollama", EngineTypeOllama)
	evt1 := NewStartedEvent(engine, "proc-1")
	evt2 := NewStartedEvent(engine, "proc-2")

	if evt1.CorrelationID() == evt2.CorrelationID() {
		t.Error("different events should have unique correlation IDs")
	}
}
