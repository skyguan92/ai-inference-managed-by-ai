package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/engine"
)

type mockEngineCommand struct {
	name    string
	execute func(ctx context.Context, input any) (any, error)
}

func (m *mockEngineCommand) Name() string              { return m.name }
func (m *mockEngineCommand) Domain() string            { return "engine" }
func (m *mockEngineCommand) InputSchema() unit.Schema  { return unit.Schema{} }
func (m *mockEngineCommand) OutputSchema() unit.Schema { return unit.Schema{} }
func (m *mockEngineCommand) Description() string       { return "" }
func (m *mockEngineCommand) Examples() []unit.Example  { return nil }
func (m *mockEngineCommand) Execute(ctx context.Context, input any) (any, error) {
	return m.execute(ctx, input)
}

type mockEngineQuery struct {
	name    string
	execute func(ctx context.Context, input any) (any, error)
}

func (m *mockEngineQuery) Name() string              { return m.name }
func (m *mockEngineQuery) Domain() string            { return "engine" }
func (m *mockEngineQuery) InputSchema() unit.Schema  { return unit.Schema{} }
func (m *mockEngineQuery) OutputSchema() unit.Schema { return unit.Schema{} }
func (m *mockEngineQuery) Description() string       { return "" }
func (m *mockEngineQuery) Examples() []unit.Example  { return nil }
func (m *mockEngineQuery) Execute(ctx context.Context, input any) (any, error) {
	return m.execute(ctx, input)
}

func TestEngineService_GetStatus_Success(t *testing.T) {
	registry := unit.NewRegistry()
	store := engine.NewMemoryStore()
	provider := &engine.MockProvider{}

	testEngine := &engine.Engine{
		ID:        "test-id",
		Name:      "ollama",
		Type:      engine.EngineTypeOllama,
		Status:    engine.EngineStatusRunning,
		Version:   "1.0.0",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	_ = store.Create(context.Background(), testEngine)

	svc := NewEngineService(registry, store, provider)
	status, err := svc.GetStatus(context.Background(), "ollama")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if status.Engine.Name != "ollama" {
		t.Errorf("expected engine name 'ollama', got: %s", status.Engine.Name)
	}
	if !status.Health.Healthy {
		t.Error("expected engine to be healthy")
	}
	if status.Feature == nil {
		t.Error("expected features for running engine")
	}
}

func TestEngineService_GetStatus_NotFound(t *testing.T) {
	registry := unit.NewRegistry()
	store := engine.NewMemoryStore()
	provider := &engine.MockProvider{}

	svc := NewEngineService(registry, store, provider)
	_, err := svc.GetStatus(context.Background(), "nonexistent")

	if err == nil {
		t.Fatal("expected error for nonexistent engine")
	}
}

func TestEngineService_ListAvailable_Success(t *testing.T) {
	registry := unit.NewRegistry()
	store := engine.NewMemoryStore()
	provider := &engine.MockProvider{}

	listQuery := &mockEngineQuery{
		name: "engine.list",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"items": []map[string]any{
					{"name": "ollama", "type": "ollama", "status": "running", "version": "1.0.0"},
					{"name": "vllm", "type": "vllm", "status": "stopped", "version": "2.0.0"},
				},
			}, nil
		},
	}
	_ = registry.RegisterQuery(listQuery)

	svc := NewEngineService(registry, store, provider)
	engines, err := svc.ListAvailable(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(engines) != 2 {
		t.Errorf("expected 2 engines, got: %d", len(engines))
	}
	if engines[0].Name != "ollama" {
		t.Errorf("expected first engine 'ollama', got: %s", engines[0].Name)
	}
}

func TestEngineService_ListAvailable_QueryNotFound(t *testing.T) {
	registry := unit.NewRegistry()
	store := engine.NewMemoryStore()
	provider := &engine.MockProvider{}

	svc := NewEngineService(registry, store, provider)
	_, err := svc.ListAvailable(context.Background())

	if err == nil {
		t.Fatal("expected error when query not found")
	}
}

func TestEngineService_InstallEngine_Success(t *testing.T) {
	registry := unit.NewRegistry()
	store := engine.NewMemoryStore()
	provider := &engine.MockProvider{}

	installCmd := &mockEngineCommand{
		name: "engine.install",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"success": true,
				"path":    "/usr/local/bin/ollama",
			}, nil
		},
	}
	_ = registry.RegisterCommand(installCmd)

	svc := NewEngineService(registry, store, provider)
	result, err := svc.InstallEngine(context.Background(), "ollama", "1.0.0")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Error("expected installation to succeed")
	}
	if result.Path != "/usr/local/bin/ollama" {
		t.Errorf("expected path '/usr/local/bin/ollama', got: %s", result.Path)
	}
}

