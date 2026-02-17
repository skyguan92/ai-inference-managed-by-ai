package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/engine"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/inference"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/resource"
)

func TestInferenceService_NewInferenceService(t *testing.T) {
	registry := unit.NewRegistry()
	modelStore := model.NewMemoryStore()
	engineStore := engine.NewMemoryStore()
	resourceStore := resource.NewMemoryStore()
	resourceProv := &resource.MockProvider{}
	inferenceProv := inference.NewMockProvider()

	svc := NewInferenceService(registry, modelStore, engineStore, resourceStore, resourceProv, inferenceProv)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.registry != registry {
		t.Error("registry not set correctly")
	}
	if svc.modelStore != modelStore {
		t.Error("modelStore not set correctly")
	}
	if svc.engineStore != engineStore {
		t.Error("engineStore not set correctly")
	}
	if svc.resourceStore != resourceStore {
		t.Error("resourceStore not set correctly")
	}
	if svc.resourceProv != resourceProv {
		t.Error("resourceProv not set correctly")
	}
	if svc.inferenceProv != inferenceProv {
		t.Error("inferenceProv not set correctly")
	}
	if svc.router == nil {
		t.Error("router should not be nil")
	}
}

func TestInferenceService_Chat_Success(t *testing.T) {
	ctx := context.Background()
	registry := unit.NewRegistry()
	modelStore := model.NewMemoryStore()
	engineStore := engine.NewMemoryStore()
	resourceStore := resource.NewMemoryStore()
	resourceProv := &resource.MockProvider{}
	inferenceProv := inference.NewMockProvider()

	testModel := &model.Model{
		ID:        "test-model",
		Name:      "Test Model",
		Type:      model.ModelTypeLLM,
		Format:    model.FormatGGUF,
		Status:    model.StatusReady,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = modelStore.Create(ctx, testModel)

	testEngine := &engine.Engine{
		ID:        "engine-1",
		Name:      "ollama",
		Type:      engine.EngineTypeOllama,
		Status:    engine.EngineStatusRunning,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = engineStore.Create(ctx, testEngine)

	svc := NewInferenceService(registry, modelStore, engineStore, resourceStore, resourceProv, inferenceProv)

	req := ChatRequest{
		Model:    "test-model",
		Messages: []inference.Message{{Role: "user", Content: "Hello"}},
	}

	resp, err := svc.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Content == "" {
		t.Error("expected non-empty content")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("expected finish_reason 'stop', got %s", resp.FinishReason)
	}
}

func TestInferenceService_Chat_ModelNotFound(t *testing.T) {
	ctx := context.Background()
	registry := unit.NewRegistry()
	modelStore := model.NewMemoryStore()
	engineStore := engine.NewMemoryStore()
	resourceStore := resource.NewMemoryStore()
	resourceProv := &resource.MockProvider{}
	inferenceProv := inference.NewMockProvider()

	svc := NewInferenceService(registry, modelStore, engineStore, resourceStore, resourceProv, inferenceProv)

	req := ChatRequest{
		Model:    "nonexistent-model",
		Messages: []inference.Message{{Role: "user", Content: "Hello"}},
	}

	_, err := svc.Chat(ctx, req)
	if err == nil {
		t.Fatal("expected error for nonexistent model")
	}
	if !errors.Is(err, ErrModelNotFound) && !errors.Is(errors.Unwrap(err), model.ErrModelNotFound) {
		t.Errorf("expected ErrModelNotFound, got: %v", err)
	}
}

func TestInferenceService_Chat_EngineSelection(t *testing.T) {
	ctx := context.Background()
	registry := unit.NewRegistry()
	modelStore := model.NewMemoryStore()
	engineStore := engine.NewMemoryStore()
	resourceStore := resource.NewMemoryStore()
	resourceProv := &resource.MockProvider{}
	inferenceProv := inference.NewMockProvider()

	testModel := &model.Model{
		ID:        "test-model",
		Name:      "Test Model",
		Type:      model.ModelTypeLLM,
		Format:    model.FormatGGUF,
		Status:    model.StatusReady,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = modelStore.Create(ctx, testModel)

	svc := NewInferenceService(registry, modelStore, engineStore, resourceStore, resourceProv, inferenceProv)

	req := ChatRequest{
		Model:    "test-model",
		Messages: []inference.Message{{Role: "user", Content: "Hello"}},
	}

	_, err := svc.Chat(ctx, req)
	if err == nil {
		t.Fatal("expected error when no engine available")
	}
	if !errors.Is(err, ErrEngineNotAvailable) {
		t.Errorf("expected ErrEngineNotAvailable, got: %v", err)
	}
}

func TestInferenceService_Chat_ResourceCheck(t *testing.T) {
	ctx := context.Background()
	registry := unit.NewRegistry()
	modelStore := model.NewMemoryStore()
	engineStore := engine.NewMemoryStore()
	resourceStore := resource.NewMemoryStore()
	resourceProv := &testResourceProvider{
		canAllocateRes: &resource.CanAllocateResult{
			CanAllocate: false,
			Reason:      "insufficient memory",
		},
	}
	inferenceProv := inference.NewMockProvider()

	testModel := &model.Model{
		ID:     "test-model",
		Name:   "Test Model",
		Type:   model.ModelTypeLLM,
		Format: model.FormatGGUF,
		Status: model.StatusReady,
		Requirements: &model.ModelRequirements{
			MemoryMin: 8 * 1024 * 1024 * 1024,
		},
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = modelStore.Create(ctx, testModel)

	testEngine := &engine.Engine{
		ID:        "engine-1",
		Name:      "ollama",
		Type:      engine.EngineTypeOllama,
		Status:    engine.EngineStatusRunning,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = engineStore.Create(ctx, testEngine)

	svc := NewInferenceService(registry, modelStore, engineStore, resourceStore, resourceProv, inferenceProv)

	req := ChatRequest{
		Model:    "test-model",
		Messages: []inference.Message{{Role: "user", Content: "Hello"}},
	}

	_, err := svc.Chat(ctx, req)
	if err == nil {
		t.Fatal("expected error for insufficient resources")
	}
	if !errors.Is(err, ErrInsufficientResources) {
		t.Errorf("expected ErrInsufficientResources, got: %v", err)
	}
}

func TestInferenceService_Chat_InvalidRequest(t *testing.T) {
	ctx := context.Background()
	svc := NewInferenceService(nil, nil, nil, nil, nil, nil)

	tests := []struct {
		name    string
		req     ChatRequest
		wantErr error
	}{
		{
			name:    "empty model",
			req:     ChatRequest{Model: "", Messages: []inference.Message{{Role: "user", Content: "Hello"}}},
			wantErr: ErrInvalidRequest,
		},
		{
			name:    "empty messages",
			req:     ChatRequest{Model: "test-model", Messages: []inference.Message{}},
			wantErr: ErrInvalidRequest,
		},
		{
			name:    "nil messages",
			req:     ChatRequest{Model: "test-model", Messages: nil},
			wantErr: ErrInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Chat(ctx, tt.req)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestInferenceService_Complete(t *testing.T) {
	ctx := context.Background()
	registry := unit.NewRegistry()
	modelStore := model.NewMemoryStore()
	engineStore := engine.NewMemoryStore()
	resourceStore := resource.NewMemoryStore()
	resourceProv := &resource.MockProvider{}
	inferenceProv := inference.NewMockProvider()

	testModel := &model.Model{
		ID:        "test-model",
		Name:      "Test Model",
		Type:      model.ModelTypeLLM,
		Format:    model.FormatGGUF,
		Status:    model.StatusReady,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = modelStore.Create(ctx, testModel)

	testEngine := &engine.Engine{
		ID:        "engine-1",
		Name:      "ollama",
		Type:      engine.EngineTypeOllama,
		Status:    engine.EngineStatusRunning,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = engineStore.Create(ctx, testEngine)

	svc := NewInferenceService(registry, modelStore, engineStore, resourceStore, resourceProv, inferenceProv)

	req := CompleteRequest{
		Model:  "test-model",
		Prompt: "Once upon a time",
	}

	resp, err := svc.Complete(ctx, req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Text == "" {
		t.Error("expected non-empty text")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("expected finish_reason 'stop', got %s", resp.FinishReason)
	}
}

func TestInferenceService_Complete_InvalidRequest(t *testing.T) {
	ctx := context.Background()
	svc := NewInferenceService(nil, nil, nil, nil, nil, nil)

	tests := []struct {
		name    string
		req     CompleteRequest
		wantErr error
	}{
		{
			name:    "empty model",
			req:     CompleteRequest{Model: "", Prompt: "test"},
			wantErr: ErrInvalidRequest,
		},
		{
			name:    "empty prompt",
			req:     CompleteRequest{Model: "test-model", Prompt: ""},
			wantErr: ErrInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Complete(ctx, tt.req)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestInferenceService_Embed(t *testing.T) {
	ctx := context.Background()
	registry := unit.NewRegistry()
	modelStore := model.NewMemoryStore()
	engineStore := engine.NewMemoryStore()
	resourceStore := resource.NewMemoryStore()
	resourceProv := &resource.MockProvider{}
	inferenceProv := inference.NewMockProvider()

	testModel := &model.Model{
		ID:        "test-embed-model",
		Name:      "Test Embed Model",
		Type:      model.ModelTypeEmbedding,
		Format:    model.FormatPyTorch,
		Status:    model.StatusReady,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = modelStore.Create(ctx, testModel)

	testEngine := &engine.Engine{
		ID:        "engine-1",
		Name:      "transformers",
		Type:      engine.EngineTypeTransformers,
		Status:    engine.EngineStatusRunning,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = engineStore.Create(ctx, testEngine)

	svc := NewInferenceService(registry, modelStore, engineStore, resourceStore, resourceProv, inferenceProv)

	req := EmbedRequest{
		Model: "test-embed-model",
		Input: []string{"Hello world", "Test embedding"},
	}

	resp, err := svc.Embed(ctx, req)
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Embeddings) != 2 {
		t.Errorf("expected 2 embeddings, got %d", len(resp.Embeddings))
	}
	if len(resp.Embeddings[0]) != 1536 {
		t.Errorf("expected embedding dimension 1536, got %d", len(resp.Embeddings[0]))
	}
}

func TestInferenceService_Embed_InvalidRequest(t *testing.T) {
	ctx := context.Background()
	svc := NewInferenceService(nil, nil, nil, nil, nil, nil)

	tests := []struct {
		name    string
		req     EmbedRequest
		wantErr error
	}{
		{
			name:    "empty model",
			req:     EmbedRequest{Model: "", Input: []string{"test"}},
			wantErr: ErrInvalidRequest,
		},
		{
			name:    "empty input",
			req:     EmbedRequest{Model: "test-model", Input: []string{}},
			wantErr: ErrInvalidRequest,
		},
		{
			name:    "nil input",
			req:     EmbedRequest{Model: "test-model", Input: nil},
			wantErr: ErrInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Embed(ctx, tt.req)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestInferenceService_DefaultRouter_SelectEngine(t *testing.T) {
	tests := []struct {
		name        string
		modelType   model.ModelType
		modelFormat model.ModelFormat
		engines     []*engine.Engine
		wantErr     bool
	}{
		{
			name:        "llm with gguf format selects ollama",
			modelType:   model.ModelTypeLLM,
			modelFormat: model.FormatGGUF,
			engines: []*engine.Engine{
				{Name: "ollama", Type: engine.EngineTypeOllama, Status: engine.EngineStatusRunning},
			},
			wantErr: false,
		},
		{
			name:        "llm with safetensors format selects vllm",
			modelType:   model.ModelTypeLLM,
			modelFormat: model.FormatSafetensors,
			engines: []*engine.Engine{
				{Name: "vllm", Type: engine.EngineTypeVLLM, Status: engine.EngineStatusRunning},
			},
			wantErr: false,
		},
		{
			name:        "asr selects whisper",
			modelType:   model.ModelTypeASR,
			modelFormat: model.FormatPyTorch,
			engines: []*engine.Engine{
				{Name: "whisper", Type: engine.EngineTypeWhisper, Status: engine.EngineStatusRunning},
			},
			wantErr: false,
		},
		{
			name:        "tts selects tts engine",
			modelType:   model.ModelTypeTTS,
			modelFormat: model.FormatPyTorch,
			engines: []*engine.Engine{
				{Name: "tts", Type: engine.EngineTypeTTS, Status: engine.EngineStatusRunning},
			},
			wantErr: false,
		},
		{
			name:        "embedding selects transformers",
			modelType:   model.ModelTypeEmbedding,
			modelFormat: model.FormatPyTorch,
			engines: []*engine.Engine{
				{Name: "transformers", Type: engine.EngineTypeTransformers, Status: engine.EngineStatusRunning},
			},
			wantErr: false,
		},
		{
			name:        "diffusion selects diffusion engine",
			modelType:   model.ModelTypeDiffusion,
			modelFormat: model.FormatSafetensors,
			engines: []*engine.Engine{
				{Name: "diffusion", Type: engine.EngineTypeDiffusion, Status: engine.EngineStatusRunning},
			},
			wantErr: false,
		},
		{
			name:        "no running engine selects stopped engine",
			modelType:   model.ModelTypeLLM,
			modelFormat: model.FormatGGUF,
			engines: []*engine.Engine{
				{Name: "ollama", Type: engine.EngineTypeOllama, Status: engine.EngineStatusStopped},
			},
			wantErr: false,
		},
		{
			name:        "no engine available returns error",
			modelType:   model.ModelTypeLLM,
			modelFormat: model.FormatGGUF,
			engines:     []*engine.Engine{},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store := engine.NewMemoryStore()
			for _, e := range tt.engines {
				e.ID = "engine-" + e.Name
				e.CreatedAt = 1000
				e.UpdatedAt = 1000
				_ = store.Create(ctx, e)
			}

			router := NewDefaultRouter(store)
			name, err := router.SelectEngine(tt.modelType, tt.modelFormat)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if !errors.Is(err, ErrEngineNotAvailable) {
					t.Errorf("expected ErrEngineNotAvailable, got: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if name == "" {
					t.Error("expected non-empty engine name")
				}
			}
		})
	}
}

func TestInferenceService_WithRouter(t *testing.T) {
	registry := unit.NewRegistry()
	modelStore := model.NewMemoryStore()
	engineStore := engine.NewMemoryStore()
	resourceStore := resource.NewMemoryStore()
	resourceProv := &resource.MockProvider{}
	inferenceProv := inference.NewMockProvider()

	svc := NewInferenceService(registry, modelStore, engineStore, resourceStore, resourceProv, inferenceProv)

	mockRouter := &mockRouter{engineName: "custom-engine"}
	svc.WithRouter(mockRouter)

	if svc.router != mockRouter {
		t.Error("router not set correctly")
	}
}

func TestInferenceService_WithRouter_Chained(t *testing.T) {
	registry := unit.NewRegistry()
	modelStore := model.NewMemoryStore()
	engineStore := engine.NewMemoryStore()
	resourceStore := resource.NewMemoryStore()
	resourceProv := &resource.MockProvider{}
	inferenceProv := inference.NewMockProvider()

	svc := NewInferenceService(registry, modelStore, engineStore, resourceStore, resourceProv, inferenceProv)
	mockRouter := &mockRouter{engineName: "custom-engine"}

	returnedSvc := svc.WithRouter(mockRouter)

	if returnedSvc != svc {
		t.Error("WithRouter should return the same service instance")
	}
}

type mockRouter struct {
	engineName string
	err        error
}

func (m *mockRouter) SelectEngine(modelType model.ModelType, modelFormat model.ModelFormat) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.engineName, nil
}

type testResourceProvider struct {
	status         *resource.ResourceStatus
	budget         *resource.ResourceBudget
	canAllocateRes *resource.CanAllocateResult
	canAllocateErr error
	statusErr      error
	budgetErr      error
}

func (m *testResourceProvider) GetStatus(ctx context.Context) (*resource.ResourceStatus, error) {
	if m.statusErr != nil {
		return nil, m.statusErr
	}
	if m.status != nil {
		return m.status, nil
	}
	return &resource.ResourceStatus{}, nil
}

func (m *testResourceProvider) GetBudget(ctx context.Context) (*resource.ResourceBudget, error) {
	if m.budgetErr != nil {
		return nil, m.budgetErr
	}
	if m.budget != nil {
		return m.budget, nil
	}
	return &resource.ResourceBudget{}, nil
}

func (m *testResourceProvider) CanAllocate(ctx context.Context, memoryBytes uint64, priority int) (*resource.CanAllocateResult, error) {
	if m.canAllocateErr != nil {
		return nil, m.canAllocateErr
	}
	if m.canAllocateRes != nil {
		return m.canAllocateRes, nil
	}
	return &resource.CanAllocateResult{CanAllocate: true}, nil
}

func TestInferenceService_Transcribe(t *testing.T) {
	ctx := context.Background()
	registry := unit.NewRegistry()
	modelStore := model.NewMemoryStore()
	engineStore := engine.NewMemoryStore()
	resourceStore := resource.NewMemoryStore()
	resourceProv := &resource.MockProvider{}
	inferenceProv := inference.NewMockProvider()

	testModel := &model.Model{
		ID:        "whisper-model",
		Name:      "Whisper Model",
		Type:      model.ModelTypeASR,
		Format:    model.FormatPyTorch,
		Status:    model.StatusReady,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = modelStore.Create(ctx, testModel)

	testEngine := &engine.Engine{
		ID:        "engine-1",
		Name:      "whisper",
		Type:      engine.EngineTypeWhisper,
		Status:    engine.EngineStatusRunning,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = engineStore.Create(ctx, testEngine)

	svc := NewInferenceService(registry, modelStore, engineStore, resourceStore, resourceProv, inferenceProv)

	req := TranscribeRequest{
		Model:    "whisper-model",
		Audio:    []byte("mock audio data"),
		Language: "en",
	}

	resp, err := svc.Transcribe(ctx, req)
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Text == "" {
		t.Error("expected non-empty text")
	}
	if resp.Language != "en" {
		t.Errorf("expected language 'en', got %s", resp.Language)
	}
}

func TestInferenceService_Synthesize(t *testing.T) {
	ctx := context.Background()
	registry := unit.NewRegistry()
	modelStore := model.NewMemoryStore()
	engineStore := engine.NewMemoryStore()
	resourceStore := resource.NewMemoryStore()
	resourceProv := &resource.MockProvider{}
	inferenceProv := inference.NewMockProvider()

	testModel := &model.Model{
		ID:        "tts-model",
		Name:      "TTS Model",
		Type:      model.ModelTypeTTS,
		Format:    model.FormatPyTorch,
		Status:    model.StatusReady,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = modelStore.Create(ctx, testModel)

	testEngine := &engine.Engine{
		ID:        "engine-1",
		Name:      "tts",
		Type:      engine.EngineTypeTTS,
		Status:    engine.EngineStatusRunning,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = engineStore.Create(ctx, testEngine)

	svc := NewInferenceService(registry, modelStore, engineStore, resourceStore, resourceProv, inferenceProv)

	req := SynthesizeRequest{
		Model: "tts-model",
		Text:  "Hello world",
		Voice: "alloy",
	}

	resp, err := svc.Synthesize(ctx, req)
	if err != nil {
		t.Fatalf("Synthesize failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Audio) == 0 {
		t.Error("expected non-empty audio")
	}
	if resp.Format != "wav" {
		t.Errorf("expected format 'wav', got %s", resp.Format)
	}
}

func TestInferenceService_GenerateImage(t *testing.T) {
	ctx := context.Background()
	registry := unit.NewRegistry()
	modelStore := model.NewMemoryStore()
	engineStore := engine.NewMemoryStore()
	resourceStore := resource.NewMemoryStore()
	resourceProv := &resource.MockProvider{}
	inferenceProv := inference.NewMockProvider()

	testModel := &model.Model{
		ID:        "sd-model",
		Name:      "Stable Diffusion",
		Type:      model.ModelTypeDiffusion,
		Format:    model.FormatSafetensors,
		Status:    model.StatusReady,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = modelStore.Create(ctx, testModel)

	testEngine := &engine.Engine{
		ID:        "engine-1",
		Name:      "diffusion",
		Type:      engine.EngineTypeDiffusion,
		Status:    engine.EngineStatusRunning,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = engineStore.Create(ctx, testEngine)

	svc := NewInferenceService(registry, modelStore, engineStore, resourceStore, resourceProv, inferenceProv)

	req := ImageRequest{
		Model:  "sd-model",
		Prompt: "A beautiful sunset",
		Width:  512,
		Height: 512,
	}

	resp, err := svc.GenerateImage(ctx, req)
	if err != nil {
		t.Fatalf("GenerateImage failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Images) != 1 {
		t.Errorf("expected 1 image, got %d", len(resp.Images))
	}
	if resp.Format != "png" {
		t.Errorf("expected format 'png', got %s", resp.Format)
	}
}

func TestInferenceService_GenerateVideo(t *testing.T) {
	ctx := context.Background()
	registry := unit.NewRegistry()
	modelStore := model.NewMemoryStore()
	engineStore := engine.NewMemoryStore()
	resourceStore := resource.NewMemoryStore()
	resourceProv := &resource.MockProvider{}
	inferenceProv := inference.NewMockProvider()

	testModel := &model.Model{
		ID:        "video-model",
		Name:      "Video Model",
		Type:      model.ModelTypeVideoGen,
		Format:    model.FormatSafetensors,
		Status:    model.StatusReady,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = modelStore.Create(ctx, testModel)

	testEngine := &engine.Engine{
		ID:        "engine-1",
		Name:      "video",
		Type:      engine.EngineTypeVideo,
		Status:    engine.EngineStatusRunning,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = engineStore.Create(ctx, testEngine)

	svc := NewInferenceService(registry, modelStore, engineStore, resourceStore, resourceProv, inferenceProv)

	req := VideoRequest{
		Model:    "video-model",
		Prompt:   "A flying car",
		Duration: 5.0,
	}

	resp, err := svc.GenerateVideo(ctx, req)
	if err != nil {
		t.Fatalf("GenerateVideo failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Video) == 0 {
		t.Error("expected non-empty video")
	}
	if resp.Format != "mp4" {
		t.Errorf("expected format 'mp4', got %s", resp.Format)
	}
}

func TestInferenceService_Rerank(t *testing.T) {
	ctx := context.Background()
	registry := unit.NewRegistry()
	modelStore := model.NewMemoryStore()
	engineStore := engine.NewMemoryStore()
	resourceStore := resource.NewMemoryStore()
	resourceProv := &resource.MockProvider{}
	inferenceProv := inference.NewMockProvider()

	testModel := &model.Model{
		ID:        "rerank-model",
		Name:      "Rerank Model",
		Type:      model.ModelTypeRerank,
		Format:    model.FormatPyTorch,
		Status:    model.StatusReady,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = modelStore.Create(ctx, testModel)

	testEngine := &engine.Engine{
		ID:        "engine-1",
		Name:      "rerank",
		Type:      engine.EngineTypeRerank,
		Status:    engine.EngineStatusRunning,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = engineStore.Create(ctx, testEngine)

	svc := NewInferenceService(registry, modelStore, engineStore, resourceStore, resourceProv, inferenceProv)

	req := RerankRequest{
		Model:     "rerank-model",
		Query:     "What is machine learning?",
		Documents: []string{"ML is AI", "Cooking recipes"},
	}

	resp, err := svc.Rerank(ctx, req)
	if err != nil {
		t.Fatalf("Rerank failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(resp.Results))
	}
}

func TestInferenceService_Detect(t *testing.T) {
	ctx := context.Background()
	registry := unit.NewRegistry()
	modelStore := model.NewMemoryStore()
	engineStore := engine.NewMemoryStore()
	resourceStore := resource.NewMemoryStore()
	resourceProv := &resource.MockProvider{}
	inferenceProv := inference.NewMockProvider()

	testModel := &model.Model{
		ID:        "detection-model",
		Name:      "Detection Model",
		Type:      model.ModelTypeDetection,
		Format:    model.FormatPyTorch,
		Status:    model.StatusReady,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = modelStore.Create(ctx, testModel)

	testEngine := &engine.Engine{
		ID:        "engine-1",
		Name:      "transformers",
		Type:      engine.EngineTypeTransformers,
		Status:    engine.EngineStatusRunning,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = engineStore.Create(ctx, testEngine)

	svc := NewInferenceService(registry, modelStore, engineStore, resourceStore, resourceProv, inferenceProv)

	req := DetectRequest{
		Model: "detection-model",
		Image: []byte("mock image data"),
	}

	resp, err := svc.Detect(ctx, req)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Detections) != 2 {
		t.Errorf("expected 2 detections, got %d", len(resp.Detections))
	}
}

func TestDefaultRouter_mapModelToEngine(t *testing.T) {
	router := &DefaultRouter{}

	tests := []struct {
		name        string
		modelType   model.ModelType
		modelFormat model.ModelFormat
		expected    engine.EngineType
	}{
		{"llm gguf", model.ModelTypeLLM, model.FormatGGUF, engine.EngineTypeOllama},
		{"llm safetensors", model.ModelTypeLLM, model.FormatSafetensors, engine.EngineTypeVLLM},
		{"llm onnx", model.ModelTypeLLM, model.FormatONNX, engine.EngineTypeVLLM},
		{"vlm gguf", model.ModelTypeVLM, model.FormatGGUF, engine.EngineTypeOllama},
		{"vlm safetensors", model.ModelTypeVLM, model.FormatSafetensors, engine.EngineTypeVLLM},
		{"asr", model.ModelTypeASR, model.FormatPyTorch, engine.EngineTypeWhisper},
		{"tts", model.ModelTypeTTS, model.FormatPyTorch, engine.EngineTypeTTS},
		{"embedding", model.ModelTypeEmbedding, model.FormatPyTorch, engine.EngineTypeTransformers},
		{"diffusion", model.ModelTypeDiffusion, model.FormatSafetensors, engine.EngineTypeDiffusion},
		{"video_gen", model.ModelTypeVideoGen, model.FormatSafetensors, engine.EngineTypeVideo},
		{"rerank", model.ModelTypeRerank, model.FormatPyTorch, engine.EngineTypeRerank},
		{"detection", model.ModelTypeDetection, model.FormatPyTorch, engine.EngineTypeTransformers},
		{"unknown type", model.ModelType("unknown"), model.FormatGGUF, engine.EngineTypeOllama},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.mapModelToEngine(tt.modelType, tt.modelFormat)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestInferenceService_NilModelStore(t *testing.T) {
	ctx := context.Background()
	svc := NewInferenceService(nil, nil, nil, nil, nil, nil)

	req := ChatRequest{
		Model:    "test-model",
		Messages: []inference.Message{{Role: "user", Content: "Hello"}},
	}

	_, err := svc.Chat(ctx, req)
	if err == nil {
		t.Fatal("expected error for nil model store")
	}
	if !errors.Is(err, ErrModelNotFound) {
		t.Errorf("expected ErrModelNotFound, got: %v", err)
	}
}

func TestInferenceService_NilResourceProvider(t *testing.T) {
	ctx := context.Background()
	registry := unit.NewRegistry()
	modelStore := model.NewMemoryStore()
	engineStore := engine.NewMemoryStore()
	resourceStore := resource.NewMemoryStore()
	inferenceProv := inference.NewMockProvider()

	testModel := &model.Model{
		ID:     "test-model",
		Name:   "Test Model",
		Type:   model.ModelTypeLLM,
		Format: model.FormatGGUF,
		Status: model.StatusReady,
		Requirements: &model.ModelRequirements{
			MemoryMin: 8 * 1024 * 1024 * 1024,
		},
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = modelStore.Create(ctx, testModel)

	testEngine := &engine.Engine{
		ID:        "engine-1",
		Name:      "ollama",
		Type:      engine.EngineTypeOllama,
		Status:    engine.EngineStatusRunning,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = engineStore.Create(ctx, testEngine)

	svc := NewInferenceService(registry, modelStore, engineStore, resourceStore, nil, inferenceProv)

	req := ChatRequest{
		Model:    "test-model",
		Messages: []inference.Message{{Role: "user", Content: "Hello"}},
	}

	_, err := svc.Chat(ctx, req)
	if err != nil {
		t.Errorf("expected no error when resource provider is nil, got: %v", err)
	}
}

func TestInferenceService_ResourceCheck_Error(t *testing.T) {
	ctx := context.Background()
	registry := unit.NewRegistry()
	modelStore := model.NewMemoryStore()
	engineStore := engine.NewMemoryStore()
	resourceStore := resource.NewMemoryStore()
	resourceProv := &testResourceProvider{
		canAllocateErr: errors.New("resource check failed"),
	}
	inferenceProv := inference.NewMockProvider()

	testModel := &model.Model{
		ID:     "test-model",
		Name:   "Test Model",
		Type:   model.ModelTypeLLM,
		Format: model.FormatGGUF,
		Status: model.StatusReady,
		Requirements: &model.ModelRequirements{
			MemoryMin: 8 * 1024 * 1024 * 1024,
		},
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = modelStore.Create(ctx, testModel)

	testEngine := &engine.Engine{
		ID:        "engine-1",
		Name:      "ollama",
		Type:      engine.EngineTypeOllama,
		Status:    engine.EngineStatusRunning,
		CreatedAt: 1000,
		UpdatedAt: 1000,
	}
	_ = engineStore.Create(ctx, testEngine)

	svc := NewInferenceService(registry, modelStore, engineStore, resourceStore, resourceProv, inferenceProv)

	req := ChatRequest{
		Model:    "test-model",
		Messages: []inference.Message{{Role: "user", Content: "Hello"}},
	}

	_, err := svc.Chat(ctx, req)
	if err == nil {
		t.Fatal("expected error for resource check failure")
	}
}
