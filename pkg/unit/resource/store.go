package resource

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ResourceStore interface {
	CreateSlot(ctx context.Context, slot *ResourceSlot) error
	GetSlot(ctx context.Context, slotID string) (*ResourceSlot, error)
	ListSlots(ctx context.Context, filter SlotFilter) ([]ResourceSlot, int, error)
	DeleteSlot(ctx context.Context, slotID string) error
	UpdateSlot(ctx context.Context, slot *ResourceSlot) error
}

type ResourceProvider interface {
	GetStatus(ctx context.Context) (*ResourceStatus, error)
	GetBudget(ctx context.Context) (*ResourceBudget, error)
	CanAllocate(ctx context.Context, memoryBytes uint64, priority int) (*CanAllocateResult, error)
}

type MemoryStore struct {
	slots map[string]*ResourceSlot
	mu    sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		slots: make(map[string]*ResourceSlot),
	}
}

func (s *MemoryStore) CreateSlot(ctx context.Context, slot *ResourceSlot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.slots[slot.ID]; exists {
		return ErrSlotAlreadyExists
	}

	s.slots[slot.ID] = slot
	return nil
}

func (s *MemoryStore) GetSlot(ctx context.Context, slotID string) (*ResourceSlot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	slot, exists := s.slots[slotID]
	if !exists {
		return nil, ErrSlotNotFound
	}
	return slot, nil
}

func (s *MemoryStore) ListSlots(ctx context.Context, filter SlotFilter) ([]ResourceSlot, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []ResourceSlot
	for _, slot := range s.slots {
		if filter.Type != "" && slot.Type != filter.Type {
			continue
		}
		if filter.SlotID != "" && slot.ID != filter.SlotID {
			continue
		}
		result = append(result, *slot)
	}

	return result, len(result), nil
}

func (s *MemoryStore) DeleteSlot(ctx context.Context, slotID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.slots[slotID]; !exists {
		return ErrSlotNotFound
	}

	delete(s.slots, slotID)
	return nil
}

func (s *MemoryStore) UpdateSlot(ctx context.Context, slot *ResourceSlot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.slots[slot.ID]; !exists {
		return ErrSlotNotFound
	}

	s.slots[slot.ID] = slot
	return nil
}

type MockProvider struct {
	status         *ResourceStatus
	budget         *ResourceBudget
	canAllocateRes *CanAllocateResult
	canAllocateErr error
	statusErr      error
	budgetErr      error
}

func (m *MockProvider) GetStatus(ctx context.Context) (*ResourceStatus, error) {
	if m.statusErr != nil {
		return nil, m.statusErr
	}
	if m.status != nil {
		return m.status, nil
	}
	return &ResourceStatus{
		Memory: MemoryInfo{
			Total:     64 * 1024 * 1024 * 1024,
			Used:      32 * 1024 * 1024 * 1024,
			Available: 32 * 1024 * 1024 * 1024,
		},
		Storage: StorageInfo{
			Total:     1024 * 1024 * 1024 * 1024,
			Used:      512 * 1024 * 1024 * 1024,
			Available: 512 * 1024 * 1024 * 1024,
		},
		Slots:    []ResourceSlot{},
		Pressure: PressureLevelLow,
	}, nil
}

func (m *MockProvider) GetBudget(ctx context.Context) (*ResourceBudget, error) {
	if m.budgetErr != nil {
		return nil, m.budgetErr
	}
	if m.budget != nil {
		return m.budget, nil
	}
	return &ResourceBudget{
		Total:    64 * 1024 * 1024 * 1024,
		Reserved: 16 * 1024 * 1024 * 1024,
		Pools: map[string]ResourcePool{
			"inference": {
				Name:      "inference",
				Total:     48 * 1024 * 1024 * 1024,
				Reserved:  8 * 1024 * 1024 * 1024,
				Available: 40 * 1024 * 1024 * 1024,
			},
			"system": {
				Name:      "system",
				Total:     16 * 1024 * 1024 * 1024,
				Reserved:  8 * 1024 * 1024 * 1024,
				Available: 8 * 1024 * 1024 * 1024,
			},
		},
	}, nil
}

func (m *MockProvider) CanAllocate(ctx context.Context, memoryBytes uint64, priority int) (*CanAllocateResult, error) {
	if m.canAllocateErr != nil {
		return nil, m.canAllocateErr
	}
	if m.canAllocateRes != nil {
		return m.canAllocateRes, nil
	}
	return &CanAllocateResult{
		CanAllocate: true,
	}, nil
}

func createTestSlot(id, name string, slotType SlotType) *ResourceSlot {
	now := time.Now().Unix()
	return &ResourceSlot{
		ID:          id,
		Name:        name,
		Type:        slotType,
		MemoryLimit: 8 * 1024 * 1024 * 1024,
		GPUFraction: 1.0,
		Priority:    5,
		Status:      SlotStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func generateSlotID() string {
	return "slot-" + uuid.New().String()[:8]
}
