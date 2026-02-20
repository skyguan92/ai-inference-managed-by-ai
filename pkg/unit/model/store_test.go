package model

import (
	"context"
	"testing"
)

func TestMemoryStore_Create(t *testing.T) {
	s := NewMemoryStore()
	m := createTestModel("model-1", "llama3")

	err := s.Create(context.Background(), m)
	if err != nil {
		t.Errorf("Create() failed: %v", err)
	}

	// Duplicate create should fail
	err = s.Create(context.Background(), m)
	if err != ErrModelAlreadyExists {
		t.Errorf("expected ErrModelAlreadyExists, got %v", err)
	}
}

func TestMemoryStore_Get(t *testing.T) {
	s := NewMemoryStore()
	m := createTestModel("model-1", "llama3")
	_ = s.Create(context.Background(), m)

	got, err := s.Get(context.Background(), "model-1")
	if err != nil {
		t.Errorf("Get() failed: %v", err)
	}
	if got.Name != "llama3" {
		t.Errorf("expected Name 'llama3', got %q", got.Name)
	}

	_, err = s.Get(context.Background(), "nonexistent")
	if err != ErrModelNotFound {
		t.Errorf("expected ErrModelNotFound, got %v", err)
	}
}

func TestMemoryStore_Update(t *testing.T) {
	s := NewMemoryStore()
	m := createTestModel("model-1", "llama3")
	_ = s.Create(context.Background(), m)

	m.Name = "llama3-updated"
	err := s.Update(context.Background(), m)
	if err != nil {
		t.Errorf("Update() failed: %v", err)
	}

	got, _ := s.Get(context.Background(), "model-1")
	if got.Name != "llama3-updated" {
		t.Errorf("expected Name 'llama3-updated', got %q", got.Name)
	}

	// Update non-existent should fail
	notExist := createTestModel("nonexistent", "test")
	err = s.Update(context.Background(), notExist)
	if err != ErrModelNotFound {
		t.Errorf("expected ErrModelNotFound, got %v", err)
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	s := NewMemoryStore()
	m := createTestModel("model-1", "llama3")
	_ = s.Create(context.Background(), m)

	err := s.Delete(context.Background(), "model-1")
	if err != nil {
		t.Errorf("Delete() failed: %v", err)
	}

	_, err = s.Get(context.Background(), "model-1")
	if err != ErrModelNotFound {
		t.Errorf("expected ErrModelNotFound after delete, got %v", err)
	}

	// Delete non-existent should fail
	err = s.Delete(context.Background(), "nonexistent")
	if err != ErrModelNotFound {
		t.Errorf("expected ErrModelNotFound, got %v", err)
	}
}

func TestMemoryStore_List_WithFilters(t *testing.T) {
	s := NewMemoryStore()
	llm1 := createTestModel("model-1", "llama3")
	llm2 := createTestModel("model-2", "mistral")
	vlm1 := createTestModel("model-3", "llava")
	vlm1.Type = ModelTypeVLM
	vlm1.Format = FormatSafetensors
	vlm1.Status = StatusPulling

	_ = s.Create(context.Background(), llm1)
	_ = s.Create(context.Background(), llm2)
	_ = s.Create(context.Background(), vlm1)

	// Filter by type
	results, total, err := s.List(context.Background(), ModelFilter{Type: ModelTypeLLM})
	if err != nil {
		t.Errorf("List() failed: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Filter by status
	_, total, err = s.List(context.Background(), ModelFilter{Status: StatusPulling})
	if err != nil {
		t.Errorf("List() failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}

	// Filter by format
	_, total, err = s.List(context.Background(), ModelFilter{Format: FormatSafetensors})
	if err != nil {
		t.Errorf("List() failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}

	// Limit and offset
	results, total, err = s.List(context.Background(), ModelFilter{Limit: 1, Offset: 0})
	if err != nil {
		t.Errorf("List() failed: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result (limited), got %d", len(results))
	}

	// Offset beyond end
	results, _, err = s.List(context.Background(), ModelFilter{Offset: 100})
	if err != nil {
		t.Errorf("List() failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for large offset, got %d", len(results))
	}

	// Limit+offset beyond end
	results, total, err = s.List(context.Background(), ModelFilter{Limit: 10, Offset: 2})
	if err != nil {
		t.Errorf("List() failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	_ = total
	_ = results
}

func TestCreateCommandWithEvents(t *testing.T) {
	store := NewMemoryStore()
	pub := &mockPublisher{}

	cmd := NewCreateCommandWithEvents(store, pub)
	result, err := cmd.Execute(context.Background(), map[string]any{
		"name": "test-model",
		"type": "llm",
	})
	if err != nil {
		t.Errorf("Execute() failed: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Error("expected result to be map[string]any")
	}
	if _, exists := resultMap["model_id"]; !exists {
		t.Error("expected 'model_id' field")
	}
	if len(pub.events) == 0 {
		t.Error("expected event to be published")
	}
}

func TestDeleteCommandWithEvents(t *testing.T) {
	store := NewMemoryStore()
	_ = store.Create(context.Background(), createTestModel("model-del-1", "test"))
	pub := &mockPublisher{}

	cmd := NewDeleteCommandWithEvents(store, pub)
	_, err := cmd.Execute(context.Background(), map[string]any{
		"model_id": "model-del-1",
	})
	if err != nil {
		t.Errorf("Execute() failed: %v", err)
	}
	if len(pub.events) == 0 {
		t.Error("expected event to be published")
	}
}

func TestCreateCommand_Execute_InvalidInput(t *testing.T) {
	cmd := NewCreateCommand(NewMemoryStore())
	_, err := cmd.Execute(context.Background(), "not-a-map")
	if err == nil {
		t.Error("expected error for invalid input type")
	}
}

func TestDeleteCommand_Execute_InvalidInput(t *testing.T) {
	cmd := NewDeleteCommand(NewMemoryStore())
	_, err := cmd.Execute(context.Background(), "not-a-map")
	if err == nil {
		t.Error("expected error for invalid input type")
	}
}

// mockPublisher implements EventPublisher for model package tests
type mockPublisher struct {
	events []any
}

func (m *mockPublisher) Publish(event any) error {
	m.events = append(m.events, event)
	return nil
}
