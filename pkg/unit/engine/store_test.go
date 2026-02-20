package engine

import (
	"context"
	"testing"
)

func TestMemoryStore_Create(t *testing.T) {
	s := NewMemoryStore()
	e := createTestEngine("ollama", EngineTypeOllama)

	err := s.Create(context.Background(), e)
	if err != nil {
		t.Errorf("Create() failed: %v", err)
	}

	// Duplicate create should fail
	err = s.Create(context.Background(), e)
	if err != ErrEngineAlreadyExists {
		t.Errorf("expected ErrEngineAlreadyExists, got %v", err)
	}
}

func TestMemoryStore_Get(t *testing.T) {
	s := NewMemoryStore()
	e := createTestEngine("ollama", EngineTypeOllama)
	_ = s.Create(context.Background(), e)

	got, err := s.Get(context.Background(), "ollama")
	if err != nil {
		t.Errorf("Get() failed: %v", err)
	}
	if got.Type != EngineTypeOllama {
		t.Errorf("expected Type %q, got %q", EngineTypeOllama, got.Type)
	}

	_, err = s.Get(context.Background(), "nonexistent")
	if err != ErrEngineNotFound {
		t.Errorf("expected ErrEngineNotFound, got %v", err)
	}
}

func TestMemoryStore_GetByID(t *testing.T) {
	s := NewMemoryStore()
	e := createTestEngine("ollama", EngineTypeOllama)
	_ = s.Create(context.Background(), e)

	got, err := s.GetByID(context.Background(), e.ID)
	if err != nil {
		t.Errorf("GetByID() failed: %v", err)
	}
	if got.Name != "ollama" {
		t.Errorf("expected Name 'ollama', got %q", got.Name)
	}

	_, err = s.GetByID(context.Background(), "nonexistent-id")
	if err != ErrEngineNotFound {
		t.Errorf("expected ErrEngineNotFound, got %v", err)
	}
}

func TestMemoryStore_Update(t *testing.T) {
	s := NewMemoryStore()
	e := createTestEngine("ollama", EngineTypeOllama)
	_ = s.Create(context.Background(), e)

	e.Status = EngineStatusRunning
	err := s.Update(context.Background(), e)
	if err != nil {
		t.Errorf("Update() failed: %v", err)
	}

	got, _ := s.Get(context.Background(), "ollama")
	if got.Status != EngineStatusRunning {
		t.Errorf("expected Status %q, got %q", EngineStatusRunning, got.Status)
	}

	// Update non-existent should fail
	notExist := createTestEngine("nonexistent", EngineTypeVLLM)
	err = s.Update(context.Background(), notExist)
	if err != ErrEngineNotFound {
		t.Errorf("expected ErrEngineNotFound, got %v", err)
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	s := NewMemoryStore()
	e := createTestEngine("ollama", EngineTypeOllama)
	_ = s.Create(context.Background(), e)

	err := s.Delete(context.Background(), "ollama")
	if err != nil {
		t.Errorf("Delete() failed: %v", err)
	}

	_, err = s.Get(context.Background(), "ollama")
	if err != ErrEngineNotFound {
		t.Errorf("expected ErrEngineNotFound after delete, got %v", err)
	}

	// Delete non-existent should fail
	err = s.Delete(context.Background(), "nonexistent")
	if err != ErrEngineNotFound {
		t.Errorf("expected ErrEngineNotFound, got %v", err)
	}
}

func TestMemoryStore_List_WithFilters(t *testing.T) {
	s := NewMemoryStore()

	e1 := createTestEngine("ollama", EngineTypeOllama)
	e2 := createTestEngine("vllm", EngineTypeVLLM)
	e3 := createTestEngine("whisper", EngineTypeWhisper)
	e3.Status = EngineStatusRunning

	_ = s.Create(context.Background(), e1)
	_ = s.Create(context.Background(), e2)
	_ = s.Create(context.Background(), e3)

	// Filter by type
	results, total, err := s.List(context.Background(), EngineFilter{Type: EngineTypeOllama})
	if err != nil {
		t.Errorf("List() failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	// Filter by status
	results, total, err = s.List(context.Background(), EngineFilter{Status: EngineStatusRunning})
	if err != nil {
		t.Errorf("List() failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}

	// Limit and offset
	results, total, err = s.List(context.Background(), EngineFilter{Limit: 1, Offset: 0})
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
	results, total, err = s.List(context.Background(), EngineFilter{Offset: 100})
	if err != nil {
		t.Errorf("List() failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for large offset, got %d", len(results))
	}

	// Limit+offset beyond end
	results, total, err = s.List(context.Background(), EngineFilter{Limit: 10, Offset: 2})
	if err != nil {
		t.Errorf("List() failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	_ = total
	_ = results
}
