package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/eventbus"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
)

type mockCommand struct {
	name    string
	execute func(ctx context.Context, input any) (any, error)
}

func (m *mockCommand) Name() string              { return m.name }
func (m *mockCommand) Domain() string            { return "model" }
func (m *mockCommand) InputSchema() unit.Schema  { return unit.Schema{} }
func (m *mockCommand) OutputSchema() unit.Schema { return unit.Schema{} }
func (m *mockCommand) Execute(ctx context.Context, input any) (any, error) {
	return m.execute(ctx, input)
}
func (m *mockCommand) Description() string      { return "" }
func (m *mockCommand) Examples() []unit.Example { return nil }

type mockQuery struct {
	name    string
	execute func(ctx context.Context, input any) (any, error)
}

func (m *mockQuery) Name() string              { return m.name }
func (m *mockQuery) Domain() string            { return "model" }
func (m *mockQuery) InputSchema() unit.Schema  { return unit.Schema{} }
func (m *mockQuery) OutputSchema() unit.Schema { return unit.Schema{} }
func (m *mockQuery) Execute(ctx context.Context, input any) (any, error) {
	return m.execute(ctx, input)
}
func (m *mockQuery) Description() string      { return "" }
func (m *mockQuery) Examples() []unit.Example { return nil }

func TestModelService_NewModelService(t *testing.T) {
	store := model.NewMemoryStore()
	provider := &model.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	tests := []struct {
		name     string
		registry *unit.Registry
		store    model.ModelStore
		provider model.ModelProvider
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
			name:     "with nil registry",
			registry: nil,
			store:    store,
			provider: provider,
			bus:      bus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewModelService(tt.registry, tt.store, tt.provider, tt.bus)
			if svc == nil {
				t.Error("expected non-nil ModelService")
			}
		})
	}
}

func TestModelService_PullAndVerify_Success(t *testing.T) {
	store := model.NewMemoryStore()
	provider := &model.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	registry.RegisterCommand(&mockCommand{
		name: "model.pull",
		execute: func(ctx context.Context, input any) (any, error) {
			m := &model.Model{
				ID:        "model-test123",
				Name:      "llama3",
				Type:      model.ModelTypeLLM,
				Format:    model.FormatGGUF,
				Status:    model.StatusReady,
				Size:      4500000000,
				CreatedAt: 1000,
				UpdatedAt: 1000,
				Requirements: &model.ModelRequirements{
					MemoryMin:         8000000000,
					MemoryRecommended: 16000000000,
				},
			}
			store.Create(ctx, m)
			return map[string]any{"model_id": m.ID, "status": "ready"}, nil
		},
	})
	registry.RegisterCommand(&mockCommand{
		name: "model.verify",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"valid": true, "issues": []string{}}, nil
		},
	})

	svc := NewModelService(registry, store, provider, bus)

	result, err := svc.PullAndVerify(context.Background(), "ollama", "llama3", "latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Model == nil {
		t.Fatal("expected non-nil model")
	}
	if !result.Valid {
		t.Error("expected valid=true")
	}
	if result.Requirements == nil {
		t.Error("expected non-nil requirements")
	}
}

func TestModelService_PullAndVerify_PullFails(t *testing.T) {
	store := model.NewMemoryStore()
	provider := &model.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	registry.RegisterCommand(&mockCommand{
		name: "model.pull",
		execute: func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("pull failed")
		},
	})

	svc := NewModelService(registry, store, provider, bus)

	result, err := svc.PullAndVerify(context.Background(), "ollama", "llama3", "latest")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestModelService_PullAndVerify_PullCommandNotFound(t *testing.T) {
	store := model.NewMemoryStore()
	provider := &model.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	svc := NewModelService(registry, store, provider, bus)

	result, err := svc.PullAndVerify(context.Background(), "ollama", "llama3", "latest")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestModelService_PullAndVerify_VerifyFails(t *testing.T) {
	store := model.NewMemoryStore()
	provider := &model.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	registry.RegisterCommand(&mockCommand{
		name: "model.pull",
		execute: func(ctx context.Context, input any) (any, error) {
			m := &model.Model{
				ID:        "model-test123",
				Name:      "llama3",
				Type:      model.ModelTypeLLM,
				Format:    model.FormatGGUF,
				Status:    model.StatusReady,
				Size:      4500000000,
				CreatedAt: 1000,
				UpdatedAt: 1000,
			}
			store.Create(ctx, m)
			return map[string]any{"model_id": m.ID, "status": "ready"}, nil
		},
	})
	registry.RegisterCommand(&mockCommand{
		name: "model.verify",
		execute: func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("verify failed")
		},
	})
	registry.RegisterCommand(&mockCommand{
		name: "model.delete",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			store.Delete(ctx, inputMap["model_id"].(string))
			return map[string]any{"success": true}, nil
		},
	})

	svc := NewModelService(registry, store, provider, bus)

	result, err := svc.PullAndVerify(context.Background(), "ollama", "llama3", "latest")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}

	_, getErr := store.Get(context.Background(), "model-test123")
	if getErr == nil {
		t.Error("expected model to be deleted after verify failure")
	}
}

