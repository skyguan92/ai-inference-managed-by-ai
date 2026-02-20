package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestEngineResource_URI(t *testing.T) {
	r := NewEngineResource("ollama", nil)
	expected := "asms://engine/ollama"
	if r.URI() != expected {
		t.Errorf("expected URI '%s', got '%s'", expected, r.URI())
	}
}

func TestEngineResource_Domain(t *testing.T) {
	r := NewEngineResource("ollama", nil)
	if r.Domain() != "engine" {
		t.Errorf("expected domain 'engine', got '%s'", r.Domain())
	}
}

func TestEngineResource_Schema(t *testing.T) {
	r := NewEngineResource("ollama", nil)
	schema := r.Schema()
	if schema.Type != "object" {
		t.Errorf("expected schema type 'object', got '%s'", schema.Type)
	}
	if _, ok := schema.Properties["name"]; !ok {
		t.Error("expected 'name' property in schema")
	}
	if _, ok := schema.Properties["type"]; !ok {
		t.Error("expected 'type' property in schema")
	}
}

func TestEngineResource_Get(t *testing.T) {
	tests := []struct {
		name       string
		store      EngineStore
		engineName string
		wantErr    bool
		checkField string
		checkValue any
	}{
		{
			name: "successful get",
			store: func() EngineStore {
				s := NewMemoryStore()
				s.Create(context.Background(), createTestEngine("ollama", EngineTypeOllama))
				return s
			}(),
			engineName: "ollama",
			wantErr:    false,
			checkField: "type",
			checkValue: "ollama",
		},
		{
			name:       "nil store",
			store:      nil,
			engineName: "ollama",
			wantErr:    true,
		},
		{
			name:       "engine not found",
			store:      NewMemoryStore(),
			engineName: "nonexistent",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewEngineResource(tt.engineName, tt.store)
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

			if _, ok := resultMap["capabilities"]; !ok {
				t.Error("expected 'capabilities' field")
			}
			if _, ok := resultMap["models"]; !ok {
				t.Error("expected 'models' field")
			}
		})
	}
}

func TestEngineResource_Watch(t *testing.T) {
	store := NewMemoryStore()
	_ = store.Create(context.Background(), createTestEngine("ollama", EngineTypeOllama))

	r := NewEngineResource("ollama", store)

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

func TestEngineResource_GetWithModels(t *testing.T) {
	store := NewMemoryStore()
	engine := createTestEngine("ollama", EngineTypeOllama)
	engine.Models = []string{"llama3", "mistral"}
	engine.Capabilities = []string{"chat", "completion", "embedding"}
	store.Create(context.Background(), engine)

	r := NewEngineResource("ollama", store)
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

	models, ok := resultMap["models"].([]string)
	if !ok {
		t.Error("expected 'models' to be []string")
		return
	}

	if len(models) != 2 {
		t.Errorf("expected 2 models, got %d", len(models))
	}

	capabilities, ok := resultMap["capabilities"].([]string)
	if !ok {
		t.Error("expected 'capabilities' to be []string")
		return
	}

	if len(capabilities) != 3 {
		t.Errorf("expected 3 capabilities, got %d", len(capabilities))
	}
}

func TestParseEngineResourceURI(t *testing.T) {
	tests := []struct {
		uri      string
		wantName string
		wantOK   bool
	}{
		{"asms://engine/ollama", "ollama", true},
		{"asms://engine/vllm", "vllm", true},
		{"asms://engine/", "", false},
		{"asms://model/model-123", "", false},
		{"invalid-uri", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			name, ok := ParseEngineResourceURI(tt.uri)
			if ok != tt.wantOK {
				t.Errorf("expected ok=%v, got %v", tt.wantOK, ok)
			}
			if name != tt.wantName {
				t.Errorf("expected name=%s, got %s", tt.wantName, name)
			}
		})
	}
}

func TestResourceImplementsInterface(t *testing.T) {
	var _ unit.Resource = NewEngineResource("ollama", nil)
}

func TestMemoryStore_CRUD(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	engine := createTestEngine("ollama", EngineTypeOllama)

	if err := store.Create(ctx, engine); err != nil {
		t.Errorf("Create failed: %v", err)
	}

	if err := store.Create(ctx, engine); !errors.Is(err, ErrEngineAlreadyExists) {
		t.Errorf("expected ErrEngineAlreadyExists, got %v", err)
	}

	got, err := store.Get(ctx, "ollama")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if got.Type != EngineTypeOllama {
		t.Errorf("expected type=ollama, got %s", got.Type)
	}

	_, err = store.Get(ctx, "nonexistent")
	if !errors.Is(err, ErrEngineNotFound) {
		t.Errorf("expected ErrEngineNotFound, got %v", err)
	}

	engines, total, err := store.List(ctx, EngineFilter{})
	if err != nil {
		t.Errorf("List failed: %v", err)
	}
	if len(engines) != 1 || total != 1 {
		t.Errorf("expected 1 engine, got %d (total: %d)", len(engines), total)
	}

	engine.Status = EngineStatusRunning
	if err := store.Update(ctx, engine); err != nil {
		t.Errorf("Update failed: %v", err)
	}

	got, _ = store.Get(ctx, "ollama")
	if got.Status != EngineStatusRunning {
		t.Errorf("expected status=running, got %s", got.Status)
	}

	nonexistent := createTestEngine("nonexistent", EngineTypeVLLM)
	if err := store.Update(ctx, nonexistent); !errors.Is(err, ErrEngineNotFound) {
		t.Errorf("expected ErrEngineNotFound, got %v", err)
	}

	if err := store.Delete(ctx, "ollama"); err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	if err := store.Delete(ctx, "ollama"); !errors.Is(err, ErrEngineNotFound) {
		t.Errorf("expected ErrEngineNotFound, got %v", err)
	}
}

func TestMemoryStore_ListWithFilter(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	engines := []*Engine{
		createTestEngine("ollama", EngineTypeOllama),
		createTestEngine("vllm", EngineTypeVLLM),
	}
	engines[1].Status = EngineStatusRunning

	for _, e := range engines {
		store.Create(ctx, e)
	}

	ollamaEngines, total, err := store.List(ctx, EngineFilter{Type: EngineTypeOllama})
	if err != nil {
		t.Errorf("List with filter failed: %v", err)
	}
	if len(ollamaEngines) != 1 || total != 1 {
		t.Errorf("expected 1 ollama engine, got %d (total: %d)", len(ollamaEngines), total)
	}

	runningEngines, total, err := store.List(ctx, EngineFilter{Status: EngineStatusRunning})
	if err != nil {
		t.Errorf("List with status filter failed: %v", err)
	}
	if len(runningEngines) != 1 || total != 1 {
		t.Errorf("expected 1 running engine, got %d (total: %d)", len(runningEngines), total)
	}

	store.Create(ctx, createTestEngine("sglang", EngineTypeSGLang))
	store.Create(ctx, createTestEngine("whisper", EngineTypeWhisper))

	pagedEngines, total, err := store.List(ctx, EngineFilter{Limit: 2, Offset: 1})
	if err != nil {
		t.Errorf("List with pagination failed: %v", err)
	}
	if len(pagedEngines) != 2 {
		t.Errorf("expected 2 engines, got %d", len(pagedEngines))
	}
	if total != 4 {
		t.Errorf("expected total=4, got %d", total)
	}
}

