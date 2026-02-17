package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/eventbus"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/app"
)

func TestAppService_NewAppService(t *testing.T) {
	store := app.NewMemoryStore()
	provider := &app.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	tests := []struct {
		name     string
		registry *unit.Registry
		store    app.AppStore
		provider app.AppProvider
		bus      *eventbus.InMemoryEventBus
	}{
		{
			name:     "with all dependencies",
			registry: unit.NewRegistry(),
			store:    store,
			provider: provider,
			bus:      bus,
		},
		{
			name:     "with nil bus",
			registry: unit.NewRegistry(),
			store:    store,
			provider: provider,
			bus:      nil,
		},
		{
			name:     "with nil provider",
			registry: unit.NewRegistry(),
			store:    store,
			provider: nil,
			bus:      bus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAppService(tt.registry, tt.store, tt.provider, tt.bus)
			if svc == nil {
				t.Error("expected non-nil AppService")
			}
		})
	}
}

func TestAppService_InstallWithVerify_Success(t *testing.T) {
	store := app.NewMemoryStore()
	provider := &app.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	registry.RegisterCommand(&mockCommand{
		name: "app.install",
		execute: func(ctx context.Context, input any) (any, error) {
			a := &app.App{
				ID:        "app-test123",
				Name:      "test-app",
				Template:  "open-webui",
				Status:    app.AppStatusInstalled,
				CreatedAt: time.Now().Unix(),
				UpdatedAt: time.Now().Unix(),
			}
			store.Create(ctx, a)
			return map[string]any{"app_id": a.ID}, nil
		},
	})
	registry.RegisterCommand(&mockCommand{
		name: "app.start",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			a, _ := store.Get(ctx, inputMap["app_id"].(string))
			a.Status = app.AppStatusRunning
			store.Update(ctx, a)
			return map[string]any{"success": true}, nil
		},
	})

	svc := NewAppService(registry, store, provider, bus)

	result, err := svc.InstallWithVerify(context.Background(), "open-webui", "my-app", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.App == nil {
		t.Fatal("expected non-nil app")
	}
}

func TestAppService_InstallWithVerify_InstallFails(t *testing.T) {
	store := app.NewMemoryStore()
	provider := &app.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	registry.RegisterCommand(&mockCommand{
		name: "app.install",
		execute: func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("install failed")
		},
	})

	svc := NewAppService(registry, store, provider, bus)

	result, err := svc.InstallWithVerify(context.Background(), "open-webui", "my-app", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestAppService_InstallWithVerify_CommandNotFound(t *testing.T) {
	store := app.NewMemoryStore()
	provider := &app.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	svc := NewAppService(registry, store, provider, bus)

	result, err := svc.InstallWithVerify(context.Background(), "open-webui", "my-app", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestAppService_UninstallWithCleanup_Success(t *testing.T) {
	store := app.NewMemoryStore()
	provider := &app.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	now := time.Now().Unix()
	store.Create(context.Background(), &app.App{
		ID:        "app-test123",
		Name:      "test-app",
		Template:  "open-webui",
		Status:    app.AppStatusRunning,
		Volumes:   []string{"/data/app"},
		CreatedAt: now,
		UpdatedAt: now,
	})

	registry := unit.NewRegistry()
	registry.RegisterCommand(&mockCommand{
		name: "app.stop",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"success": true}, nil
		},
	})
	registry.RegisterCommand(&mockCommand{
		name: "app.uninstall",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			store.Delete(ctx, inputMap["app_id"].(string))
			return map[string]any{"success": true}, nil
		},
	})

	svc := NewAppService(registry, store, provider, bus)

	result, err := svc.UninstallWithCleanup(context.Background(), "app-test123", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success=true")
	}
	if !result.RemovedData {
		t.Error("expected removed_data=true")
	}
	if len(result.CleanedItems) == 0 {
		t.Error("expected cleaned items")
	}
}

func TestAppService_UninstallWithCleanup_AppNotFound(t *testing.T) {
	store := app.NewMemoryStore()
	provider := &app.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	registry.RegisterCommand(&mockCommand{
		name: "app.uninstall",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"success": true}, nil
		},
	})

	svc := NewAppService(registry, store, provider, bus)

	result, err := svc.UninstallWithCleanup(context.Background(), "nonexistent", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestAppService_StartWithHealthCheck_Success(t *testing.T) {
	store := app.NewMemoryStore()
	provider := &app.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	now := time.Now().Unix()
	store.Create(context.Background(), &app.App{
		ID:        "app-test123",
		Name:      "test-app",
		Template:  "open-webui",
		Status:    app.AppStatusStopped,
		CreatedAt: now,
		UpdatedAt: now,
	})

	registry := unit.NewRegistry()
	registry.RegisterCommand(&mockCommand{
		name: "app.start",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			a, _ := store.Get(ctx, inputMap["app_id"].(string))
			a.Status = app.AppStatusRunning
			store.Update(ctx, a)
			return map[string]any{"success": true}, nil
		},
	})

	svc := NewAppService(registry, store, provider, bus)

	result, err := svc.StartWithHealthCheck(context.Background(), "app-test123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success=true")
	}
}

