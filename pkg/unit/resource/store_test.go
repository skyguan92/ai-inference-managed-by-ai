package resource

import (
	"context"
	"testing"
)

func TestMemoryStore_CreateSlot(t *testing.T) {
	s := NewMemoryStore()
	slot := createTestSlot("slot-1", "test-slot", SlotTypeInferenceNative)

	err := s.CreateSlot(context.Background(), slot)
	if err != nil {
		t.Errorf("CreateSlot() failed: %v", err)
	}

	// Duplicate create should fail
	err = s.CreateSlot(context.Background(), slot)
	if err != ErrSlotAlreadyExists {
		t.Errorf("expected ErrSlotAlreadyExists, got %v", err)
	}
}

func TestMemoryStore_GetSlot(t *testing.T) {
	s := NewMemoryStore()
	slot := createTestSlot("slot-1", "test-slot", SlotTypeInferenceNative)
	s.CreateSlot(context.Background(), slot)

	got, err := s.GetSlot(context.Background(), "slot-1")
	if err != nil {
		t.Errorf("GetSlot() failed: %v", err)
	}
	if got.Name != "test-slot" {
		t.Errorf("expected Name 'test-slot', got %q", got.Name)
	}

	_, err = s.GetSlot(context.Background(), "nonexistent")
	if err != ErrSlotNotFound {
		t.Errorf("expected ErrSlotNotFound, got %v", err)
	}
}

func TestMemoryStore_UpdateSlot(t *testing.T) {
	s := NewMemoryStore()
	slot := createTestSlot("slot-1", "test-slot", SlotTypeInferenceNative)
	s.CreateSlot(context.Background(), slot)

	slot.Status = SlotStatusIdle
	err := s.UpdateSlot(context.Background(), slot)
	if err != nil {
		t.Errorf("UpdateSlot() failed: %v", err)
	}

	got, _ := s.GetSlot(context.Background(), "slot-1")
	if got.Status != SlotStatusIdle {
		t.Errorf("expected Status %q, got %q", SlotStatusIdle, got.Status)
	}

	// Update non-existent should fail
	notExist := createTestSlot("nonexistent", "test", SlotTypeDockerContainer)
	err = s.UpdateSlot(context.Background(), notExist)
	if err != ErrSlotNotFound {
		t.Errorf("expected ErrSlotNotFound, got %v", err)
	}
}

func TestMemoryStore_DeleteSlot(t *testing.T) {
	s := NewMemoryStore()
	slot := createTestSlot("slot-1", "test-slot", SlotTypeInferenceNative)
	s.CreateSlot(context.Background(), slot)

	err := s.DeleteSlot(context.Background(), "slot-1")
	if err != nil {
		t.Errorf("DeleteSlot() failed: %v", err)
	}

	_, err = s.GetSlot(context.Background(), "slot-1")
	if err != ErrSlotNotFound {
		t.Errorf("expected ErrSlotNotFound after delete, got %v", err)
	}

	// Delete non-existent should fail
	err = s.DeleteSlot(context.Background(), "nonexistent")
	if err != ErrSlotNotFound {
		t.Errorf("expected ErrSlotNotFound, got %v", err)
	}
}

func TestMemoryStore_ListSlots_WithFilters(t *testing.T) {
	s := NewMemoryStore()

	s1 := createTestSlot("slot-1", "inference-slot", SlotTypeInferenceNative)
	s2 := createTestSlot("slot-2", "docker-slot", SlotTypeDockerContainer)
	s3 := createTestSlot("slot-3", "system-slot", SlotTypeSystemService)

	s.CreateSlot(context.Background(), s1)
	s.CreateSlot(context.Background(), s2)
	s.CreateSlot(context.Background(), s3)

	// List all
	results, total, err := s.ListSlots(context.Background(), SlotFilter{})
	if err != nil {
		t.Errorf("ListSlots() failed: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// Filter by type
	results, total, err = s.ListSlots(context.Background(), SlotFilter{Type: SlotTypeInferenceNative})
	if err != nil {
		t.Errorf("ListSlots() failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	// Filter by slot ID
	results, total, err = s.ListSlots(context.Background(), SlotFilter{SlotID: "slot-2"})
	if err != nil {
		t.Errorf("ListSlots() failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if results[0].ID != "slot-2" {
		t.Errorf("expected slot-2, got %q", results[0].ID)
	}
}

func TestSlotTypeValidation(t *testing.T) {
	if !IsValidSlotType(SlotTypeInferenceNative) {
		t.Error("SlotTypeInferenceNative should be valid")
	}
	if !IsValidSlotType(SlotTypeDockerContainer) {
		t.Error("SlotTypeDockerContainer should be valid")
	}
	if !IsValidSlotType(SlotTypeSystemService) {
		t.Error("SlotTypeSystemService should be valid")
	}
	if IsValidSlotType("invalid_type") {
		t.Error("'invalid_type' should not be valid")
	}
}

func TestSlotStatusValidation(t *testing.T) {
	if !IsValidSlotStatus(SlotStatusActive) {
		t.Error("SlotStatusActive should be valid")
	}
	if !IsValidSlotStatus(SlotStatusIdle) {
		t.Error("SlotStatusIdle should be valid")
	}
	if !IsValidSlotStatus(SlotStatusPreempted) {
		t.Error("SlotStatusPreempted should be valid")
	}
	if IsValidSlotStatus("invalid_status") {
		t.Error("'invalid_status' should not be valid")
	}
}

func TestValidSlotTypes(t *testing.T) {
	types := ValidSlotTypes()
	if len(types) != 3 {
		t.Errorf("expected 3 valid slot types, got %d", len(types))
	}
}

func TestValidSlotStatuses(t *testing.T) {
	statuses := ValidSlotStatuses()
	if len(statuses) != 3 {
		t.Errorf("expected 3 valid slot statuses, got %d", len(statuses))
	}
}
