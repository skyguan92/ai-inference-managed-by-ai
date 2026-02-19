package model

import (
	"testing"
	"time"
)

func TestNewCreatedEvent(t *testing.T) {
	m := createTestModel("model-1", "llama3")

	before := time.Now()
	evt := NewCreatedEvent(m)
	after := time.Now()

	if evt.Type() != EventTypeCreated {
		t.Errorf("Type() = %q, want %q", evt.Type(), EventTypeCreated)
	}
	if evt.Domain() != "model" {
		t.Errorf("Domain() = %q, want 'model'", evt.Domain())
	}
	if evt.CorrelationID() == "" {
		t.Error("CorrelationID() should not be empty")
	}
	ts := evt.Timestamp()
	if ts.Before(before) || ts.After(after) {
		t.Errorf("Timestamp() = %v, should be between %v and %v", ts, before, after)
	}
	if evt.Payload() == nil {
		t.Error("Payload() should not be nil")
	}

	payload, ok := evt.Payload().(map[string]any)
	if !ok {
		t.Fatal("Payload() should be map[string]any")
	}
	if payload["model_id"] != m.ID {
		t.Errorf("payload[model_id] = %v, want %q", payload["model_id"], m.ID)
	}
	if payload["name"] != "llama3" {
		t.Errorf("payload[name] = %v, want 'llama3'", payload["name"])
	}
}

func TestNewDeletedEvent(t *testing.T) {
	evt := NewDeletedEvent("model-1", "llama3")

	if evt.Type() != EventTypeDeleted {
		t.Errorf("Type() = %q, want %q", evt.Type(), EventTypeDeleted)
	}
	if evt.Domain() != "model" {
		t.Errorf("Domain() = %q, want 'model'", evt.Domain())
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
	if payload["model_id"] != "model-1" {
		t.Errorf("payload[model_id] = %v, want 'model-1'", payload["model_id"])
	}
	if payload["name"] != "llama3" {
		t.Errorf("payload[name] = %v, want 'llama3'", payload["name"])
	}
}

func TestNewPullProgressEvent(t *testing.T) {
	progress := &PullProgress{
		ModelID:    "model-1",
		Status:     "downloading",
		Progress:   45.5,
		BytesTotal: 4 * 1024 * 1024 * 1024,
		BytesDone:  2 * 1024 * 1024 * 1024,
		Speed:      10.5,
		Error:      "",
	}

	evt := NewPullProgressEvent(progress)

	if evt.Type() != EventTypePullProgress {
		t.Errorf("Type() = %q, want %q", evt.Type(), EventTypePullProgress)
	}
	if evt.Domain() != "model" {
		t.Errorf("Domain() = %q, want 'model'", evt.Domain())
	}
	if evt.CorrelationID() == "" {
		t.Error("CorrelationID() should not be empty")
	}

	payload, ok := evt.Payload().(map[string]any)
	if !ok {
		t.Fatal("Payload() should be map[string]any")
	}
	if payload["model_id"] != "model-1" {
		t.Errorf("payload[model_id] = %v, want 'model-1'", payload["model_id"])
	}
	if payload["status"] != "downloading" {
		t.Errorf("payload[status] = %v, want 'downloading'", payload["status"])
	}
	if payload["progress"] != 45.5 {
		t.Errorf("payload[progress] = %v, want 45.5", payload["progress"])
	}
}

func TestNewVerifiedEvent(t *testing.T) {
	result := &VerificationResult{
		Valid:  true,
		Issues: nil,
	}
	evt := NewVerifiedEvent("model-1", result)

	if evt.Type() != EventTypeVerified {
		t.Errorf("Type() = %q, want %q", evt.Type(), EventTypeVerified)
	}
	if evt.Domain() != "model" {
		t.Errorf("Domain() = %q, want 'model'", evt.Domain())
	}
	if evt.CorrelationID() == "" {
		t.Error("CorrelationID() should not be empty")
	}

	payload, ok := evt.Payload().(map[string]any)
	if !ok {
		t.Fatal("Payload() should be map[string]any")
	}
	if payload["model_id"] != "model-1" {
		t.Errorf("payload[model_id] = %v, want 'model-1'", payload["model_id"])
	}
	if payload["valid"] != true {
		t.Errorf("payload[valid] = %v, want true", payload["valid"])
	}
}

func TestNewVerifiedEvent_Invalid(t *testing.T) {
	result := &VerificationResult{
		Valid:  false,
		Issues: []string{"checksum mismatch", "size invalid"},
	}
	evt := NewVerifiedEvent("model-2", result)

	payload, ok := evt.Payload().(map[string]any)
	if !ok {
		t.Fatal("Payload() should be map[string]any")
	}
	if payload["valid"] != false {
		t.Errorf("payload[valid] = %v, want false", payload["valid"])
	}
	issues, ok := payload["issues"].([]string)
	if !ok {
		t.Fatal("payload[issues] should be []string")
	}
	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(issues))
	}
}

func TestModelEventTypeConstants(t *testing.T) {
	if EventTypeCreated != "model.created" {
		t.Errorf("EventTypeCreated = %q, want 'model.created'", EventTypeCreated)
	}
	if EventTypeDeleted != "model.deleted" {
		t.Errorf("EventTypeDeleted = %q, want 'model.deleted'", EventTypeDeleted)
	}
	if EventTypePullProgress != "model.pull_progress" {
		t.Errorf("EventTypePullProgress = %q, want 'model.pull_progress'", EventTypePullProgress)
	}
	if EventTypeVerified != "model.verified" {
		t.Errorf("EventTypeVerified = %q, want 'model.verified'", EventTypeVerified)
	}
}

func TestModelEvents_UniqueCorrelationIDs(t *testing.T) {
	evt1 := NewDeletedEvent("model-1", "llama3")
	evt2 := NewDeletedEvent("model-1", "llama3")

	if evt1.CorrelationID() == evt2.CorrelationID() {
		t.Error("different events should have unique correlation IDs")
	}
}