func TestModelService_PullAndVerify_VerifyInvalidModel(t *testing.T) {
	store := model.NewMemoryStore()
	provider := &model.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	registry.RegisterCommand(&mockCommand{
		name: "model.pull",
		execute: func(ctx context.Context, input any) (any, error) {
			m := &model.Model{
				ID:        "model-test123",
				Name:      "llama3",
				Type:      model.ModelTypeLLM,
				Format:    model.FormatGGUF,
				Status:    model.StatusReady,
				Size:      4500000000,
				CreatedAt: 1000,
				UpdatedAt: 1000,
			}
			store.Create(ctx, m)
			return map[string]any{"model_id": m.ID, "status": "ready"}, nil
		},
	})
	registry.RegisterCommand(&mockCommand{
		name: "model.verify",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"valid": false, "issues": []string{"checksum mismatch"}}, nil
		},
	})
	registry.RegisterCommand(&mockCommand{
		name: "model.delete",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			store.Delete(ctx, inputMap["model_id"].(string))
			return map[string]any{"success": true}, nil
		},
	})

	svc := NewModelService(registry, store, provider, bus)

	result, err := svc.PullAndVerify(context.Background(), "ollama", "llama3", "latest")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}

	_, getErr := store.Get(context.Background(), "model-test123")
	if getErr == nil {
		t.Error("expected model to be deleted after invalid verification")
	}
}

