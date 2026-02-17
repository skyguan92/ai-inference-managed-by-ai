package repositories

import (
	"context"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
)

// ModelRepository 是基于内存的 Model 存储实现
type ModelRepository struct {
	store *model.MemoryStore
}

// NewModelRepository 创建一个新的 ModelRepository 实例
func NewModelRepository() *ModelRepository {
	return &ModelRepository{
		store: model.NewMemoryStore(),
	}
}

// Create 创建模型记录
func (r *ModelRepository) Create(ctx context.Context, m *model.Model) error {
	return r.store.Create(ctx, m)
}

// Get 根据 ID 获取模型
func (r *ModelRepository) Get(ctx context.Context, id string) (*model.Model, error) {
	return r.store.Get(ctx, id)
}

// List 列出模型（支持过滤和分页）
func (r *ModelRepository) List(ctx context.Context, filter model.ModelFilter) ([]model.Model, int, error) {
	return r.store.List(ctx, filter)
}

// Delete 删除模型
func (r *ModelRepository) Delete(ctx context.Context, id string) error {
	return r.store.Delete(ctx, id)
}

// Update 更新模型
func (r *ModelRepository) Update(ctx context.Context, m *model.Model) error {
	return r.store.Update(ctx, m)
}

// 确保 ModelRepository 实现了 ModelStore 接口
var _ model.ModelStore = (*ModelRepository)(nil)