func TestAppService_StopGracefully_Success(t *testing.T) {
	store := app.NewMemoryStore()
	provider := &app.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	now := time.Now().Unix()
	store.Create(context.Background(), &app.App{
		ID:        "app-test123",
		Name:      "test-app",
		Template:  "open-webui",
		Status:    app.AppStatusRunning,
		CreatedAt: now,
		UpdatedAt: now,
	})

	registry := unit.NewRegistry()
	registry.RegisterCommand(&mockCommand{
		name: "app.stop",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			a, _ := store.Get(ctx, inputMap["app_id"].(string))
			a.Status = app.AppStatusStopped
			store.Update(ctx, a)
			return map[string]any{"success": true}, nil
		},
	})

	svc := NewAppService(registry, store, provider, bus)

	result, err := svc.StopGracefully(context.Background(), "app-test123", 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success=true")
	}
	if result.Timeout != 30 {
		t.Errorf("expected timeout=30, got %d", result.Timeout)
	}
}

func TestAppService_StopGracefully_AlreadyStopped(t *testing.T) {
	store := app.NewMemoryStore()
	provider := &app.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	now := time.Now().Unix()
	store.Create(context.Background(), &app.App{
		ID:        "app-test123",
		Name:      "test-app",
		Template:  "open-webui",
		Status:    app.AppStatusStopped,
		CreatedAt: now,
		UpdatedAt: now,
	})

	registry := unit.NewRegistry()
	svc := NewAppService(registry, store, provider, bus)

	result, err := svc.StopGracefully(context.Background(), "app-test123", 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success=true")
	}
	if result.WaitTime != 0 {
		t.Errorf("expected wait_time=0 for already stopped app, got %d", result.WaitTime)
	}
}

func TestAppService_GetFullInfo_Success(t *testing.T) {
	store := app.NewMemoryStore()
	provider := &app.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	registry.RegisterQuery(&mockQuery{
		name: "app.get",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"id":         "app-123",
				"name":       "test-app",
				"template":   "open-webui",
				"status":     "installed",
				"ports":      []int{8080},
				"volumes":    []string{"/data"},
				"created_at": int64(1000),
				"updated_at": int64(1000),
			}, nil
		},
	})

	svc := NewAppService(registry, store, provider, bus)

	result, err := svc.GetFullInfo(context.Background(), "app-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "app-123" {
		t.Errorf("expected ID=app-123, got %s", result.ID)
	}
	if result.Name != "test-app" {
		t.Errorf("expected name=test-app, got %s", result.Name)
	}
}

func TestAppService_ListByStatus(t *testing.T) {
	store := app.NewMemoryStore()

	now := time.Now().Unix()
	store.Create(context.Background(), &app.App{ID: "app1", Name: "app1", Status: app.AppStatusRunning, CreatedAt: now, UpdatedAt: now})
	store.Create(context.Background(), &app.App{ID: "app2", Name: "app2", Status: app.AppStatusRunning, CreatedAt: now, UpdatedAt: now})
	store.Create(context.Background(), &app.App{ID: "app3", Name: "app3", Status: app.AppStatusStopped, CreatedAt: now, UpdatedAt: now})

	registry := unit.NewRegistry()
	provider := &app.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	svc := NewAppService(registry, store, provider, bus)

	apps, total, err := svc.ListByStatus(context.Background(), app.AppStatusRunning)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(apps) != 2 {
		t.Errorf("expected 2 running apps, got %d", len(apps))
	}
	if total != 2 {
		t.Errorf("expected total=2, got %d", total)
	}
}

func TestAppService_GetLogsWithTail_Success(t *testing.T) {
	store := app.NewMemoryStore()
	provider := &app.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	registry.RegisterQuery(&mockQuery{
		name: "app.logs",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"logs": []any{
					map[string]any{"timestamp": int64(1000), "message": "App started", "level": "info"},
					map[string]any{"timestamp": int64(1001), "message": "Listening on port 8080", "level": "info"},
				},
			}, nil
		},
	})

	svc := NewAppService(registry, store, provider, bus)

	logs, err := svc.GetLogsWithTail(context.Background(), "app-123", 100, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(logs))
	}
}

func TestAppService_ListTemplatesByCategory_Success(t *testing.T) {
	store := app.NewMemoryStore()
	provider := &app.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	registry.RegisterQuery(&mockQuery{
		name: "app.templates",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"templates": []any{
					map[string]any{"id": "open-webui", "name": "Open WebUI", "category": "ai-chat", "description": "AI Chat", "image": "test:latest"},
					map[string]any{"id": "grafana", "name": "Grafana", "category": "monitoring", "description": "Dashboard", "image": "grafana:latest"},
				},
			}, nil
		},
	})

	svc := NewAppService(registry, store, provider, bus)

	templates, err := svc.ListTemplatesByCategory(context.Background(), app.AppCategoryAIChat)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(templates) != 2 {
		t.Errorf("expected 2 templates, got %d", len(templates))
	}
}
