package service

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

	svc := createTestService("svc-1", "model-llm", ServiceStatusRunning)
	err := s.Create(ctx, svc)
	require.NoError(t, err)
}

func TestMemoryStore_Create_Duplicate_ReturnsError(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	svc := createTestService("svc-1", "model-llm", ServiceStatusRunning)
	require.NoError(t, s.Create(ctx, svc))

	err := s.Create(ctx, svc)
	assert.ErrorIs(t, err, ErrServiceAlreadyExists)
}

// --- Get ---

func TestMemoryStore_Get_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	svc := createTestService("svc-1", "model-llm", ServiceStatusRunning)
	require.NoError(t, s.Create(ctx, svc))

	got, err := s.Get(ctx, "svc-1")
	require.NoError(t, err)
	assert.Equal(t, "svc-1", got.ID)
	assert.Equal(t, "model-llm", got.ModelID)
	assert.Equal(t, ServiceStatusRunning, got.Status)
}

func TestMemoryStore_Get_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	_, err := s.Get(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrServiceNotFound)
}

// --- GetByName ---

func TestMemoryStore_GetByName_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	svc := createTestService("svc-1", "model-llm", ServiceStatusRunning)
	require.NoError(t, s.Create(ctx, svc))

	got, err := s.GetByName(ctx, svc.Name)
	require.NoError(t, err)
	assert.Equal(t, "svc-1", got.ID)
}

func TestMemoryStore_GetByName_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	_, err := s.GetByName(ctx, "nonexistent-name")
	assert.ErrorIs(t, err, ErrServiceNotFound)
}

func TestMemoryStore_GetByName_ReturnsFirstMatch(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	svc := createTestService("svc-1", "model-llm", ServiceStatusRunning)
	svc.Name = "unique-name"
	require.NoError(t, s.Create(ctx, svc))

	got, err := s.GetByName(ctx, "unique-name")
	require.NoError(t, err)
	assert.Equal(t, "unique-name", got.Name)
}

// --- Update ---

func TestMemoryStore_Update_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	svc := createTestService("svc-1", "model-llm", ServiceStatusCreating)
	require.NoError(t, s.Create(ctx, svc))

	svc.Status = ServiceStatusRunning
	svc.ActiveReplicas = 2
	require.NoError(t, s.Update(ctx, svc))

	got, err := s.Get(ctx, "svc-1")
	require.NoError(t, err)
	assert.Equal(t, ServiceStatusRunning, got.Status)
	assert.Equal(t, 2, got.ActiveReplicas)
}

func TestMemoryStore_Update_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	svc := createTestService("nonexistent", "model-llm", ServiceStatusRunning)
	err := s.Update(ctx, svc)
	assert.ErrorIs(t, err, ErrServiceNotFound)
}

// --- Delete ---

func TestMemoryStore_Delete_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	svc := createTestService("svc-1", "model-llm", ServiceStatusStopped)
	require.NoError(t, s.Create(ctx, svc))

	require.NoError(t, s.Delete(ctx, "svc-1"))

	_, err := s.Get(ctx, "svc-1")
	assert.ErrorIs(t, err, ErrServiceNotFound)
}

func TestMemoryStore_Delete_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	err := s.Delete(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrServiceNotFound)
}

// --- List ---

func TestMemoryStore_List_All(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.Create(ctx, createTestService("svc-1", "model-a", ServiceStatusRunning)))
	require.NoError(t, s.Create(ctx, createTestService("svc-2", "model-b", ServiceStatusStopped)))
	require.NoError(t, s.Create(ctx, createTestService("svc-3", "model-a", ServiceStatusFailed)))

	services, total, err := s.List(ctx, ServiceFilter{})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, services, 3)
}

func TestMemoryStore_List_FilterByStatus(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.Create(ctx, createTestService("svc-1", "model-a", ServiceStatusRunning)))
	require.NoError(t, s.Create(ctx, createTestService("svc-2", "model-b", ServiceStatusStopped)))
	require.NoError(t, s.Create(ctx, createTestService("svc-3", "model-c", ServiceStatusRunning)))

	services, total, err := s.List(ctx, ServiceFilter{Status: ServiceStatusRunning})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, services, 2)
	for _, svc := range services {
		assert.Equal(t, ServiceStatusRunning, svc.Status)
	}
}

func TestMemoryStore_List_FilterByModelID(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.Create(ctx, createTestService("svc-1", "model-llm", ServiceStatusRunning)))
	require.NoError(t, s.Create(ctx, createTestService("svc-2", "model-asr", ServiceStatusRunning)))

	services, total, err := s.List(ctx, ServiceFilter{ModelID: "model-llm"})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "model-llm", services[0].ModelID)
}

func TestMemoryStore_List_Limit(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		id := "svc-" + string(rune('0'+i))
		require.NoError(t, s.Create(ctx, createTestService(id, "model-llm", ServiceStatusRunning)))
	}

	services, total, err := s.List(ctx, ServiceFilter{Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, services, 2)
}

func TestMemoryStore_List_Offset(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		id := "svc-" + string(rune('0'+i))
		require.NoError(t, s.Create(ctx, createTestService(id, "model-llm", ServiceStatusRunning)))
	}

	services, total, err := s.List(ctx, ServiceFilter{Offset: 3})
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Len(t, services, 1)
}

func TestMemoryStore_List_Empty(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	services, total, err := s.List(ctx, ServiceFilter{})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, services)
}

// --- Concurrent access ---

func TestMemoryStore_ConcurrentOps_NoRace(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		id := "svc-" + string(rune('0'+i))
		_ = s.Create(ctx, createTestService(id, "model-llm", ServiceStatusRunning))
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = s.List(ctx, ServiceFilter{})
			_, _ = s.Get(ctx, "svc-0")
			_, _ = s.GetByName(ctx, "test-service-svc-0")
		}()
	}
	wg.Wait()
}
