package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/engine"
)

func TestEngineRepository_Create_Get(t *testing.T) {
	repo := NewEngineRepository()
	ctx := context.Background()

	t.Run("create and get engine", func(t *testing.T) {
		e := &engine.Engine{
			ID:      "engine-1",
			Name:    "test-ollama",
			Type:    engine.EngineTypeOllama,
			Status:  engine.EngineStatusStopped,
			Version: "1.0.0",
		}

		err := repo.Create(ctx, e)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		got, err := repo.Get(ctx, "test-ollama")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if got.ID != "engine-1" || got.Name != "test-ollama" {
			t.Errorf("got %+v, want ID=engine-1, Name=test-ollama", got)
		}
	})

	t.Run("create duplicate fails", func(t *testing.T) {
		e := &engine.Engine{
			ID:      "engine-2",
			Name:    "test-ollama",
			Type:    engine.EngineTypeOllama,
			Status:  engine.EngineStatusStopped,
			Version: "1.0.0",
		}

		err := repo.Create(ctx, e)
		if !errors.Is(err, engine.ErrEngineAlreadyExists) {
			t.Errorf("expected ErrEngineAlreadyExists, got %v", err)
		}
	})

	t.Run("get non-existent engine", func(t *testing.T) {
		_, err := repo.Get(ctx, "nonexistent")
		if !errors.Is(err, engine.ErrEngineNotFound) {
			t.Errorf("expected ErrEngineNotFound, got %v", err)
		}
	})

	t.Run("get by id", func(t *testing.T) {
		got, err := repo.GetByID(ctx, "engine-1")
		if err != nil {
			t.Fatalf("GetByID failed: %v", err)
		}
		if got.ID != "engine-1" {
			t.Errorf("got ID %s, want engine-1", got.ID)
		}
	})
}

func TestEngineRepository_List(t *testing.T) {
	repo := NewEngineRepository()
	ctx := context.Background()

	t.Run("list with type filter", func(t *testing.T) {
		_ = repo.Create(ctx, &engine.Engine{
			ID:     "engine-3",
			Name:   "test-ollama-2",
			Type:   engine.EngineTypeOllama,
			Status: engine.EngineStatusRunning,
		})
		_ = repo.Create(ctx, &engine.Engine{
			ID:     "engine-4",
			Name:   "test-vllm",
			Type:   engine.EngineTypeVLLM,
			Status: engine.EngineStatusStopped,
		})

		ollamaEngines, total, err := repo.List(ctx, engine.EngineFilter{Type: engine.EngineTypeOllama})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 1 {
			t.Errorf("expected total 1 Ollama engine, got %d", total)
		}
		if len(ollamaEngines) != 1 || ollamaEngines[0].Type != engine.EngineTypeOllama {
			t.Errorf("expected 1 Ollama engine, got %+v", ollamaEngines)
		}
	})

	t.Run("list with status filter", func(t *testing.T) {
		runningEngines, total, err := repo.List(ctx, engine.EngineFilter{Status: engine.EngineStatusRunning})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 1 {
			t.Errorf("expected total 1 running engine, got %d", total)
		}
		for _, e := range runningEngines {
			if e.Status != engine.EngineStatusRunning {
				t.Errorf("expected running status, got %s", e.Status)
			}
		}
	})

	t.Run("list with pagination", func(t *testing.T) {
		// Clear previous data and create exactly 3 engines for pagination test
		_ = repo.Delete(ctx, "test-ollama-2")
		_ = repo.Delete(ctx, "test-vllm")

		_ = repo.Create(ctx, &engine.Engine{
			ID:     "engine-page-1",
			Name:   "engine-page-1",
			Type:   engine.EngineTypeOllama,
			Status: engine.EngineStatusRunning,
		})
		_ = repo.Create(ctx, &engine.Engine{
			ID:     "engine-page-2",
			Name:   "engine-page-2",
			Type:   engine.EngineTypeVLLM,
			Status: engine.EngineStatusStopped,
		})
		_ = repo.Create(ctx, &engine.Engine{
			ID:     "engine-page-3",
			Name:   "engine-page-3",
			Type:   engine.EngineTypeOllama,
			Status: engine.EngineStatusRunning,
		})

		page1, total, err := repo.List(ctx, engine.EngineFilter{Limit: 2, Offset: 0})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 3 {
			t.Errorf("expected total 3, got %d", total)
		}
		if len(page1) != 2 {
			t.Errorf("expected 2 engines in page1, got %d", len(page1))
		}
	})
}

func TestEngineRepository_Update_Delete(t *testing.T) {
	repo := NewEngineRepository()
	ctx := context.Background()

	t.Run("update existing engine", func(t *testing.T) {
		e := &engine.Engine{
			ID:      "engine-update",
			Name:    "update-test",
			Type:    engine.EngineTypeOllama,
			Status:  engine.EngineStatusStopped,
			Version: "1.0.0",
		}
		_ = repo.Create(ctx, e)

		e.Status = engine.EngineStatusRunning
		err := repo.Update(ctx, e)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		got, _ := repo.Get(ctx, "update-test")
		if got.Status != engine.EngineStatusRunning {
			t.Errorf("status not updated, got %s", got.Status)
		}
	})

	t.Run("update non-existent engine fails", func(t *testing.T) {
		e := &engine.Engine{
			ID:      "nonexistent",
			Name:    "nonexistent",
			Type:    engine.EngineTypeOllama,
			Status:  engine.EngineStatusStopped,
			Version: "1.0.0",
		}
		err := repo.Update(ctx, e)
		if !errors.Is(err, engine.ErrEngineNotFound) {
			t.Errorf("expected ErrEngineNotFound, got %v", err)
		}
	})

	t.Run("delete existing engine", func(t *testing.T) {
		e := &engine.Engine{
			ID:      "engine-delete",
			Name:    "delete-test",
			Type:    engine.EngineTypeOllama,
			Status:  engine.EngineStatusStopped,
			Version: "1.0.0",
		}
		_ = repo.Create(ctx, e)

		err := repo.Delete(ctx, "delete-test")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, err = repo.Get(ctx, "delete-test")
		if !errors.Is(err, engine.ErrEngineNotFound) {
			t.Error("engine should have been deleted")
		}
	})

	t.Run("delete non-existent engine fails", func(t *testing.T) {
		err := repo.Delete(ctx, "nonexistent")
		if !errors.Is(err, engine.ErrEngineNotFound) {
			t.Errorf("expected ErrEngineNotFound, got %v", err)
		}
	})
}