func TestModelService_ImportAndVerify_Success(t *testing.T) {
	store := model.NewMemoryStore()
	provider := &model.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	registry.RegisterCommand(&mockCommand{
		name: "model.import",
		execute: func(ctx context.Context, input any) (any, error) {
			m := &model.Model{
				ID:        "model-import123",
				Name:      "imported-model",
				Type:      model.ModelTypeLLM,
				Format:    model.FormatGGUF,
				Status:    model.StatusReady,
				Path:      "/models/imported",
				Size:      3000000000,
				CreatedAt: 1000,
				UpdatedAt: 1000,
				Requirements: &model.ModelRequirements{
					MemoryMin:         6000000000,
					MemoryRecommended: 12000000000,
				},
			}
			store.Create(ctx, m)
			return map[string]any{"model_id": m.ID}, nil
		},
	})
	registry.RegisterCommand(&mockCommand{
		name: "model.verify",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"valid": true, "issues": []string{}}, nil
		},
	})

	svc := NewModelService(registry, store, provider, bus)

	result, err := svc.ImportAndVerify(context.Background(), "/models/imported", WithImportName("my-model"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Model == nil {
		t.Fatal("expected non-nil model")
	}
	if !result.Valid {
		t.Error("expected valid=true")
	}
}

func TestModelService_ImportAndVerify_ImportFails(t *testing.T) {
	store := model.NewMemoryStore()
	provider := &model.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	registry.RegisterCommand(&mockCommand{
		name: "model.import",
		execute: func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("import failed")
		},
	})

	svc := NewModelService(registry, store, provider, bus)

	result, err := svc.ImportAndVerify(context.Background(), "/models/imported")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestModelService_ImportAndVerify_ImportCommandNotFound(t *testing.T) {
	store := model.NewMemoryStore()
	provider := &model.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	svc := NewModelService(registry, store, provider, bus)

	result, err := svc.ImportAndVerify(context.Background(), "/models/imported")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestModelService_ImportAndVerify_VerifyFails(t *testing.T) {
	store := model.NewMemoryStore()
	provider := &model.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	registry.RegisterCommand(&mockCommand{
		name: "model.import",
		execute: func(ctx context.Context, input any) (any, error) {
			m := &model.Model{
				ID:        "model-import123",
				Name:      "imported-model",
				Type:      model.ModelTypeLLM,
				Format:    model.FormatGGUF,
				Status:    model.StatusReady,
				Path:      "/models/imported",
				CreatedAt: 1000,
				UpdatedAt: 1000,
			}
			store.Create(ctx, m)
			return map[string]any{"model_id": m.ID}, nil
		},
	})
	registry.RegisterCommand(&mockCommand{
		name: "model.verify",
		execute: func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("verify failed")
		},
	})
	registry.RegisterCommand(&mockCommand{
		name: "model.delete",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			store.Delete(ctx, inputMap["model_id"].(string))
			return map[string]any{"success": true}, nil
		},
	})

	svc := NewModelService(registry, store, provider, bus)

	result, err := svc.ImportAndVerify(context.Background(), "/models/imported")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestModelService_ImportAndVerify_WithOptions(t *testing.T) {
	store := model.NewMemoryStore()
	provider := &model.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	var receivedInput map[string]any
	registry.RegisterCommand(&mockCommand{
		name: "model.import",
		execute: func(ctx context.Context, input any) (any, error) {
			receivedInput = input.(map[string]any)
			m := &model.Model{
				ID:        "model-import123",
				Name:      "imported-model",
				Type:      model.ModelTypeLLM,
				Format:    model.FormatGGUF,
				Status:    model.StatusReady,
				CreatedAt: 1000,
				UpdatedAt: 1000,
			}
			store.Create(ctx, m)
			return map[string]any{"model_id": m.ID}, nil
		},
	})
	registry.RegisterCommand(&mockCommand{
		name: "model.verify",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"valid": true, "issues": []string{}}, nil
		},
	})

	svc := NewModelService(registry, store, provider, bus)

	_, err := svc.ImportAndVerify(context.Background(), "/models/imported",
		WithImportName("custom-name"),
		WithImportType("vlm"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedInput["name"] != "custom-name" {
		t.Errorf("expected name=custom-name, got %v", receivedInput["name"])
	}
	if receivedInput["type"] != "vlm" {
		t.Errorf("expected type=vlm, got %v", receivedInput["type"])
	}
}

func TestModelService_GetWithRequirements(t *testing.T) {
	tests := []struct {
		name            string
		setupRegistry   func(*unit.Registry)
		modelID         string
		wantErr         bool
		wantModelID     string
		hasRequirements bool
	}{
		{
			name: "success with requirements",
			setupRegistry: func(r *unit.Registry) {
				r.RegisterQuery(&mockQuery{
					name: "model.get",
					execute: func(ctx context.Context, input any) (any, error) {
						return map[string]any{
							"id":     "model-123",
							"name":   "llama3",
							"type":   "llm",
							"format": "gguf",
							"status": "ready",
							"size":   int64(4500000000),
							"requirements": map[string]any{
								"memory_min":         int64(8000000000),
								"memory_recommended": int64(16000000000),
								"gpu_type":           "NVIDIA RTX 4090",
							},
						}, nil
					},
				})
			},
			modelID:         "model-123",
			wantErr:         false,
			wantModelID:     "model-123",
			hasRequirements: true,
		},
		{
			name: "success without requirements",
			setupRegistry: func(r *unit.Registry) {
				r.RegisterQuery(&mockQuery{
					name: "model.get",
					execute: func(ctx context.Context, input any) (any, error) {
						return map[string]any{
							"id":     "model-456",
							"name":   "test-model",
							"type":   "vlm",
							"format": "safetensors",
							"status": "ready",
							"size":   int64(2000000000),
						}, nil
					},
				})
				r.RegisterQuery(&mockQuery{
					name: "model.estimate_resources",
					execute: func(ctx context.Context, input any) (any, error) {
						return map[string]any{
							"memory_min":         int64(4000000000),
							"memory_recommended": int64(8000000000),
							"gpu_type":           "NVIDIA RTX 3080",
						}, nil
					},
				})
			},
			modelID:         "model-456",
			wantErr:         false,
			wantModelID:     "model-456",
			hasRequirements: true,
		},
		{
			name: "query not found",
			setupRegistry: func(r *unit.Registry) {
			},
			modelID: "model-123",
			wantErr: true,
		},
		{
			name: "query error",
			setupRegistry: func(r *unit.Registry) {
				r.RegisterQuery(&mockQuery{
					name: "model.get",
					execute: func(ctx context.Context, input any) (any, error) {
						return nil, errors.New("model not found")
					},
				})
			},
			modelID: "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := model.NewMemoryStore()
			provider := &model.MockProvider{}
			bus := eventbus.NewInMemoryEventBus()
			defer bus.Close()

			registry := unit.NewRegistry()
			tt.setupRegistry(registry)

			svc := NewModelService(registry, store, provider, bus)

			result, err := svc.GetWithRequirements(context.Background(), tt.modelID)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Model.ID != tt.wantModelID {
				t.Errorf("expected model ID %s, got %s", tt.wantModelID, result.Model.ID)
			}

			if tt.hasRequirements && result.Requirements == nil {
				t.Error("expected non-nil requirements")
			}
		})
	}
}

