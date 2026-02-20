package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/resource"
)

func TestResourceRepository_Create_Get(t *testing.T) {
	repo := NewResourceRepository()
	ctx := context.Background()

	t.Run("create and get slot", func(t *testing.T) {
		slot := &resource.ResourceSlot{
			ID:          "slot-1",
			Name:        "Test Slot",
			Type:        resource.SlotTypeInferenceNative,
			MemoryLimit: 8 * 1024 * 1024 * 1024,
			GPUFraction: 1.0,
			Priority:    5,
			Status:      resource.SlotStatusActive,
		}

		err := repo.CreateSlot(ctx, slot)
		if err != nil {
			t.Fatalf("CreateSlot failed: %v", err)
		}

		got, err := repo.GetSlot(ctx, "slot-1")
		if err != nil {
			t.Fatalf("GetSlot failed: %v", err)
		}
		if got.ID != "slot-1" || got.Name != "Test Slot" {
			t.Errorf("got %+v, want ID=slot-1, Name=Test Slot", got)
		}
	})

	t.Run("create duplicate fails", func(t *testing.T) {
		slot := &resource.ResourceSlot{
			ID:          "slot-1",
			Name:        "Duplicate Slot",
			Type:        resource.SlotTypeDockerContainer,
			MemoryLimit: 4 * 1024 * 1024 * 1024,
			GPUFraction: 0.5,
			Priority:    3,
			Status:      resource.SlotStatusIdle,
		}

		err := repo.CreateSlot(ctx, slot)
		if !errors.Is(err, resource.ErrSlotAlreadyExists) {
			t.Errorf("expected ErrSlotAlreadyExists, got %v", err)
		}
	})

	t.Run("get non-existent slot", func(t *testing.T) {
		_, err := repo.GetSlot(ctx, "nonexistent")
		if !errors.Is(err, resource.ErrSlotNotFound) {
			t.Errorf("expected ErrSlotNotFound, got %v", err)
		}
	})
}

func TestResourceRepository_List(t *testing.T) {
	repo := NewResourceRepository()
	ctx := context.Background()

	t.Run("list with type filter", func(t *testing.T) {
		// Create 2 native slots and 1 docker slot for this test
		_ = repo.CreateSlot(ctx, &resource.ResourceSlot{
			ID:          "slot-native-1",
			Name:        "Native Slot 1",
			Type:        resource.SlotTypeInferenceNative,
			MemoryLimit: 8 * 1024 * 1024 * 1024,
			GPUFraction: 1.0,
			Status:      resource.SlotStatusActive,
		})
		_ = repo.CreateSlot(ctx, &resource.ResourceSlot{
			ID:          "slot-native-2",
			Name:        "Native Slot 2",
			Type:        resource.SlotTypeInferenceNative,
			MemoryLimit: 16 * 1024 * 1024 * 1024,
			GPUFraction: 1.0,
			Status:      resource.SlotStatusIdle,
		})
		_ = repo.CreateSlot(ctx, &resource.ResourceSlot{
			ID:          "slot-docker-1",
			Name:        "Docker Slot 1",
			Type:        resource.SlotTypeDockerContainer,
			MemoryLimit: 8 * 1024 * 1024 * 1024,
			GPUFraction: 0.5,
			Status:      resource.SlotStatusActive,
		})

		nativeSlots, total, err := repo.ListSlots(ctx, resource.SlotFilter{Type: resource.SlotTypeInferenceNative})
		if err != nil {
			t.Fatalf("ListSlots failed: %v", err)
		}
		if total != 2 {
			t.Errorf("expected total 2 native slots, got %d", total)
		}
		for _, s := range nativeSlots {
			if s.Type != resource.SlotTypeInferenceNative {
				t.Errorf("expected native type, got %s", s.Type)
			}
		}
	})

	t.Run("list with slot id filter", func(t *testing.T) {
		// Create a unique slot for this test
		_ = repo.CreateSlot(ctx, &resource.ResourceSlot{
			ID:          "slot-filter-test",
			Name:        "Filter Test Slot",
			Type:        resource.SlotTypeInferenceNative,
			MemoryLimit: 4 * 1024 * 1024 * 1024,
			GPUFraction: 0.5,
			Status:      resource.SlotStatusActive,
		})

		slots, total, err := repo.ListSlots(ctx, resource.SlotFilter{SlotID: "slot-filter-test"})
		if err != nil {
			t.Fatalf("ListSlots failed: %v", err)
		}
		if total != 1 {
			t.Errorf("expected total 1 slot, got %d", total)
		}
		if len(slots) != 1 || slots[0].ID != "slot-filter-test" {
			t.Errorf("expected slot-filter-test, got %+v", slots)
		}
	})

	t.Run("list all slots", func(t *testing.T) {
		// Clear and create exactly 3 slots for this test
		_ = repo.DeleteSlot(ctx, "slot-native-1")
		_ = repo.DeleteSlot(ctx, "slot-native-2")
		_ = repo.DeleteSlot(ctx, "slot-docker-1")
		_ = repo.DeleteSlot(ctx, "slot-filter-test")

		_ = repo.CreateSlot(ctx, &resource.ResourceSlot{
			ID:          "slot-all-1",
			Name:        "Slot 1",
			Type:        resource.SlotTypeInferenceNative,
			MemoryLimit: 8 * 1024 * 1024 * 1024,
			GPUFraction: 1.0,
			Status:      resource.SlotStatusActive,
		})
		_ = repo.CreateSlot(ctx, &resource.ResourceSlot{
			ID:          "slot-all-2",
			Name:        "Slot 2",
			Type:        resource.SlotTypeDockerContainer,
			MemoryLimit: 4 * 1024 * 1024 * 1024,
			GPUFraction: 0.5,
			Status:      resource.SlotStatusIdle,
		})
		_ = repo.CreateSlot(ctx, &resource.ResourceSlot{
			ID:          "slot-all-3",
			Name:        "Slot 3",
			Type:        resource.SlotTypeSystemService,
			MemoryLimit: 2 * 1024 * 1024 * 1024,
			GPUFraction: 0.0,
			Status:      resource.SlotStatusActive,
		})

		slots, total, err := repo.ListSlots(ctx, resource.SlotFilter{})
		if err != nil {
			t.Fatalf("ListSlots failed: %v", err)
		}
		if total != 3 {
			t.Errorf("expected total 3 slots, got %d", total)
		}
		if len(slots) != 3 {
			t.Errorf("expected 3 slots, got %d", len(slots))
		}
	})
}

