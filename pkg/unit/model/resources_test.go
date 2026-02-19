package model

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestModelResource_URI(t *testing.T) {
	r := NewModelResource("model-123", nil)
	expected := "asms://model/model-123"
	if r.URI() != expected {
		t.Errorf("expected URI '%s', got '%s'", expected, r.URI())
	}
}

func TestModelResource_Domain(t *testing.T) {
	r := NewModelResource("model-123", nil)
	if r.Domain() != "model" {
		t.Errorf("expected domain 'model', got '%s'", r.Domain())
	}
}

func TestModelResource_Schema(t *testing.T) {
	r := NewModelResource("model-123", nil)
	schema := r.Schema()
	if schema.Type != "object" {
		t.Errorf("expected schema type 'object', got '%s'", schema.Type)
	}
	if _, ok := schema.Properties["id"]; !ok {
		t.Error("expected 'id' property in schema")
	}
	if _, ok := schema.Properties["name"]; !ok {
		t.Error("expected 'name' property in schema")
	}
}

func TestModelResource_Get(t *testing.T) {
	tests := []struct {
		name       string
		store      ModelStore
		modelID    string
		wantErr    bool
		checkField string
		checkValue any
	}{
		{
			name: "successful get",
			store: func() ModelStore {
				s := NewMemoryStore()
				s.Create(context.Background(), createTestModel("model-123", "llama3"))
				return s
			}(),
			modelID:    "model-123",
			wantErr:    false,
			checkField: "name",
			checkValue: "llama3",
		},
		{
			name:    "nil store",
			store:   nil,
			modelID: "model-123",
			wantErr: true,
		},
		{
			name:    "model not found",
			store:   NewMemoryStore(),
			modelID: "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewModelResource(tt.modelID, tt.store)
			result, err := r.Get(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Error("expected result to be map[string]any")
				return
			}

			if tt.checkValue != nil {
				if val, exists := resultMap[tt.checkField]; exists {
					if val != tt.checkValue {
						t.Errorf("expected %s=%v, got %v", tt.checkField, tt.checkValue, val)
					}
				} else {
					t.Errorf("expected field '%s' not found", tt.checkField)
				}
			}

			if _, ok := resultMap["requirements"]; !ok {
				t.Error("expected 'requirements' field")
			}
		})
	}
}

func TestModelResource_Watch(t *testing.T) {
	store := NewMemoryStore()
	store.Create(context.Background(), createTestModel("model-123", "llama3"))

	r := NewModelResource("model-123", store)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Errorf("unexpected error from Watch: %v", err)
		return
	}

	select {
	case update, ok := <-ch:
		if ok {
			if update.URI != r.URI() {
				t.Errorf("expected URI=%s, got %s", r.URI(), update.URI)
			}
		}
	case <-ctx.Done():
	}
}

func TestModelResource_GetWithRequirements(t *testing.T) {
	store := NewMemoryStore()
	model := createTestModel("model-123", "llama3")
	model.Requirements = &ModelRequirements{
		MemoryMin:         8000000000,
		MemoryRecommended: 16000000000,
		GPUType:           "NVIDIA RTX 4090",
		GPUMemory:         24000000000,
	}
	store.Create(context.Background(), model)

	r := NewModelResource("model-123", store)
	result, err := r.Get(context.Background())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Error("expected result to be map[string]any")
		return
	}

	req, ok := resultMap["requirements"].(map[string]any)
	if !ok {
		t.Error("expected 'requirements' to be map[string]any")
		return
	}

	if req["gpu_type"] != "NVIDIA RTX 4090" {
		t.Errorf("expected gpu_type=NVIDIA RTX 4090, got %v", req["gpu_type"])
	}
}

func TestParseModelResourceURI(t *testing.T) {
	tests := []struct {
		uri         string
		wantModelID string
		wantOK      bool
	}{
		{"asms://model/model-123", "model-123", true},
		{"asms://model/abc", "abc", true},
		{"asms://model/", "", false},
		{"asms://device/gpu-0", "", false},
		{"invalid-uri", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			modelID, ok := ParseModelResourceURI(tt.uri)
			if ok != tt.wantOK {
				t.Errorf("expected ok=%v, got %v", tt.wantOK, ok)
			}
			if modelID != tt.wantModelID {
				t.Errorf("expected modelID=%s, got %s", tt.wantModelID, modelID)
			}
		})
	}
}

func TestResourceImplementsInterface(t *testing.T) {
	var _ unit.Resource = NewModelResource("model-123", nil)
}

func TestMemoryStore_CRUD(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	model := createTestModel("model-1", "test-model")

	if err := store.Create(ctx, model); err != nil {
		t.Errorf("Create failed: %v", err)
	}

	if err := store.Create(ctx, model); !errors.Is(err, ErrModelAlreadyExists) {
		t.Errorf("expected ErrModelAlreadyExists, got %v", err)
	}

	got, err := store.Get(ctx, "model-1")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if got.Name != "test-model" {
		t.Errorf("expected name=test-model, got %s", got.Name)
	}

	_, err = store.Get(ctx, "nonexistent")
	if !errors.Is(err, ErrModelNotFound) {
		t.Errorf("expected ErrModelNotFound, got %v", err)
	}

	models, total, err := store.List(ctx, ModelFilter{})
	if err != nil {
		t.Errorf("List failed: %v", err)
	}
	if len(models) != 1 || total != 1 {
		t.Errorf("expected 1 model, got %d (total: %d)", len(models), total)
	}

	model.Name = "updated-model"
	if err := store.Update(ctx, model); err != nil {
		t.Errorf("Update failed: %v", err)
	}

	got, _ = store.Get(ctx, "model-1")
	if got.Name != "updated-model" {
		t.Errorf("expected name=updated-model, got %s", got.Name)
	}

	nonexistent := createTestModel("nonexistent", "test")
	if err := store.Update(ctx, nonexistent); !errors.Is(err, ErrModelNotFound) {
		t.Errorf("expected ErrModelNotFound, got %v", err)
	}

	if err := store.Delete(ctx, "model-1"); err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	if err := store.Delete(ctx, "model-1"); !errors.Is(err, ErrModelNotFound) {
		t.Errorf("expected ErrModelNotFound, got %v", err)
	}
}