func TestModelService_DeleteWithCleanup(t *testing.T) {
	tests := []struct {
		name          string
		setupStore    func(model.ModelStore)
		setupRegistry func(*unit.Registry)
		modelID       string
		force         bool
		wantErr       bool
		wantSuccess   bool
	}{
		{
			name: "successful delete",
			setupStore: func(s model.ModelStore) {
				s.Create(context.Background(), &model.Model{
					ID:        "model-123",
					Name:      "llama3",
					Type:      model.ModelTypeLLM,
					Format:    model.FormatGGUF,
					Status:    model.StatusReady,
					Path:      "/models/llama3",
					Size:      4500000000,
					CreatedAt: 1000,
					UpdatedAt: 1000,
				})
			},
			setupRegistry: func(r *unit.Registry) {
				r.RegisterCommand(&mockCommand{
					name: "model.delete",
					execute: func(ctx context.Context, input any) (any, error) {
						return map[string]any{"success": true}, nil
					},
				})
			},
			modelID:     "model-123",
			force:       false,
			wantErr:     false,
			wantSuccess: true,
		},
		{
			name: "delete with force",
			setupStore: func(s model.ModelStore) {
				s.Create(context.Background(), &model.Model{
					ID:        "model-456",
					Name:      "test",
					Type:      model.ModelTypeLLM,
					Format:    model.FormatGGUF,
					Status:    model.StatusReady,
					Size:      1000000000,
					CreatedAt: 1000,
					UpdatedAt: 1000,
				})
			},
			setupRegistry: func(r *unit.Registry) {
				r.RegisterCommand(&mockCommand{
					name: "model.delete",
					execute: func(ctx context.Context, input any) (any, error) {
						return map[string]any{"success": true}, nil
					},
				})
			},
			modelID:     "model-456",
			force:       true,
			wantErr:     false,
			wantSuccess: true,
		},
		{
			name: "model not found",
			setupStore: func(s model.ModelStore) {
			},
			setupRegistry: func(r *unit.Registry) {
				r.RegisterCommand(&mockCommand{
					name: "model.delete",
					execute: func(ctx context.Context, input any) (any, error) {
						return map[string]any{"success": true}, nil
					},
				})
			},
			modelID: "nonexistent",
			wantErr: true,
		},
		{
			name: "delete command not found",
			setupStore: func(s model.ModelStore) {
				s.Create(context.Background(), &model.Model{
					ID:        "model-789",
					Name:      "test",
					Type:      model.ModelTypeLLM,
					Format:    model.FormatGGUF,
					Status:    model.StatusReady,
					CreatedAt: 1000,
					UpdatedAt: 1000,
				})
			},
			setupRegistry: func(r *unit.Registry) {
			},
			modelID: "model-789",
			wantErr: true,
		},
		{
			name: "delete command fails",
			setupStore: func(s model.ModelStore) {
				s.Create(context.Background(), &model.Model{
					ID:        "model-999",
					Name:      "test",
					Type:      model.ModelTypeLLM,
					Format:    model.FormatGGUF,
					Status:    model.StatusReady,
					CreatedAt: 1000,
					UpdatedAt: 1000,
				})
			},
			setupRegistry: func(r *unit.Registry) {
				r.RegisterCommand(&mockCommand{
					name: "model.delete",
					execute: func(ctx context.Context, input any) (any, error) {
						return nil, errors.New("delete failed")
					},
				})
			},
			modelID: "model-999",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := model.NewMemoryStore()
			tt.setupStore(store)

			provider := &model.MockProvider{}
			bus := eventbus.NewInMemoryEventBus()
			defer bus.Close()

			registry := unit.NewRegistry()
			tt.setupRegistry(registry)

			svc := NewModelService(registry, store, provider, bus)

			result, err := svc.DeleteWithCleanup(context.Background(), tt.modelID, tt.force)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Success != tt.wantSuccess {
				t.Errorf("expected success=%v, got %v", tt.wantSuccess, result.Success)
			}
		})
	}
}