func TestEngineService_InstallEngine_Failure(t *testing.T) {
	registry := unit.NewRegistry()
	store := engine.NewMemoryStore()
	provider := &engine.MockProvider{}

	installCmd := &mockEngineCommand{
		name: "engine.install",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"success": false,
				"path":    "",
			}, nil
		},
	}
	_ = registry.RegisterCommand(installCmd)

	svc := NewEngineService(registry, store, provider)
	result, err := svc.InstallEngine(context.Background(), "ollama", "1.0.0")

	if err == nil {
		t.Fatal("expected error when installation fails")
	}
	if result.Success {
		t.Error("expected installation to fail")
	}
}

func TestEngineService_ForceStop_Success(t *testing.T) {
	registry := unit.NewRegistry()
	store := engine.NewMemoryStore()
	provider := &engine.MockProvider{}

	stopCmd := &mockEngineCommand{
		name: "engine.stop",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			if !inputMap["force"].(bool) {
				t.Error("expected force=true")
			}
			return map[string]any{"success": true}, nil
		},
	}
	_ = registry.RegisterCommand(stopCmd)

	svc := NewEngineService(registry, store, provider)
	err := svc.ForceStop(context.Background(), "ollama")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestEngineService_IsHealthy_Running(t *testing.T) {
	registry := unit.NewRegistry()
	store := engine.NewMemoryStore()
	provider := &engine.MockProvider{}

	testEngine := &engine.Engine{
		ID:        "test-id",
		Name:      "ollama",
		Type:      engine.EngineTypeOllama,
		Status:    engine.EngineStatusRunning,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	_ = store.Create(context.Background(), testEngine)

	svc := NewEngineService(registry, store, provider)
	health, err := svc.IsHealthy(context.Background(), "ollama")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !health.Healthy {
		t.Error("expected engine to be healthy")
	}
}

func TestEngineService_IsHealthy_Stopped(t *testing.T) {
	registry := unit.NewRegistry()
	store := engine.NewMemoryStore()
	provider := &engine.MockProvider{}

	testEngine := &engine.Engine{
		ID:        "test-id",
		Name:      "ollama",
		Type:      engine.EngineTypeOllama,
		Status:    engine.EngineStatusStopped,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	_ = store.Create(context.Background(), testEngine)

	svc := NewEngineService(registry, store, provider)
	health, err := svc.IsHealthy(context.Background(), "ollama")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if health.Healthy {
		t.Error("expected engine to be unhealthy when stopped")
	}
}

func TestEngineService_StopGracefully_Success(t *testing.T) {
	registry := unit.NewRegistry()
	store := engine.NewMemoryStore()
	provider := &engine.MockProvider{}

	stopCmd := &mockEngineCommand{
		name: "engine.stop",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			if inputMap["force"].(bool) {
				t.Error("expected force=false for graceful stop")
			}
			return map[string]any{"success": true}, nil
		},
	}
	_ = registry.RegisterCommand(stopCmd)

	svc := NewEngineService(registry, store, provider)
	err := svc.StopGracefully(context.Background(), "ollama", 30*time.Second)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestEngineService_GetFeatures_Success(t *testing.T) {
	registry := unit.NewRegistry()
	store := engine.NewMemoryStore()
	provider := &engine.MockProvider{}

	featuresQuery := &mockEngineQuery{
		name: "engine.features",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"supports_streaming":    true,
				"supports_batch":        false,
				"supports_multimodal":   true,
				"supports_tools":        true,
				"supports_embedding":    false,
				"max_concurrent":        10,
				"max_context_length":    8192,
				"max_batch_size":        16,
				"supports_gpu_layers":   true,
				"supports_quantization": false,
			}, nil
		},
	}
	_ = registry.RegisterQuery(featuresQuery)

	svc := NewEngineService(registry, store, provider)
	features, err := svc.GetFeatures(context.Background(), "ollama")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !features.SupportsStreaming {
		t.Error("expected supports_streaming=true")
	}
	if features.MaxConcurrent != 10 {
		t.Errorf("expected max_concurrent=10, got: %d", features.MaxConcurrent)
	}
}

func TestEngineService_StartWithConfig_CommandNotFound(t *testing.T) {
	registry := unit.NewRegistry()
	store := engine.NewMemoryStore()
	provider := &engine.MockProvider{}

	svc := NewEngineService(registry, store, provider)
	_, err := svc.StartWithConfig(context.Background(), "ollama", map[string]any{"port": 8080})

	if err == nil {
		t.Fatal("expected error when command not found")
	}
	if !errors.Is(err, unit.ErrCommandNotFound) {
		t.Errorf("expected ErrCommandNotFound, got: %v", err)
	}
}

func TestEngineService_Restart_CommandNotFound(t *testing.T) {
	registry := unit.NewRegistry()
	store := engine.NewMemoryStore()
	provider := &engine.MockProvider{}

	svc := NewEngineService(registry, store, provider)
	_, err := svc.Restart(context.Background(), "ollama")

	if err == nil {
		t.Fatal("expected error when command not found")
	}
}
