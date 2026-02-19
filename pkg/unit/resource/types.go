package resource

// Domain errors are defined in errors.go

type SlotType string

const (
	SlotTypeInferenceNative SlotType = "inference_native"
	SlotTypeDockerContainer SlotType = "docker_container"
	SlotTypeSystemService   SlotType = "system_service"
)

type SlotStatus string

const (
	SlotStatusActive    SlotStatus = "active"
	SlotStatusIdle      SlotStatus = "idle"
	SlotStatusPreempted SlotStatus = "preempted"
)

type ResourceSlot struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Type        SlotType   `json:"type"`
	MemoryLimit uint64     `json:"memory_limit"`
	GPUFraction float64    `json:"gpu_fraction"`
	Priority    int        `json:"priority"`
	Status      SlotStatus `json:"status"`
	CreatedAt   int64      `json:"created_at"`
	UpdatedAt   int64      `json:"updated_at"`
}

type MemoryInfo struct {
	Total     uint64 `json:"total"`
	Used      uint64 `json:"used"`
	Available uint64 `json:"available"`
}

type StorageInfo struct {
	Total     uint64 `json:"total"`
	Used      uint64 `json:"used"`
	Available uint64 `json:"available"`
}

type PressureLevel string

const (
	PressureLevelLow      PressureLevel = "low"
	PressureLevelMedium   PressureLevel = "medium"
	PressureLevelHigh     PressureLevel = "high"
	PressureLevelCritical PressureLevel = "critical"
)

type ResourceStatus struct {
	Memory   MemoryInfo     `json:"memory"`
	Storage  StorageInfo    `json:"storage"`
	Slots    []ResourceSlot `json:"slots"`
	Pressure PressureLevel  `json:"pressure"`
}

type ResourcePool struct {
	Name      string `json:"name"`
	Total     uint64 `json:"total"`
	Reserved  uint64 `json:"reserved"`
	Available uint64 `json:"available"`
}

type ResourceBudget struct {
	Total    uint64                  `json:"total"`
	Reserved uint64                  `json:"reserved"`
	Pools    map[string]ResourcePool `json:"pools"`
}

type Allocation struct {
	SlotID      string     `json:"slot_id"`
	Name        string     `json:"name"`
	Type        SlotType   `json:"type"`
	MemoryUsed  uint64     `json:"memory_used"`
	GPUFraction float64    `json:"gpu_fraction"`
	Priority    int        `json:"priority"`
	Status      SlotStatus `json:"status"`
}

type AllocateResult struct {
	SlotID string `json:"slot_id"`
}

type ReleaseResult struct {
	Success bool `json:"success"`
}

type UpdateSlotResult struct {
	Success bool `json:"success"`
}

type CanAllocateResult struct {
	CanAllocate bool   `json:"can_allocate"`
	Reason      string `json:"reason,omitempty"`
}

type SlotFilter struct {
	SlotID string   `json:"slot_id,omitempty"`
	Type   SlotType `json:"type,omitempty"`
}

func ValidSlotTypes() []SlotType {
	return []SlotType{SlotTypeInferenceNative, SlotTypeDockerContainer, SlotTypeSystemService}
}

func ValidSlotStatuses() []SlotStatus {
	return []SlotStatus{SlotStatusActive, SlotStatusIdle, SlotStatusPreempted}
}

func IsValidSlotType(t SlotType) bool {
	for _, vt := range ValidSlotTypes() {
		if t == vt {
			return true
		}
	}
	return false
}

func IsValidSlotStatus(s SlotStatus) bool {
	for _, vs := range ValidSlotStatuses() {
		if s == vs {
			return true
		}
	}
	return false
}
