package repositories

import (
	"context"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/app"
)

// AppRepository 是基于内存的 App 存储实现
type AppRepository struct {
	store *app.MemoryStore
}

// NewAppRepository 创建一个新的 AppRepository 实例
func NewAppRepository() *AppRepository {
	return &AppRepository{
		store: app.NewMemoryStore(),
	}
}

// Create 创建应用记录
func (r *AppRepository) Create(ctx context.Context, a *app.App) error {
	return r.store.Create(ctx, a)
}

// Get 根据 ID 获取应用
func (r *AppRepository) Get(ctx context.Context, id string) (*app.App, error) {
	return r.store.Get(ctx, id)
}

// List 列出应用（支持过滤和分页）
func (r *AppRepository) List(ctx context.Context, filter app.AppFilter) ([]app.App, int, error) {
	return r.store.List(ctx, filter)
}

// Delete 删除应用
func (r *AppRepository) Delete(ctx context.Context, id string) error {
	return r.store.Delete(ctx, id)
}

// Update 更新应用
func (r *AppRepository) Update(ctx context.Context, a *app.App) error {
	return r.store.Update(ctx, a)
}

// 确保 AppRepository 实现了 AppStore 接口
var _ app.AppStore = (*AppRepository)(nil)
