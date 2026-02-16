# Resource Domain

资源管理领域。

## 源码映射

| AIMA | ASMS |
|------|------|
| `pkg/unit/resource/` | `pkg/resource/` |

## 原子单元

### Commands

| 名称 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `resource.allocate` | `{name, type, memory_bytes, gpu_fraction?, priority?}` | `{slot_id}` | 分配资源 |
| `resource.release` | `{slot_id}` | `{success}` | 释放资源 |
| `resource.update_slot` | `{slot_id, memory_limit?, status?}` | `{success}` | 更新槽位 |

### Queries

| 名称 | 输入 | 输出 | 说明 |
|------|------|------|------|
| `resource.status` | `{}` | `{memory, storage, slots: [], pressure}` | 资源状态 |
| `resource.budget` | `{}` | `{total, reserved, pools: {}}` | 资源预算 |
| `resource.allocations` | `{slot_id?, type?}` | `{allocations: []}` | 分配列表 |
| `resource.can_allocate` | `{memory_bytes, priority?}` | `{can_allocate, reason?}` | 检查可分配 |

## 核心结构

```go
type ResourceSlot struct {
    ID           string
    Name         string
    Type         SlotType      // inference_native, docker_container, system_service
    ModelType    model.ModelType
    MemoryLimit  uint64
    MemoryTarget uint64
    GPUFraction  float64
    CPUCores     float64
    Priority     int
    Preemptible  bool
    Persistent   bool
    Status       SlotStatus
    CurrentModel string
    ProcessPID   int
    ActualMemory uint64
}

type MemoryBudget struct {
    TotalBytes     uint64
    SystemReserved uint64
    ASMSReserved   uint64
    InferencePool  uint64
    ContainerPool  uint64
    BufferFlexible uint64
}
```

## 迁移状态

| 原子单元 | 状态 | ASMS 实现 |
|----------|------|-----------|
| `resource.allocate` | ✅ | `resource/manager.go` Allocate() |
| `resource.release` | ✅ | `resource/manager.go` Release() |
| `resource.status` | ✅ | `resource/manager.go` Status() |
| `resource.budget` | ✅ | `resource/manager.go` MemoryBudget |
| `resource.allocations` | ✅ | `resource/manager.go` ListSlots() |
| `resource.can_allocate` | ✅ | `resource/manager.go` CanAllocate() |
| `resource.update_slot` | 🔧 | 需完善 |
