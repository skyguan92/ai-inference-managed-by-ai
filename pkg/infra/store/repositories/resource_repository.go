package repositories

import (
	"context"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/resource"
)

// ResourceRepository 是基于内存的 Resource 存储实现
type ResourceRepository struct {
	store *resource.MemoryStore
}

// NewResourceRepository 创建一个新的 ResourceRepository 实例
func NewResourceRepository() *ResourceRepository {
	return &ResourceRepository{
		store: resource.NewMemoryStore(),
	}
}

// CreateSlot 创建资源槽
func (r *ResourceRepository) CreateSlot(ctx context.Context, slot *resource.ResourceSlot) error {
	return r.store.CreateSlot(ctx, slot)
}

// GetSlot 根据 ID 获取资源槽
func (r *ResourceRepository) GetSlot(ctx context.Context, slotID string) (*resource.ResourceSlot, error) {
	return r.store.GetSlot(ctx, slotID)
}

// ListSlots 列出资源槽（支持过滤）
func (r *ResourceRepository) ListSlots(ctx context.Context, filter resource.SlotFilter) ([]resource.ResourceSlot, int, error) {
	return r.store.ListSlots(ctx, filter)
}

// DeleteSlot 删除资源槽
func (r *ResourceRepository) DeleteSlot(ctx context.Context, slotID string) error {
	return r.store.DeleteSlot(ctx, slotID)
}

// UpdateSlot 更新资源槽
func (r *ResourceRepository) UpdateSlot(ctx context.Context, slot *resource.ResourceSlot) error {
	return r.store.UpdateSlot(ctx, slot)
}

// 确保 ResourceRepository 实现了 ResourceStore 接口
var _ resource.ResourceStore = (*ResourceRepository)(nil)