func TestMemoryStore_ListWithFilter(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	models := []*Model{
		createTestModel("model-1", "llama3"),
		createTestModel("model-2", "mistral"),
	}
	models[1].Type = ModelTypeEmbedding

	for _, m := range models {
		store.Create(ctx, m)
	}

	llmModels, total, err := store.List(ctx, ModelFilter{Type: ModelTypeLLM})
	if err != nil {
		t.Errorf("List with filter failed: %v", err)
	}
	if len(llmModels) != 1 || total != 1 {
		t.Errorf("expected 1 LLM model, got %d (total: %d)", len(llmModels), total)
	}

	store.Create(ctx, createTestModel("model-3", "codellama"))
	store.Create(ctx, createTestModel("model-4", "gemma"))

	pagedModels, total, err := store.List(ctx, ModelFilter{Limit: 2, Offset: 1})
	if err != nil {
		t.Errorf("List with pagination failed: %v", err)
	}
	if len(pagedModels) != 2 {
		t.Errorf("expected 2 models, got %d", len(pagedModels))
	}
	if total != 4 {
		t.Errorf("expected total=4, got %d", total)
	}
}

func TestCompatibilityResource_URI(t *testing.T) {
	r := NewCompatibilityResource()
	if r.URI() != "asms://models/compatibility" {
		t.Errorf("expected URI 'asms://models/compatibility', got '%s'", r.URI())
	}
}

func TestCompatibilityResource_Domain(t *testing.T) {
	r := NewCompatibilityResource()
	if r.Domain() != "model" {
		t.Errorf("expected domain 'model', got '%s'", r.Domain())
	}
}

func TestCompatibilityResource_Schema(t *testing.T) {
	r := NewCompatibilityResource()
	schema := r.Schema()
	if schema.Type != "object" {
		t.Errorf("expected schema type 'object', got '%s'", schema.Type)
	}
	if _, ok := schema.Properties["engines"]; !ok {
		t.Error("expected 'engines' property in schema")
	}
	if _, ok := schema.Properties["updated_at"]; !ok {
		t.Error("expected 'updated_at' property in schema")
	}
}

func TestCompatibilityResource_Get(t *testing.T) {
	r := NewCompatibilityResource()
	result, err := r.Get(context.Background())
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected result to be map[string]any")
	}

	engines, ok := m["engines"].([]map[string]any)
	if !ok {
		t.Fatal("expected 'engines' to be []map[string]any")
	}

	if len(engines) == 0 {
		t.Error("expected at least one engine entry")
	}

	// Verify required fields are present in each entry
	for _, e := range engines {
		if _, ok := e["engine"]; !ok {
			t.Errorf("engine entry missing 'engine' field: %v", e)
		}
		if _, ok := e["supported_types"]; !ok {
			t.Errorf("engine entry missing 'supported_types' field: %v", e)
		}
		if _, ok := e["supported_formats"]; !ok {
			t.Errorf("engine entry missing 'supported_formats' field: %v", e)
		}
		if _, ok := e["gpu_required"]; !ok {
			t.Errorf("engine entry missing 'gpu_required' field: %v", e)
		}
		if _, ok := e["quantization_support"]; !ok {
			t.Errorf("engine entry missing 'quantization_support' field: %v", e)
		}
	}

	// Check known engines are present
	engineNames := make(map[string]bool)
	for _, e := range engines {
		if name, ok := e["engine"].(string); ok {
			engineNames[name] = true
		}
	}
	for _, expected := range []string{"vllm", "whisper", "tts", "ollama"} {
		if !engineNames[expected] {
			t.Errorf("expected engine '%s' not found in compatibility matrix", expected)
		}
	}

	if _, ok := m["updated_at"]; !ok {
		t.Error("expected 'updated_at' field in result")
	}
}

func TestCompatibilityResource_VllmGPURequired(t *testing.T) {
	r := NewCompatibilityResource()
	result, _ := r.Get(context.Background())
	m := result.(map[string]any)
	engines := m["engines"].([]map[string]any)

	for _, e := range engines {
		if e["engine"] == "vllm" {
			if gpuRequired, ok := e["gpu_required"].(bool); !ok || !gpuRequired {
				t.Error("expected vllm to require GPU")
			}
			return
		}
	}
	t.Error("vllm engine not found")
}

func TestCompatibilityResource_Watch(t *testing.T) {
	r := NewCompatibilityResource()
	ctx, cancel := context.WithCancel(context.Background())

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Fatalf("Watch() returned error: %v", err)
	}

	cancel()

	// Channel should be closed after context cancellation
	_, open := <-ch
	if open {
		t.Error("expected channel to be closed after context cancellation")
	}
}

func TestCompatibilityResource_ImplementsInterface(t *testing.T) {
	var _ unit.Resource = NewCompatibilityResource()
}

// Ensure Watch is non-blocking and does not send spurious updates.
func TestCompatibilityResource_WatchNoUpdates(t *testing.T) {
	r := NewCompatibilityResource()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Fatalf("Watch() returned error: %v", err)
	}

	select {
	case update, ok := <-ch:
		if ok {
			t.Errorf("expected no updates from static resource, got: %v", update)
		}
	case <-ctx.Done():
		// Expected: no updates within the timeout.
	}
}