func TestModelService_SearchAndEstimate(t *testing.T) {
	tests := []struct {
		name          string
		setupRegistry func(*unit.Registry)
		query         string
		source        string
		modelType     model.ModelType
		limit         int
		wantErr       bool
		wantCount     int
	}{
		{
			name: "successful search",
			setupRegistry: func(r *unit.Registry) {
				r.RegisterQuery(&mockQuery{
					name: "model.search",
					execute: func(ctx context.Context, input any) (any, error) {
						return map[string]any{
							"results": []map[string]any{
								{"id": "llama3", "name": "Llama 3", "type": "llm", "source": "ollama", "downloads": 1000000},
								{"id": "llama2", "name": "Llama 2", "type": "llm", "source": "ollama", "downloads": 500000},
							},
						}, nil
					},
				})
			},
			query:     "llama",
			source:    "ollama",
			modelType: model.ModelTypeLLM,
			limit:     10,
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "search with empty results",
			setupRegistry: func(r *unit.Registry) {
				r.RegisterQuery(&mockQuery{
					name: "model.search",
					execute: func(ctx context.Context, input any) (any, error) {
						return map[string]any{
							"results": []map[string]any{},
						}, nil
					},
				})
			},
			query:     "nonexistent",
			source:    "",
			modelType: "",
			limit:     0,
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "search query not found",
			setupRegistry: func(r *unit.Registry) {
			},
			query:   "llama",
			wantErr: true,
		},
		{
			name: "search error",
			setupRegistry: func(r *unit.Registry) {
				r.RegisterQuery(&mockQuery{
					name: "model.search",
					execute: func(ctx context.Context, input any) (any, error) {
						return nil, errors.New("search failed")
					},
				})
			},
			query:   "llama",
			wantErr: true,
		},
		{
			name: "search with slice of any results",
			setupRegistry: func(r *unit.Registry) {
				r.RegisterQuery(&mockQuery{
					name: "model.search",
					execute: func(ctx context.Context, input any) (any, error) {
						return map[string]any{
							"results": []any{
								map[string]any{"id": "mistral", "name": "Mistral", "type": "llm", "source": "huggingface"},
							},
						}, nil
					},
				})
			},
			query:     "mistral",
			source:    "huggingface",
			modelType: model.ModelTypeLLM,
			limit:     5,
			wantErr:   false,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := model.NewMemoryStore()
			provider := &model.MockProvider{}
			bus := eventbus.NewInMemoryEventBus()
			defer bus.Close()

			registry := unit.NewRegistry()
			tt.setupRegistry(registry)

			svc := NewModelService(registry, store, provider, bus)

			results, err := svc.SearchAndEstimate(context.Background(), tt.query, tt.source, tt.modelType, tt.limit)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(results) != tt.wantCount {
				t.Errorf("expected %d results, got %d", tt.wantCount, len(results))
			}

			if tt.wantCount > 0 && results[0].ID == "" {
				t.Error("expected non-empty ID in first result")
			}
		})
	}
}

