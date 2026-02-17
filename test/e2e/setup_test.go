package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/registry"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/alert"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/engine"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/inference"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/pipeline"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
)

type TestEnv struct {
	Registry *unit.Registry
	Gateway  *gateway.Gateway
	Ctx      context.Context
}

func SetupTestEnv(t *testing.T) *TestEnv {
	registryInstance := unit.NewRegistry()

	opts := []registry.Option{
		registry.WithEngineProvider(newMockEngineProvider()),
		registry.WithServiceProvider(newMockServiceProvider()),
		registry.WithInferenceProvider(newMockInferenceProvider()),
	}

	if err := registry.RegisterAll(registryInstance, opts...); err != nil {
		t.Fatalf("failed to register all units: %v", err)
	}

	gw := gateway.NewGateway(registryInstance)

	return &TestEnv{
		Registry: registryInstance,
		Gateway:  gw,
		Ctx:      context.Background(),
	}
}

func newMockEngineProvider() *engine.MockProvider {
	return &engine.MockProvider{}
}

func newMockServiceProvider() *MockServiceProvider {
	return &MockServiceProvider{}
}

func newMockInferenceProvider() *MockInferenceProvider {
	return &MockInferenceProvider{}
}

type MockServiceProvider struct{}

func (m *MockServiceProvider) Create(ctx context.Context, modelID string, resourceClass service.ResourceClass, replicas int, persistent bool) (*service.ModelService, error) {
	now := time.Now().Unix()
	return &service.ModelService{
		ID:            "svc-" + uuid.New().String()[:8],
		ModelID:       modelID,
		Status:        service.ServiceStatusRunning,
		Replicas:      replicas,
		ResourceClass: resourceClass,
		Endpoints:     []string{"http://localhost:8080"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func (m *MockServiceProvider) Delete(ctx context.Context, serviceID string) error {
	return nil
}

func (m *MockServiceProvider) Scale(ctx context.Context, serviceID string, replicas int) error {
	return nil
}

func (m *MockServiceProvider) Start(ctx context.Context, serviceID string) error {
	return nil
}

func (m *MockServiceProvider) Stop(ctx context.Context, serviceID string, force bool) error {
	return nil
}

func (m *MockServiceProvider) GetMetrics(ctx context.Context, serviceID string) (*service.ServiceMetrics, error) {
	return &service.ServiceMetrics{
		RequestsPerSecond: 100.0,
		LatencyP50:        50,
		LatencyP99:        200,
		TotalRequests:     10000,
		ErrorRate:         0.01,
	}, nil
}

func (m *MockServiceProvider) GetRecommendation(ctx context.Context, modelID string, hint string) (*service.Recommendation, error) {
	return &service.Recommendation{
		ResourceClass:      service.ResourceClassMedium,
		Replicas:           2,
		ExpectedThroughput: 100.0,
	}, nil
}

type MockInferenceProvider struct{}

func (m *MockInferenceProvider) Chat(ctx context.Context, modelID string, messages []inference.Message, opts inference.ChatOptions) (*inference.ChatResponse, error) {
	return &inference.ChatResponse{
		Content:      "Hello! I'm a mock AI response.",
		FinishReason: "stop",
		Model:        modelID,
		ID:           "chat-" + uuid.New().String()[:8],
		Usage:        inference.Usage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30},
	}, nil
}

func (m *MockInferenceProvider) Complete(ctx context.Context, modelID string, prompt string, opts inference.CompleteOptions) (*inference.CompletionResponse, error) {
	return &inference.CompletionResponse{
		Text:         "This is a mock completion response.",
		FinishReason: "stop",
		Usage:        inference.Usage{PromptTokens: 5, CompletionTokens: 10, TotalTokens: 15},
	}, nil
}

func (m *MockInferenceProvider) Embed(ctx context.Context, modelID string, texts []string) (*inference.EmbeddingResponse, error) {
	embeddings := make([][]float64, len(texts))
	for i := range texts {
		embeddings[i] = []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	}
	return &inference.EmbeddingResponse{
		Embeddings: embeddings,
		Usage:      inference.Usage{PromptTokens: len(texts), TotalTokens: len(texts)},
	}, nil
}

func (m *MockInferenceProvider) Transcribe(ctx context.Context, modelID string, audio []byte, language string) (*inference.TranscriptionResponse, error) {
	return &inference.TranscriptionResponse{
		Text:     "This is a mock transcription.",
		Language: language,
		Duration: 5.0,
		Segments: []inference.TranscriptionSegment{},
	}, nil
}

func (m *MockInferenceProvider) Synthesize(ctx context.Context, modelID string, text string, voice string) (*inference.AudioResponse, error) {
	return &inference.AudioResponse{
		Audio:    []byte("mock_audio_data"),
		Format:   "wav",
		Duration: 2.5,
	}, nil
}

func (m *MockInferenceProvider) GenerateImage(ctx context.Context, modelID string, prompt string, opts inference.ImageOptions) (*inference.ImageGenerationResponse, error) {
	return &inference.ImageGenerationResponse{
		Images: []inference.GeneratedImage{
			{Base64: "mock_base64_image_data"},
		},
		Format: "png",
	}, nil
}

func (m *MockInferenceProvider) GenerateVideo(ctx context.Context, modelID string, prompt string, opts inference.VideoOptions) (*inference.VideoGenerationResponse, error) {
	return &inference.VideoGenerationResponse{
		Video:    []byte("mock_video_data"),
		Format:   "mp4",
		Duration: 5.0,
	}, nil
}

func (m *MockInferenceProvider) Rerank(ctx context.Context, modelID string, query string, documents []string) (*inference.RerankResponse, error) {
	results := make([]inference.RerankResult, len(documents))
	for i, doc := range documents {
		results[i] = inference.RerankResult{
			Document: doc,
			Score:    0.9 - float64(i)*0.1,
			Index:    i,
		}
	}
	return &inference.RerankResponse{Results: results}, nil
}

func (m *MockInferenceProvider) Detect(ctx context.Context, modelID string, image []byte) (*inference.DetectionResponse, error) {
	return &inference.DetectionResponse{
		Detections: []inference.Detection{
			{Label: "person", Confidence: 0.95, BBox: inference.BBox{100, 100, 200, 300}},
		},
	}, nil
}

func (m *MockInferenceProvider) ListModels(ctx context.Context, modelType string) ([]inference.InferenceModel, error) {
	return []inference.InferenceModel{
		{ID: "llama3", Name: "Llama 3", Type: "llm", Provider: "ollama"},
		{ID: "text-embedding-3-small", Name: "Text Embedding 3 Small", Type: "embedding", Provider: "openai"},
	}, nil
}

func (m *MockInferenceProvider) ListVoices(ctx context.Context, modelID string) ([]inference.Voice, error) {
	return []inference.Voice{
		{ID: "alloy", Name: "Alloy", Language: "en"},
		{ID: "echo", Name: "Echo", Language: "en"},
	}, nil
}

func assertSuccess(t *testing.T, resp *gateway.Response, msgAndArgs ...interface{}) {
	t.Helper()
	if !resp.Success {
		t.Errorf("expected success, got error: %v", resp.Error)
	}
}

func assertError(t *testing.T, resp *gateway.Response, msgAndArgs ...interface{}) {
	t.Helper()
	if resp.Success {
		t.Errorf("expected error, got success")
	}
}

func getStringField(data any, field string) string {
	if d, ok := data.(map[string]any); ok {
		if v, ok := d[field]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

func getFloatField(data any, field string) float64 {
	if d, ok := data.(map[string]any); ok {
		if v, ok := d[field]; ok {
			switch val := v.(type) {
			case float64:
				return val
			case int:
				return float64(val)
			case int64:
				return float64(val)
			}
		}
	}
	return 0
}

func getIntField(data any, field string) int {
	if d, ok := data.(map[string]any); ok {
		if v, ok := d[field]; ok {
			switch val := v.(type) {
			case int:
				return val
			case float64:
				return int(val)
			case int64:
				return int(val)
			}
		}
	}
	return 0
}

func getSliceField(data any, field string) []any {
	if d, ok := data.(map[string]any); ok {
		if v, ok := d[field]; ok {
			if s, ok := v.([]any); ok {
				return s
			}
		}
	}
	return nil
}

func getMapField(data any, field string) map[string]any {
	if data == nil {
		return nil
	}
	if field == "" {
		if m, ok := data.(map[string]any); ok {
			return m
		}
		return nil
	}
	if d, ok := data.(map[string]any); ok {
		if v, ok := d[field]; ok {
			if m, ok := v.(map[string]any); ok {
				return m
			}
		}
	}
	return nil
}

func createTestAlert(t *testing.T, env *TestEnv, store alert.Store) string {
	rule := &alert.AlertRule{
		ID:        uuid.New().String(),
		Name:      "test-alert-rule",
		Condition: "memory > 90%",
		Severity:  alert.AlertSeverityWarning,
		Enabled:   true,
	}
	if err := store.CreateRule(env.Ctx, rule); err != nil {
		t.Fatalf("failed to create test alert rule: %v", err)
	}
	return rule.ID
}

func createTestEngine(t *testing.T, env *TestEnv, store engine.EngineStore) string {
	now := time.Now().Unix()
	eng := &engine.Engine{
		ID:           "engine-" + uuid.New().String()[:8],
		Name:         "test-engine-" + uuid.New().String()[:4],
		Type:         engine.EngineTypeOllama,
		Status:       engine.EngineStatusStopped,
		Version:      "1.0.0",
		CreatedAt:    now,
		UpdatedAt:    now,
		Models:       []string{},
		Capabilities: []string{"chat", "completion"},
	}
	if err := store.Create(env.Ctx, eng); err != nil {
		t.Fatalf("failed to create test engine: %v", err)
	}
	return eng.Name
}

func createTestModel(t *testing.T, env *TestEnv, store model.ModelStore) string {
	model := &model.Model{
		ID:        "model-" + uuid.New().String()[:8],
		Name:      "test-model-" + uuid.New().String()[:4],
		Type:      model.ModelTypeLLM,
		Format:    model.FormatGGUF,
		Status:    model.StatusReady,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	if err := store.Create(env.Ctx, model); err != nil {
		t.Fatalf("failed to create test model: %v", err)
	}
	return model.ID
}

func createTestService(t *testing.T, env *TestEnv, store service.ServiceStore) string {
	svc := &service.ModelService{
		ID:            "svc-" + uuid.New().String()[:8],
		Name:          "test-service-" + uuid.New().String()[:4],
		ModelID:       "test-model",
		Status:        service.ServiceStatusStopped,
		Replicas:      1,
		ResourceClass: service.ResourceClassMedium,
		Endpoints:     []string{},
		CreatedAt:     time.Now().Unix(),
		UpdatedAt:     time.Now().Unix(),
	}
	if err := store.Create(env.Ctx, svc); err != nil {
		t.Fatalf("failed to create test service: %v", err)
	}
	return svc.ID
}

func createTestPipeline(t *testing.T, env *TestEnv, store pipeline.PipelineStore) string {
	pipe := &pipeline.Pipeline{
		ID:   "pipe-" + uuid.New().String()[:8],
		Name: "test-pipeline-" + uuid.New().String()[:4],
		Steps: []pipeline.PipelineStep{
			{ID: "step1", Name: "Test Step", Type: "model.list", Input: map[string]any{}},
		},
		Status:    pipeline.PipelineStatusIdle,
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	if err := store.CreatePipeline(env.Ctx, pipe); err != nil {
		t.Fatalf("failed to create test pipeline: %v", err)
	}
	return pipe.ID
}
