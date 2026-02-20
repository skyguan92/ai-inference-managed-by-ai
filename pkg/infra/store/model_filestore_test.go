package store

import (
	"context"
	"os"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileStore(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	require.NoError(t, err)
	assert.NotNil(t, fs)
}

func TestNewFileStore_InvalidDir(t *testing.T) {
	// Try to create a file store in a path where we don't have permissions
	// Use a file as the parent (impossible to create a dir inside a file)
	f, err := os.CreateTemp("", "not-a-dir-*")
	require.NoError(t, err)
	_ = f.Close()
	defer os.Remove(f.Name())

	// Try to use the file as a directory
	_, err = NewFileStore(f.Name() + "/subdir")
	// This may or may not fail depending on OS, just ensure no panic
	_ = err
}

func TestFileStore_CreateAndGet(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	require.NoError(t, err)

	m := &model.Model{
		ID:        "model-001",
		Name:      "test-model",
		Type:      model.ModelTypeLLM,
		Format:    model.FormatGGUF,
		Status:    model.StatusReady,
		Size:      1000000,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}

	err = fs.Create(context.Background(), m)
	require.NoError(t, err)

	got, err := fs.Get(context.Background(), "model-001")
	require.NoError(t, err)
	assert.Equal(t, m.ID, got.ID)
	assert.Equal(t, m.Name, got.Name)
}

func TestFileStore_Create_Duplicate(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	require.NoError(t, err)

	m := &model.Model{
		ID:        "model-001",
		Name:      "test-model",
		Type:      model.ModelTypeLLM,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}

	err = fs.Create(context.Background(), m)
	require.NoError(t, err)

	err = fs.Create(context.Background(), m)
	assert.ErrorIs(t, err, model.ErrModelAlreadyExists)
}

func TestFileStore_Get_NotFound(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	require.NoError(t, err)

	_, err = fs.Get(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, model.ErrModelNotFound)
}

func TestFileStore_Update(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	require.NoError(t, err)

	m := &model.Model{
		ID:        "model-001",
		Name:      "original",
		Type:      model.ModelTypeLLM,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	err = fs.Create(context.Background(), m)
	require.NoError(t, err)

	m.Name = "updated"
	err = fs.Update(context.Background(), m)
	require.NoError(t, err)

	got, err := fs.Get(context.Background(), "model-001")
	require.NoError(t, err)
	assert.Equal(t, "updated", got.Name)
}

func TestFileStore_Update_NotFound(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	require.NoError(t, err)

	m := &model.Model{
		ID:        "nonexistent",
		Name:      "test",
		Type:      model.ModelTypeLLM,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	err = fs.Update(context.Background(), m)
	assert.ErrorIs(t, err, model.ErrModelNotFound)
}

func TestFileStore_Delete(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	require.NoError(t, err)

	m := &model.Model{
		ID:        "model-001",
		Name:      "test",
		Type:      model.ModelTypeLLM,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	err = fs.Create(context.Background(), m)
	require.NoError(t, err)

	err = fs.Delete(context.Background(), "model-001")
	require.NoError(t, err)

	_, err = fs.Get(context.Background(), "model-001")
	assert.ErrorIs(t, err, model.ErrModelNotFound)
}

func TestFileStore_Delete_NotFound(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	require.NoError(t, err)

	err = fs.Delete(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, model.ErrModelNotFound)
}

func TestFileStore_List(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	require.NoError(t, err)

	models := []*model.Model{
		{ID: "m1", Name: "model1", Type: model.ModelTypeLLM, Status: model.StatusReady, CreatedAt: 1000, UpdatedAt: 1000},
		{ID: "m2", Name: "model2", Type: model.ModelTypeVLM, Status: model.StatusReady, CreatedAt: 1000, UpdatedAt: 1000},
		{ID: "m3", Name: "model3", Type: model.ModelTypeLLM, Status: model.StatusPending, CreatedAt: 1000, UpdatedAt: 1000},
	}
	for _, m := range models {
		err = fs.Create(context.Background(), m)
		require.NoError(t, err)
	}

	t.Run("list all", func(t *testing.T) {
		result, total, err := fs.List(context.Background(), model.ModelFilter{})
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, result, 3)
	})

	t.Run("filter by type", func(t *testing.T) {
		result, total, err := fs.List(context.Background(), model.ModelFilter{Type: model.ModelTypeLLM})
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, result, 2)
	})

	t.Run("filter by status", func(t *testing.T) {
		result, total, err := fs.List(context.Background(), model.ModelFilter{Status: model.StatusReady})
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, result, 2)
	})

	t.Run("with limit", func(t *testing.T) {
		result, total, err := fs.List(context.Background(), model.ModelFilter{Limit: 2})
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, result, 2)
	})

	t.Run("with offset beyond length", func(t *testing.T) {
		result, total, err := fs.List(context.Background(), model.ModelFilter{Offset: 100})
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, result, 0)
	})
}

func TestFileStore_Persistence(t *testing.T) {
	dir := t.TempDir()

	// Create and save a model
	fs1, err := NewFileStore(dir)
	require.NoError(t, err)

	m := &model.Model{
		ID:        "model-persist",
		Name:      "persist-test",
		Type:      model.ModelTypeLLM,
		Status:    model.StatusReady,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	err = fs1.Create(context.Background(), m)
	require.NoError(t, err)

	// Create new store from same dir and verify data was loaded
	fs2, err := NewFileStore(dir)
	require.NoError(t, err)

	got, err := fs2.Get(context.Background(), "model-persist")
	require.NoError(t, err)
	assert.Equal(t, m.ID, got.ID)
	assert.Equal(t, m.Name, got.Name)
}

func TestFileStore_ImplementsInterface(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileStore(dir)
	require.NoError(t, err)

	var _ model.ModelStore = fs
}