func TestModelService_List(t *testing.T) {
	tests := []struct {
		name       string
		setupStore func(model.ModelStore)
		filter     model.ModelFilter
		wantCount  int
		wantTotal  int
		wantErr    bool
	}{
		{
			name: "list all models",
			setupStore: func(s model.ModelStore) {
				s.Create(context.Background(), &model.Model{ID: "m1", Name: "model1", Type: model.ModelTypeLLM, Status: model.StatusReady, CreatedAt: 1000, UpdatedAt: 1000})
				s.Create(context.Background(), &model.Model{ID: "m2", Name: "model2", Type: model.ModelTypeVLM, Status: model.StatusReady, CreatedAt: 1000, UpdatedAt: 1000})
				s.Create(context.Background(), &model.Model{ID: "m3", Name: "model3", Type: model.ModelTypeLLM, Status: model.StatusPending, CreatedAt: 1000, UpdatedAt: 1000})
			},
			filter:    model.ModelFilter{},
			wantCount: 3,
			wantTotal: 3,
			wantErr:   false,
		},
		{
			name: "filter by type",
			setupStore: func(s model.ModelStore) {
				s.Create(context.Background(), &model.Model{ID: "m1", Name: "model1", Type: model.ModelTypeLLM, Status: model.StatusReady, CreatedAt: 1000, UpdatedAt: 1000})
				s.Create(context.Background(), &model.Model{ID: "m2", Name: "model2", Type: model.ModelTypeVLM, Status: model.StatusReady, CreatedAt: 1000, UpdatedAt: 1000})
				s.Create(context.Background(), &model.Model{ID: "m3", Name: "model3", Type: model.ModelTypeLLM, Status: model.StatusReady, CreatedAt: 1000, UpdatedAt: 1000})
			},
			filter:    model.ModelFilter{Type: model.ModelTypeLLM},
			wantCount: 2,
			wantTotal: 2,
			wantErr:   false,
		},
		{
			name: "filter by status",
			setupStore: func(s model.ModelStore) {
				s.Create(context.Background(), &model.Model{ID: "m1", Name: "model1", Type: model.ModelTypeLLM, Status: model.StatusReady, CreatedAt: 1000, UpdatedAt: 1000})
				s.Create(context.Background(), &model.Model{ID: "m2", Name: "model2", Type: model.ModelTypeLLM, Status: model.StatusPending, CreatedAt: 1000, UpdatedAt: 1000})
			},
			filter:    model.ModelFilter{Status: model.StatusReady},
			wantCount: 1,
			wantTotal: 1,
			wantErr:   false,
		},
		{
			name: "with limit and offset",
			setupStore: func(s model.ModelStore) {
				s.Create(context.Background(), &model.Model{ID: "m1", Name: "model1", Type: model.ModelTypeLLM, Status: model.StatusReady, CreatedAt: 1000, UpdatedAt: 1000})
				s.Create(context.Background(), &model.Model{ID: "m2", Name: "model2", Type: model.ModelTypeLLM, Status: model.StatusReady, CreatedAt: 1000, UpdatedAt: 1000})
				s.Create(context.Background(), &model.Model{ID: "m3", Name: "model3", Type: model.ModelTypeLLM, Status: model.StatusReady, CreatedAt: 1000, UpdatedAt: 1000})
			},
			filter:    model.ModelFilter{Limit: 2, Offset: 1},
			wantCount: 2,
			wantTotal: 3,
			wantErr:   false,
		},
		{
			name:       "empty store",
			setupStore: func(s model.ModelStore) {},
			filter:     model.ModelFilter{},
			wantCount:  0,
			wantTotal:  0,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := model.NewMemoryStore()
			tt.setupStore(store)

			provider := &model.MockProvider{}
			bus := eventbus.NewInMemoryEventBus()
			defer bus.Close()

			registry := unit.NewRegistry()
			svc := NewModelService(registry, store, provider, bus)

			models, total, err := svc.List(context.Background(), tt.filter)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(models) != tt.wantCount {
				t.Errorf("expected %d models, got %d", tt.wantCount, len(models))
			}

			if total != tt.wantTotal {
				t.Errorf("expected total %d, got %d", tt.wantTotal, total)
			}
		})
	}
}

