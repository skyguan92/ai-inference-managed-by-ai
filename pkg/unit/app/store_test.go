package app

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- NewMemoryStore ---

func TestNewMemoryStore_NotNil(t *testing.T) {
	s := NewMemoryStore()
	require.NotNil(t, s)
}

// --- Create ---

func TestMemoryStore_Create_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	app := createTestApp("app-1", "open-webui", AppStatusInstalled)
	err := s.Create(ctx, app)
	require.NoError(t, err)
}

func TestMemoryStore_Create_Duplicate_ReturnsError(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	app := createTestApp("app-1", "open-webui", AppStatusInstalled)
	require.NoError(t, s.Create(ctx, app))

	err := s.Create(ctx, app)
	assert.ErrorIs(t, err, ErrAppAlreadyExists)
}

// --- Get ---

func TestMemoryStore_Get_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	app := createTestApp("app-1", "grafana", AppStatusRunning)
	require.NoError(t, s.Create(ctx, app))

	got, err := s.Get(ctx, "app-1")
	require.NoError(t, err)
	assert.Equal(t, "app-1", got.ID)
	assert.Equal(t, "grafana", got.Template)
	assert.Equal(t, AppStatusRunning, got.Status)
}

func TestMemoryStore_Get_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	_, err := s.Get(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrAppNotFound)
}

// --- Update ---

func TestMemoryStore_Update_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	app := createTestApp("app-1", "open-webui", AppStatusInstalled)
	require.NoError(t, s.Create(ctx, app))

	app.Status = AppStatusRunning
	require.NoError(t, s.Update(ctx, app))

	got, err := s.Get(ctx, "app-1")
	require.NoError(t, err)
	assert.Equal(t, AppStatusRunning, got.Status)
}

func TestMemoryStore_Update_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	app := createTestApp("nonexistent", "open-webui", AppStatusRunning)
	err := s.Update(ctx, app)
	assert.ErrorIs(t, err, ErrAppNotFound)
}

// --- Delete ---

func TestMemoryStore_Delete_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	app := createTestApp("app-1", "grafana", AppStatusStopped)
	require.NoError(t, s.Create(ctx, app))

	require.NoError(t, s.Delete(ctx, "app-1"))

	_, err := s.Get(ctx, "app-1")
	assert.ErrorIs(t, err, ErrAppNotFound)
}

func TestMemoryStore_Delete_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	err := s.Delete(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrAppNotFound)
}

// --- List ---

func TestMemoryStore_List_All(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.Create(ctx, createTestApp("app-1", "open-webui", AppStatusRunning)))
	require.NoError(t, s.Create(ctx, createTestApp("app-2", "grafana", AppStatusStopped)))
	require.NoError(t, s.Create(ctx, createTestApp("app-3", "open-webui", AppStatusInstalled)))

	apps, total, err := s.List(ctx, AppFilter{})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, apps, 3)
}

func TestMemoryStore_List_FilterByStatus(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.Create(ctx, createTestApp("app-1", "open-webui", AppStatusRunning)))
	require.NoError(t, s.Create(ctx, createTestApp("app-2", "grafana", AppStatusStopped)))
	require.NoError(t, s.Create(ctx, createTestApp("app-3", "open-webui", AppStatusRunning)))

	apps, total, err := s.List(ctx, AppFilter{Status: AppStatusRunning})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, apps, 2)
	for _, a := range apps {
		assert.Equal(t, AppStatusRunning, a.Status)
	}
}

func TestMemoryStore_List_FilterByTemplate(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.Create(ctx, createTestApp("app-1", "open-webui", AppStatusRunning)))
	require.NoError(t, s.Create(ctx, createTestApp("app-2", "grafana", AppStatusRunning)))

	apps, total, err := s.List(ctx, AppFilter{Template: "grafana"})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "grafana", apps[0].Template)
}

func TestMemoryStore_List_Limit(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		id := "app-" + string(rune('0'+i))
		require.NoError(t, s.Create(ctx, createTestApp(id, "open-webui", AppStatusRunning)))
	}

	apps, total, err := s.List(ctx, AppFilter{Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, apps, 2)
}

func TestMemoryStore_List_Offset(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		id := "app-" + string(rune('0'+i))
		require.NoError(t, s.Create(ctx, createTestApp(id, "open-webui", AppStatusRunning)))
	}

	apps, total, err := s.List(ctx, AppFilter{Offset: 3})
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Len(t, apps, 1)
}

func TestMemoryStore_List_OffsetBeyondLength(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.Create(ctx, createTestApp("app-1", "open-webui", AppStatusRunning)))

	apps, total, err := s.List(ctx, AppFilter{Offset: 100})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Empty(t, apps)
}

func TestMemoryStore_List_Empty(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	apps, total, err := s.List(ctx, AppFilter{})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, apps)
}

// --- Concurrent access ---

func TestMemoryStore_ConcurrentOps_NoRace(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	// Pre-seed
	for i := 0; i < 5; i++ {
		id := "app-" + string(rune('0'+i))
		_ = s.Create(ctx, createTestApp(id, "open-webui", AppStatusRunning))
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = s.List(ctx, AppFilter{})
			_, _ = s.Get(ctx, "app-0")
		}()
	}
	wg.Wait()
}