func TestResourceRepository_Update_Delete(t *testing.T) {
	repo := NewResourceRepository()
	ctx := context.Background()

	t.Run("update existing slot", func(t *testing.T) {
		slot := &resource.ResourceSlot{
			ID:          "slot-update",
			Name:        "Original Name",
			Type:        resource.SlotTypeInferenceNative,
			MemoryLimit: 8 * 1024 * 1024 * 1024,
			GPUFraction: 1.0,
			Priority:    5,
			Status:      resource.SlotStatusActive,
		}
		_ = repo.CreateSlot(ctx, slot)

		slot.Name = "Updated Name"
		slot.Status = resource.SlotStatusPreempted
		err := repo.UpdateSlot(ctx, slot)
		if err != nil {
			t.Fatalf("UpdateSlot failed: %v", err)
		}

		got, _ := repo.GetSlot(ctx, "slot-update")
		if got.Name != "Updated Name" {
			t.Errorf("name not updated, got %s", got.Name)
		}
		if got.Status != resource.SlotStatusPreempted {
			t.Errorf("status not updated, got %s", got.Status)
		}
	})

	t.Run("update non-existent slot fails", func(t *testing.T) {
		slot := &resource.ResourceSlot{
			ID:          "nonexistent",
			Name:        "Nonexistent",
			Type:        resource.SlotTypeInferenceNative,
			MemoryLimit: 8 * 1024 * 1024 * 1024,
			GPUFraction: 1.0,
			Priority:    5,
			Status:      resource.SlotStatusActive,
		}
		err := repo.UpdateSlot(ctx, slot)
		if !errors.Is(err, resource.ErrSlotNotFound) {
			t.Errorf("expected ErrSlotNotFound, got %v", err)
		}
	})

	t.Run("delete existing slot", func(t *testing.T) {
		slot := &resource.ResourceSlot{
			ID:          "slot-delete",
			Name:        "To Delete",
			Type:        resource.SlotTypeSystemService,
			MemoryLimit: 4 * 1024 * 1024 * 1024,
			GPUFraction: 0.0,
			Priority:    3,
			Status:      resource.SlotStatusIdle,
		}
		_ = repo.CreateSlot(ctx, slot)

		err := repo.DeleteSlot(ctx, "slot-delete")
		if err != nil {
			t.Fatalf("DeleteSlot failed: %v", err)
		}

		_, err = repo.GetSlot(ctx, "slot-delete")
		if !errors.Is(err, resource.ErrSlotNotFound) {
			t.Error("slot should have been deleted")
		}
	})

	t.Run("delete non-existent slot fails", func(t *testing.T) {
		err := repo.DeleteSlot(ctx, "nonexistent")
		if !errors.Is(err, resource.ErrSlotNotFound) {
			t.Errorf("expected ErrSlotNotFound, got %v", err)
		}
	})
}
