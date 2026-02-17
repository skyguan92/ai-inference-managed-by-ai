package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/engine"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/inference"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/resource"
)

var (
	ErrInsufficientResources = errors.New("insufficient resources")
	ErrModelNotFound         = errors.New("model not found")
	ErrEngineNotAvailable    = errors.New("engine not available")
	ErrInvalidRequest        = errors.New("invalid request")
)

type ChatRequest struct {
	Model            string              `json:"model"`
	Messages         []inference.Message `json:"messages"`
	Temperature      *float64            `json:"temperature,omitempty"`
	MaxTokens        *int                `json:"max_tokens,omitempty"`
	TopP             *float64            `json:"top_p,omitempty"`
	TopK             *int                `json:"top_k,omitempty"`
	FrequencyPenalty *float64            `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64            `json:"presence_penalty,omitempty"`
	Stop             []string            `json:"stop,omitempty"`
	Stream           bool                `json:"stream,omitempty"`
}

type ChatResponse struct {
	Content      string          `json:"content"`
	FinishReason string          `json:"finish_reason"`
	Usage        inference.Usage `json:"usage"`
	Model        string          `json:"model,omitempty"`
	ID           string          `json:"id,omitempty"`
}

type CompleteRequest struct {
	Model       string   `json:"model"`
	Prompt      string   `json:"prompt"`
	Temperature *float64 `json:"temperature,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
	Stop        []string `json:"stop,omitempty"`
	Stream      bool     `json:"stream,omitempty"`
}

type CompleteResponse struct {
	Text         string          `json:"text"`
	FinishReason string          `json:"finish_reason"`
	Usage        inference.Usage `json:"usage"`
}

type EmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type EmbedResponse struct {
	Embeddings [][]float64     `json:"embeddings"`
	Usage      inference.Usage `json:"usage"`
}

type TranscribeRequest struct {
	Model    string `json:"model"`
	Audio    []byte `json:"audio"`
	Language string `json:"language,omitempty"`
}

type TranscribeResponse struct {
	Text     string                           `json:"text"`
	Language string                           `json:"language"`
	Duration float64                          `json:"duration"`
	Segments []inference.TranscriptionSegment `json:"segments"`
}

type SynthesizeRequest struct {
	Model string `json:"model"`
	Text  string `json:"text"`
	Voice string `json:"voice,omitempty"`
}

type SynthesizeResponse struct {
	Audio    []byte  `json:"audio"`
	Format   string  `json:"format"`
	Duration float64 `json:"duration"`
}

type ImageRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	Size           string `json:"size,omitempty"`
	Steps          int    `json:"steps,omitempty"`
	Width          int    `json:"width,omitempty"`
	Height         int    `json:"height,omitempty"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	Seed           *int64 `json:"seed,omitempty"`
}

type ImageResponse struct {
	Images []inference.GeneratedImage `json:"images"`
	Format string                     `json:"format"`
}

type VideoRequest struct {
	Model    string  `json:"model"`
	Prompt   string  `json:"prompt"`
	Duration float64 `json:"duration,omitempty"`
	FPS      int     `json:"fps,omitempty"`
	Width    int     `json:"width,omitempty"`
	Height   int     `json:"height,omitempty"`
	Seed     *int64  `json:"seed,omitempty"`
}

type VideoResponse struct {
	Video    []byte  `json:"video"`
	Format   string  `json:"format"`
	Duration float64 `json:"duration"`
}

type RerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
}

type RerankResponse struct {
	Results []inference.RerankResult `json:"results"`
	Usage   inference.Usage          `json:"usage"`
}

type DetectRequest struct {
	Model string `json:"model"`
	Image []byte `json:"image"`
}

type DetectResponse struct {
	Detections []inference.Detection `json:"detections"`
	Model      string                `json:"model"`
}

type EngineRouter interface {
	SelectEngine(modelType model.ModelType, modelFormat model.ModelFormat) (string, error)
}

type DefaultRouter struct {
	engineStore engine.EngineStore
}

func NewDefaultRouter(store engine.EngineStore) *DefaultRouter {
	return &DefaultRouter{engineStore: store}
}

func (r *DefaultRouter) SelectEngine(modelType model.ModelType, modelFormat model.ModelFormat) (string, error) {
	ctx := context.Background()

	engineType := r.mapModelToEngine(modelType, modelFormat)

	engines, _, err := r.engineStore.List(ctx, engine.EngineFilter{
		Type:   engineType,
		Status: engine.EngineStatusRunning,
		Limit:  1,
	})
	if err != nil {
		return "", fmt.Errorf("list engines: %w", err)
	}

	if len(engines) == 0 {
		engines, _, err = r.engineStore.List(ctx, engine.EngineFilter{
			Type:  engineType,
			Limit: 1,
		})
		if err != nil {
			return "", fmt.Errorf("list engines: %w", err)
		}
		if len(engines) == 0 {
			return "", ErrEngineNotAvailable
		}
	}

	return engines[0].Name, nil
}

func (r *DefaultRouter) mapModelToEngine(modelType model.ModelType, modelFormat model.ModelFormat) engine.EngineType {
	switch modelType {
	case model.ModelTypeLLM, model.ModelTypeVLM:
		switch modelFormat {
		case model.FormatGGUF:
			return engine.EngineTypeOllama
		default:
			return engine.EngineTypeVLLM
		}
	case model.ModelTypeASR:
		return engine.EngineTypeWhisper
	case model.ModelTypeTTS:
		return engine.EngineTypeTTS
	case model.ModelTypeEmbedding:
		return engine.EngineTypeTransformers
	case model.ModelTypeDiffusion:
		return engine.EngineTypeDiffusion
	case model.ModelTypeVideoGen:
		return engine.EngineTypeVideo
	case model.ModelTypeRerank:
		return engine.EngineTypeRerank
	case model.ModelTypeDetection:
		return engine.EngineTypeTransformers
	default:
		return engine.EngineTypeOllama
	}
}

type InferenceService struct {
	registry      *unit.Registry
	modelStore    model.ModelStore
	engineStore   engine.EngineStore
	resourceStore resource.ResourceStore
	resourceProv  resource.ResourceProvider
	inferenceProv inference.InferenceProvider
	router        EngineRouter
}

func NewInferenceService(
	registry *unit.Registry,
	modelStore model.ModelStore,
	engineStore engine.EngineStore,
	resourceStore resource.ResourceStore,
	resourceProv resource.ResourceProvider,
	inferenceProv inference.InferenceProvider,
) *InferenceService {
	return &InferenceService{
		registry:      registry,
		modelStore:    modelStore,
		engineStore:   engineStore,
		resourceStore: resourceStore,
		resourceProv:  resourceProv,
		inferenceProv: inferenceProv,
		router:        NewDefaultRouter(engineStore),
	}
}

func (s *InferenceService) WithRouter(router EngineRouter) *InferenceService {
	s.router = router
	return s
}

func (s *InferenceService) getModel(ctx context.Context, modelID string) (*model.Model, error) {
	if s.modelStore == nil {
		return nil, ErrModelNotFound
	}

	m, err := s.modelStore.Get(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("get model %s: %w", modelID, err)
	}
	return m, nil
}

func (s *InferenceService) checkResources(ctx context.Context, memoryRequired int64) error {
	if s.resourceProv == nil {
		return nil
	}

	result, err := s.resourceProv.CanAllocate(ctx, uint64(memoryRequired), 5)
	if err != nil {
		return fmt.Errorf("check resources: %w", err)
	}

	if !result.CanAllocate {
		if result.Reason != "" {
			return fmt.Errorf("%w: %s", ErrInsufficientResources, result.Reason)
		}
		return ErrInsufficientResources
	}

	return nil
}

func (s *InferenceService) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		return nil, fmt.Errorf("model is required: %w", ErrInvalidRequest)
	}
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("messages are required: %w", ErrInvalidRequest)
	}

	m, err := s.getModel(ctx, req.Model)
	if err != nil {
		return nil, err
	}

	_, err = s.router.SelectEngine(m.Type, m.Format)
	if err != nil {
		return nil, fmt.Errorf("select engine: %w", err)
	}

	if m.Requirements != nil && m.Requirements.MemoryMin > 0 {
		if err := s.checkResources(ctx, m.Requirements.MemoryMin); err != nil {
			return nil, err
		}
	}

	opts := inference.ChatOptions{
		Temperature:      req.Temperature,
		MaxTokens:        req.MaxTokens,
		TopP:             req.TopP,
		TopK:             req.TopK,
		FrequencyPenalty: req.FrequencyPenalty,
		PresencePenalty:  req.PresencePenalty,
		Stop:             req.Stop,
		Stream:           req.Stream,
	}

	resp, err := s.inferenceProv.Chat(ctx, req.Model, req.Messages, opts)
	if err != nil {
		return nil, fmt.Errorf("chat inference: %w", err)
	}

	return &ChatResponse{
		Content:      resp.Content,
		FinishReason: resp.FinishReason,
		Usage:        resp.Usage,
		Model:        resp.Model,
		ID:           resp.ID,
	}, nil
}

func (s *InferenceService) Complete(ctx context.Context, req CompleteRequest) (*CompleteResponse, error) {
	if req.Model == "" {
		return nil, fmt.Errorf("model is required: %w", ErrInvalidRequest)
	}
	if req.Prompt == "" {
		return nil, fmt.Errorf("prompt is required: %w", ErrInvalidRequest)
	}

	m, err := s.getModel(ctx, req.Model)
	if err != nil {
		return nil, err
	}

	_, err = s.router.SelectEngine(m.Type, m.Format)
	if err != nil {
		return nil, fmt.Errorf("select engine: %w", err)
	}

	if m.Requirements != nil && m.Requirements.MemoryMin > 0 {
		if err := s.checkResources(ctx, m.Requirements.MemoryMin); err != nil {
			return nil, err
		}
	}

	opts := inference.CompleteOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		TopP:        req.TopP,
		Stop:        req.Stop,
		Stream:      req.Stream,
	}

	resp, err := s.inferenceProv.Complete(ctx, req.Model, req.Prompt, opts)
	if err != nil {
		return nil, fmt.Errorf("completion inference: %w", err)
	}

	return &CompleteResponse{
		Text:         resp.Text,
		FinishReason: resp.FinishReason,
		Usage:        resp.Usage,
	}, nil
}

func (s *InferenceService) Embed(ctx context.Context, req EmbedRequest) (*EmbedResponse, error) {
	if req.Model == "" {
		return nil, fmt.Errorf("model is required: %w", ErrInvalidRequest)
	}
	if len(req.Input) == 0 {
		return nil, fmt.Errorf("input is required: %w", ErrInvalidRequest)
	}

	m, err := s.getModel(ctx, req.Model)
	if err != nil {
		return nil, err
	}

	_, err = s.router.SelectEngine(m.Type, m.Format)
	if err != nil {
		return nil, fmt.Errorf("select engine: %w", err)
	}

	if m.Requirements != nil && m.Requirements.MemoryMin > 0 {
		if err := s.checkResources(ctx, m.Requirements.MemoryMin); err != nil {
			return nil, err
		}
	}

	resp, err := s.inferenceProv.Embed(ctx, req.Model, req.Input)
	if err != nil {
		return nil, fmt.Errorf("embedding inference: %w", err)
	}

	return &EmbedResponse{
		Embeddings: resp.Embeddings,
		Usage:      resp.Usage,
	}, nil
}

func (s *InferenceService) Transcribe(ctx context.Context, req TranscribeRequest) (*TranscribeResponse, error) {
	if req.Model == "" {
		return nil, fmt.Errorf("model is required: %w", ErrInvalidRequest)
	}
	if len(req.Audio) == 0 {
		return nil, fmt.Errorf("audio is required: %w", ErrInvalidRequest)
	}

	m, err := s.getModel(ctx, req.Model)
	if err != nil {
		return nil, err
	}

	_, err = s.router.SelectEngine(m.Type, m.Format)
	if err != nil {
		return nil, fmt.Errorf("select engine: %w", err)
	}

	if m.Requirements != nil && m.Requirements.MemoryMin > 0 {
		if err := s.checkResources(ctx, m.Requirements.MemoryMin); err != nil {
			return nil, err
		}
	}

	resp, err := s.inferenceProv.Transcribe(ctx, req.Model, req.Audio, req.Language)
	if err != nil {
		return nil, fmt.Errorf("transcription inference: %w", err)
	}

	return &TranscribeResponse{
		Text:     resp.Text,
		Language: resp.Language,
		Duration: resp.Duration,
		Segments: resp.Segments,
	}, nil
}

func (s *InferenceService) Synthesize(ctx context.Context, req SynthesizeRequest) (*SynthesizeResponse, error) {
	if req.Model == "" {
		return nil, fmt.Errorf("model is required: %w", ErrInvalidRequest)
	}
	if req.Text == "" {
		return nil, fmt.Errorf("text is required: %w", ErrInvalidRequest)
	}

	m, err := s.getModel(ctx, req.Model)
	if err != nil {
		return nil, err
	}

	_, err = s.router.SelectEngine(m.Type, m.Format)
	if err != nil {
		return nil, fmt.Errorf("select engine: %w", err)
	}

	if m.Requirements != nil && m.Requirements.MemoryMin > 0 {
		if err := s.checkResources(ctx, m.Requirements.MemoryMin); err != nil {
			return nil, err
		}
	}

	resp, err := s.inferenceProv.Synthesize(ctx, req.Model, req.Text, req.Voice)
	if err != nil {
		return nil, fmt.Errorf("synthesis inference: %w", err)
	}

	return &SynthesizeResponse{
		Audio:    resp.Audio,
		Format:   resp.Format,
		Duration: resp.Duration,
	}, nil
}

func (s *InferenceService) GenerateImage(ctx context.Context, req ImageRequest) (*ImageResponse, error) {
	if req.Model == "" {
		return nil, fmt.Errorf("model is required: %w", ErrInvalidRequest)
	}
	if req.Prompt == "" {
		return nil, fmt.Errorf("prompt is required: %w", ErrInvalidRequest)
	}

	m, err := s.getModel(ctx, req.Model)
	if err != nil {
		return nil, err
	}

	_, err = s.router.SelectEngine(m.Type, m.Format)
	if err != nil {
		return nil, fmt.Errorf("select engine: %w", err)
	}

	if m.Requirements != nil && m.Requirements.MemoryMin > 0 {
		if err := s.checkResources(ctx, m.Requirements.MemoryMin); err != nil {
			return nil, err
		}
	}

	opts := inference.ImageOptions{
		Size:           req.Size,
		Steps:          req.Steps,
		Seed:           req.Seed,
		NegativePrompt: req.NegativePrompt,
		Width:          req.Width,
		Height:         req.Height,
	}

	resp, err := s.inferenceProv.GenerateImage(ctx, req.Model, req.Prompt, opts)
	if err != nil {
		return nil, fmt.Errorf("image generation inference: %w", err)
	}

	return &ImageResponse{
		Images: resp.Images,
		Format: resp.Format,
	}, nil
}

func (s *InferenceService) GenerateVideo(ctx context.Context, req VideoRequest) (*VideoResponse, error) {
	if req.Model == "" {
		return nil, fmt.Errorf("model is required: %w", ErrInvalidRequest)
	}
	if req.Prompt == "" {
		return nil, fmt.Errorf("prompt is required: %w", ErrInvalidRequest)
	}

	m, err := s.getModel(ctx, req.Model)
	if err != nil {
		return nil, err
	}

	_, err = s.router.SelectEngine(m.Type, m.Format)
	if err != nil {
		return nil, fmt.Errorf("select engine: %w", err)
	}

	if m.Requirements != nil && m.Requirements.MemoryMin > 0 {
		if err := s.checkResources(ctx, m.Requirements.MemoryMin); err != nil {
			return nil, err
		}
	}

	opts := inference.VideoOptions{
		Duration: req.Duration,
		FPS:      req.FPS,
		Width:    req.Width,
		Height:   req.Height,
		Seed:     req.Seed,
	}

	resp, err := s.inferenceProv.GenerateVideo(ctx, req.Model, req.Prompt, opts)
	if err != nil {
		return nil, fmt.Errorf("video generation inference: %w", err)
	}

	return &VideoResponse{
		Video:    resp.Video,
		Format:   resp.Format,
		Duration: resp.Duration,
	}, nil
}

func (s *InferenceService) Rerank(ctx context.Context, req RerankRequest) (*RerankResponse, error) {
	if req.Model == "" {
		return nil, fmt.Errorf("model is required: %w", ErrInvalidRequest)
	}
	if req.Query == "" {
		return nil, fmt.Errorf("query is required: %w", ErrInvalidRequest)
	}
	if len(req.Documents) == 0 {
		return nil, fmt.Errorf("documents are required: %w", ErrInvalidRequest)
	}

	m, err := s.getModel(ctx, req.Model)
	if err != nil {
		return nil, err
	}

	_, err = s.router.SelectEngine(m.Type, m.Format)
	if err != nil {
		return nil, fmt.Errorf("select engine: %w", err)
	}

	if m.Requirements != nil && m.Requirements.MemoryMin > 0 {
		if err := s.checkResources(ctx, m.Requirements.MemoryMin); err != nil {
			return nil, err
		}
	}

	resp, err := s.inferenceProv.Rerank(ctx, req.Model, req.Query, req.Documents)
	if err != nil {
		return nil, fmt.Errorf("rerank inference: %w", err)
	}

	return &RerankResponse{
		Results: resp.Results,
		Usage:   resp.Usage,
	}, nil
}

func (s *InferenceService) Detect(ctx context.Context, req DetectRequest) (*DetectResponse, error) {
	if req.Model == "" {
		return nil, fmt.Errorf("model is required: %w", ErrInvalidRequest)
	}
	if len(req.Image) == 0 {
		return nil, fmt.Errorf("image is required: %w", ErrInvalidRequest)
	}

	m, err := s.getModel(ctx, req.Model)
	if err != nil {
		return nil, err
	}

	_, err = s.router.SelectEngine(m.Type, m.Format)
	if err != nil {
		return nil, fmt.Errorf("select engine: %w", err)
	}

	if m.Requirements != nil && m.Requirements.MemoryMin > 0 {
		if err := s.checkResources(ctx, m.Requirements.MemoryMin); err != nil {
			return nil, err
		}
	}

	resp, err := s.inferenceProv.Detect(ctx, req.Model, req.Image)
	if err != nil {
		return nil, fmt.Errorf("detection inference: %w", err)
	}

	return &DetectResponse{
		Detections: resp.Detections,
		Model:      resp.Model,
	}, nil
}
