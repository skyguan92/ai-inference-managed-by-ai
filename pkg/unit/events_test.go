package unit

import (
	"errors"
	"testing"
	"time"
)

type mockEventPublisher struct {
	events []any
	err    error
}

func (m *mockEventPublisher) Publish(event any) error {
	if m.err != nil {
		return m.err
	}
	m.events = append(m.events, event)
	return nil
}

func TestNoopEventPublisher(t *testing.T) {
	pub := &NoopEventPublisher{}
	err := pub.Publish("any event")
	if err != nil {
		t.Errorf("NoopEventPublisher.Publish() should return nil, got %v", err)
	}

	// Verify interface implementation
	var _ EventPublisher = (*NoopEventPublisher)(nil)
}

func TestNewExecutionContext(t *testing.T) {
	pub := &mockEventPublisher{}
	ec := NewExecutionContext(pub, "model", "model.pull")

	if ec.Publisher != pub {
		t.Error("expected Publisher to be set")
	}
	if ec.Domain != "model" {
		t.Errorf("expected Domain 'model', got %q", ec.Domain)
	}
	if ec.UnitName != "model.pull" {
		t.Errorf("expected UnitName 'model.pull', got %q", ec.UnitName)
	}
	if ec.CorrelationID == "" {
		t.Error("expected non-empty CorrelationID")
	}
	if ec.StartTime.IsZero() {
		t.Error("expected non-zero StartTime")
	}
}

func TestNewExecutionContext_NilPublisher(t *testing.T) {
	ec := NewExecutionContext(nil, "engine", "engine.start")
	if ec == nil {
		t.Fatal("NewExecutionContext should return non-nil even with nil publisher")
	}
	if ec.Publisher != nil {
		t.Error("Publisher should be nil")
	}
}

func TestExecutionContext_PublishStarted(t *testing.T) {
	pub := &mockEventPublisher{}
	ec := NewExecutionContext(pub, "model", "model.pull")

	input := map[string]any{"source": "ollama", "repo": "llama3"}
	ec.PublishStarted(input)

	if len(pub.events) != 1 {
		t.Errorf("expected 1 event published, got %d", len(pub.events))
	}

	evt, ok := pub.events[0].(*ExecutionEvent)
	if !ok {
		t.Error("expected *ExecutionEvent")
		return
	}
	if evt.EventType != string(ExecutionStarted) {
		t.Errorf("expected EventType %q, got %q", ExecutionStarted, evt.EventType)
	}
	if evt.EventDomain != "model" {
		t.Errorf("expected Domain 'model', got %q", evt.EventDomain)
	}
	if evt.UnitName != "model.pull" {
		t.Errorf("expected UnitName 'model.pull', got %q", evt.UnitName)
	}
	if evt.Input == nil {
		t.Error("expected Input to be set")
	}
	if evt.EventCorrelationID != ec.CorrelationID {
		t.Errorf("expected CorrelationID %q, got %q", ec.CorrelationID, evt.EventCorrelationID)
	}
}

func TestExecutionContext_PublishStarted_NilPublisher(t *testing.T) {
	ec := NewExecutionContext(nil, "model", "model.pull")
	// Should not panic
	ec.PublishStarted(map[string]any{"key": "value"})
}

func TestExecutionContext_PublishCompleted(t *testing.T) {
	pub := &mockEventPublisher{}
	ec := NewExecutionContext(pub, "model", "model.pull")
	ec.StartTime = time.Now().Add(-100 * time.Millisecond)

	output := map[string]any{"model_id": "model-123", "status": "ready"}
	ec.PublishCompleted(output)

	if len(pub.events) != 1 {
		t.Errorf("expected 1 event published, got %d", len(pub.events))
	}

	evt, ok := pub.events[0].(*ExecutionEvent)
	if !ok {
		t.Error("expected *ExecutionEvent")
		return
	}
	if evt.EventType != string(ExecutionCompleted) {
		t.Errorf("expected EventType %q, got %q", ExecutionCompleted, evt.EventType)
	}
	if evt.Output == nil {
		t.Error("expected Output to be set")
	}
	if evt.DurationMs <= 0 {
		t.Errorf("expected positive DurationMs, got %d", evt.DurationMs)
	}
}