func TestModelService_PullAndVerify_UnexpectedResultType(t *testing.T) {
	store := model.NewMemoryStore()
	provider := &model.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	registry.RegisterCommand(&mockCommand{
		name: "model.pull",
		execute: func(ctx context.Context, input any) (any, error) {
			return "invalid result type", nil
		},
	})

	svc := NewModelService(registry, store, provider, bus)

	result, err := svc.PullAndVerify(context.Background(), "ollama", "llama3", "latest")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestModelService_PullAndVerify_MissingModelID(t *testing.T) {
	store := model.NewMemoryStore()
	provider := &model.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	registry.RegisterCommand(&mockCommand{
		name: "model.pull",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"status": "ready"}, nil
		},
	})

	svc := NewModelService(registry, store, provider, bus)

	result, err := svc.PullAndVerify(context.Background(), "ollama", "llama3", "latest")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestModelService_DeleteWithCleanup_WithNilPath(t *testing.T) {
	store := model.NewMemoryStore()
	store.Create(context.Background(), &model.Model{
		ID:        "model-123",
		Name:      "llama3",
		Type:      model.ModelTypeLLM,
		Format:    model.FormatGGUF,
		Status:    model.StatusReady,
		Path:      "",
		Size:      4500000000,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	})

	provider := &model.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	registry.RegisterCommand(&mockCommand{
		name: "model.delete",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"success": true}, nil
		},
	})

	svc := NewModelService(registry, store, provider, bus)

	result, err := svc.DeleteWithCleanup(context.Background(), "model-123", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success=true")
	}
	if len(result.DeletedFiles) != 0 {
		t.Errorf("expected no deleted files, got %d", len(result.DeletedFiles))
	}
	if result.CleanedSpace != 4500000000 {
		t.Errorf("expected cleaned space 4500000000, got %d", result.CleanedSpace)
	}
}

func TestModelService_GetWithRequirements_UnexpectedResultType(t *testing.T) {
	store := model.NewMemoryStore()
	provider := &model.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	registry.RegisterQuery(&mockQuery{
		name: "model.get",
		execute: func(ctx context.Context, input any) (any, error) {
			return "invalid type", nil
		},
	})

	svc := NewModelService(registry, store, provider, bus)

	result, err := svc.GetWithRequirements(context.Background(), "model-123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestModelService_SearchAndEstimate_UnexpectedResultType(t *testing.T) {
	store := model.NewMemoryStore()
	provider := &model.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	registry.RegisterQuery(&mockQuery{
		name: "model.search",
		execute: func(ctx context.Context, input any) (any, error) {
			return "invalid type", nil
		},
	})

	svc := NewModelService(registry, store, provider, bus)

	result, err := svc.SearchAndEstimate(context.Background(), "llama", "", "", 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestModelService_SearchAndEstimate_MissingResultsKey(t *testing.T) {
	store := model.NewMemoryStore()
	provider := &model.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer bus.Close()

	registry := unit.NewRegistry()
	registry.RegisterQuery(&mockQuery{
		name: "model.search",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"data": "something"}, nil
		},
	})

	svc := NewModelService(registry, store, provider, bus)

	results, err := svc.SearchAndEstimate(context.Background(), "llama", "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestModelService_PublishEvent_WithNilBus(t *testing.T) {
	store := model.NewMemoryStore()
	provider := &model.MockProvider{}

	registry := unit.NewRegistry()
	registry.RegisterCommand(&mockCommand{
		name: "model.pull",
		execute: func(ctx context.Context, input any) (any, error) {
			m := &model.Model{
				ID:        "model-test123",
				Name:      "llama3",
				Type:      model.ModelTypeLLM,
				Format:    model.FormatGGUF,
				Status:    model.StatusReady,
				CreatedAt: 1000,
				UpdatedAt: 1000,
			}
			store.Create(ctx, m)
			return map[string]any{"model_id": m.ID, "status": "ready"}, nil
		},
	})
	registry.RegisterCommand(&mockCommand{
		name: "model.verify",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"valid": true, "issues": []string{}}, nil
		},
	})

	svc := NewModelService(registry, store, provider, nil)

	result, err := svc.PullAndVerify(context.Background(), "ollama", "llama3", "latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil || result.Model == nil {
		t.Error("expected successful result even with nil bus")
	}
}