func TestExecutionContext_PublishCompleted_NilPublisher(t *testing.T) {
	ec := NewExecutionContext(nil, "model", "model.pull")
	// Should not panic
	ec.PublishCompleted(map[string]any{"status": "done"})
}

func TestExecutionContext_PublishFailed(t *testing.T) {
	pub := &mockEventPublisher{}
	ec := NewExecutionContext(pub, "engine", "engine.start")
	ec.StartTime = time.Now().Add(-50 * time.Millisecond)

	testErr := errors.New("engine start failed")
	ec.PublishFailed(testErr)

	if len(pub.events) != 1 {
		t.Errorf("expected 1 event published, got %d", len(pub.events))
	}

	evt, ok := pub.events[0].(*ExecutionEvent)
	if !ok {
		t.Error("expected *ExecutionEvent")
		return
	}
	if evt.EventType != string(ExecutionFailed) {
		t.Errorf("expected EventType %q, got %q", ExecutionFailed, evt.EventType)
	}
	if evt.Error != "engine start failed" {
		t.Errorf("expected Error 'engine start failed', got %q", evt.Error)
	}
	if evt.DurationMs <= 0 {
		t.Errorf("expected positive DurationMs, got %d", evt.DurationMs)
	}
}

func TestExecutionContext_PublishFailed_NilError(t *testing.T) {
	pub := &mockEventPublisher{}
	ec := NewExecutionContext(pub, "engine", "engine.stop")

	ec.PublishFailed(nil)

	if len(pub.events) != 1 {
		t.Errorf("expected 1 event published, got %d", len(pub.events))
	}

	evt, ok := pub.events[0].(*ExecutionEvent)
	if !ok {
		t.Error("expected *ExecutionEvent")
		return
	}
	if evt.Error != "" {
		t.Errorf("expected empty Error for nil err, got %q", evt.Error)
	}
}

func TestExecutionContext_PublishFailed_NilPublisher(t *testing.T) {
	ec := NewExecutionContext(nil, "model", "model.verify")
	// Should not panic
	ec.PublishFailed(errors.New("some error"))
}

func TestExecutionEvent_Interface(t *testing.T) {
	now := time.Now()
	evt := &ExecutionEvent{
		EventType:          string(ExecutionStarted),
		EventDomain:        "model",
		UnitName:           "model.pull",
		Input:              map[string]any{"key": "value"},
		EventTimestamp:     now,
		EventCorrelationID: "corr-123",
		DurationMs:         150,
	}

	// Verify Event interface implementation
	var _ Event = (*ExecutionEvent)(nil)

	if evt.Type() != string(ExecutionStarted) {
		t.Errorf("Type() = %q, want %q", evt.Type(), ExecutionStarted)
	}
	if evt.Domain() != "model" {
		t.Errorf("Domain() = %q, want 'model'", evt.Domain())
	}
	if evt.Payload() != evt {
		t.Error("Payload() should return the event itself")
	}
	if !evt.Timestamp().Equal(now) {
		t.Errorf("Timestamp() = %v, want %v", evt.Timestamp(), now)
	}
	if evt.CorrelationID() != "corr-123" {
		t.Errorf("CorrelationID() = %q, want 'corr-123'", evt.CorrelationID())
	}
}

func TestExecutionEventType_Constants(t *testing.T) {
	if ExecutionStarted != "execution_started" {
		t.Errorf("ExecutionStarted = %q, want 'execution_started'", ExecutionStarted)
	}
	if ExecutionCompleted != "execution_completed" {
		t.Errorf("ExecutionCompleted = %q, want 'execution_completed'", ExecutionCompleted)
	}
	if ExecutionFailed != "execution_failed" {
		t.Errorf("ExecutionFailed = %q, want 'execution_failed'", ExecutionFailed)
	}
}

func TestExecutionContext_PublishWithPublisherError(t *testing.T) {
	pub := &mockEventPublisher{err: errors.New("publish error")}
	ec := NewExecutionContext(pub, "model", "model.pull")

	// Should not panic even when publisher returns error
	ec.PublishStarted(nil)
	ec.PublishCompleted(nil)
	ec.PublishFailed(nil)
}
