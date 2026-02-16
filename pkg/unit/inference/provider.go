package inference

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrProviderNotSet    = errors.New("inference provider not set")
	ErrInvalidInput      = errors.New("invalid input")
	ErrModelNotSpecified = errors.New("model not specified")
	ErrInferenceFailed   = errors.New("inference failed")
	ErrUnsupportedModel  = errors.New("unsupported model type")
)

type InferenceProvider interface {
	Chat(ctx context.Context, model string, messages []Message, opts ChatOptions) (*ChatResponse, error)
	Complete(ctx context.Context, model string, prompt string, opts CompleteOptions) (*CompletionResponse, error)
	Embed(ctx context.Context, model string, input []string) (*EmbeddingResponse, error)
	Transcribe(ctx context.Context, model string, audio []byte, language string) (*TranscriptionResponse, error)
	Synthesize(ctx context.Context, model string, text string, voice string) (*AudioResponse, error)
	GenerateImage(ctx context.Context, model string, prompt string, opts ImageOptions) (*ImageGenerationResponse, error)
	GenerateVideo(ctx context.Context, model string, prompt string, opts VideoOptions) (*VideoGenerationResponse, error)
	Rerank(ctx context.Context, model string, query string, documents []string) (*RerankResponse, error)
	Detect(ctx context.Context, model string, image []byte) (*DetectionResponse, error)
	ListModels(ctx context.Context, modelType string) ([]InferenceModel, error)
	ListVoices(ctx context.Context, model string) ([]Voice, error)
}

type ChatOptions struct {
	Temperature      *float64
	MaxTokens        *int
	TopP             *float64
	TopK             *int
	FrequencyPenalty *float64
	PresencePenalty  *float64
	Stop             []string
	Stream           bool
}

type CompleteOptions struct {
	Temperature *float64
	MaxTokens   *int
	TopP        *float64
	Stop        []string
	Stream      bool
}

type ImageOptions struct {
	Size           string
	Steps          int
	Seed           *int64
	NegativePrompt string
	Width          int
	Height         int
}

type VideoOptions struct {
	Duration float64
	FPS      int
	Width    int
	Height   int
	Steps    int
	Seed     *int64
}

type MockProvider struct {
	chatErr       error
	completeErr   error
	embedErr      error
	transcribeErr error
	synthesizeErr error
	genImageErr   error
	genVideoErr   error
	rerankErr     error
	detectErr     error
	listModelsErr error
	listVoicesErr error
}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (m *MockProvider) Chat(ctx context.Context, model string, messages []Message, opts ChatOptions) (*ChatResponse, error) {
	if m.chatErr != nil {
		return nil, m.chatErr
	}

	promptTokens := 0
	for _, msg := range messages {
		promptTokens += len(msg.Content) / 4
	}
	completionTokens := 50

	return &ChatResponse{
		Content:      "This is a mock response from the AI model.",
		FinishReason: "stop",
		Usage: Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
		Model:   model,
		ID:      "chatcmpl-" + uuid.New().String()[:8],
		Created: time.Now().Unix(),
	}, nil
}

func (m *MockProvider) Complete(ctx context.Context, model string, prompt string, opts CompleteOptions) (*CompletionResponse, error) {
	if m.completeErr != nil {
		return nil, m.completeErr
	}

	promptTokens := len(prompt) / 4
	completionTokens := 30

	return &CompletionResponse{
		Text:         "This is a mock completion response.",
		FinishReason: "stop",
		Usage: Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}, nil
}

func (m *MockProvider) Embed(ctx context.Context, model string, input []string) (*EmbeddingResponse, error) {
	if m.embedErr != nil {
		return nil, m.embedErr
	}

	embeddings := make([][]float64, len(input))
	for i := range input {
		vec := make([]float64, 1536)
		for j := range vec {
			vec[j] = 0.1
		}
		embeddings[i] = vec
	}

	totalTokens := 0
	for _, s := range input {
		totalTokens += len(s) / 4
	}

	return &EmbeddingResponse{
		Embeddings: embeddings,
		Usage: Usage{
			PromptTokens: totalTokens,
			TotalTokens:  totalTokens,
		},
	}, nil
}

func (m *MockProvider) Transcribe(ctx context.Context, model string, audio []byte, language string) (*TranscriptionResponse, error) {
	if m.transcribeErr != nil {
		return nil, m.transcribeErr
	}

	if language == "" {
		language = "en"
	}

	return &TranscriptionResponse{
		Text:     "This is a mock transcription of the audio.",
		Language: language,
		Duration: float64(len(audio)) / 16000.0,
		Segments: []TranscriptionSegment{
			{ID: 0, Start: 0.0, End: 2.5, Text: "This is a mock transcription"},
			{ID: 1, Start: 2.5, End: 5.0, Text: "of the audio."},
		},
	}, nil
}

func (m *MockProvider) Synthesize(ctx context.Context, model string, text string, voice string) (*AudioResponse, error) {
	if m.synthesizeErr != nil {
		return nil, m.synthesizeErr
	}

	if voice == "" {
		voice = "default"
	}

	duration := float64(len(text)) * 0.05
	audioData := make([]byte, int(duration*16000))

	return &AudioResponse{
		Audio:    audioData,
		Format:   "wav",
		Duration: duration,
	}, nil
}

func (m *MockProvider) GenerateImage(ctx context.Context, model string, prompt string, opts ImageOptions) (*ImageGenerationResponse, error) {
	if m.genImageErr != nil {
		return nil, m.genImageErr
	}

	return &ImageGenerationResponse{
		Images: []GeneratedImage{
			{Base64: "mock_base64_image_data"},
		},
		Format: "png",
	}, nil
}

func (m *MockProvider) GenerateVideo(ctx context.Context, model string, prompt string, opts VideoOptions) (*VideoGenerationResponse, error) {
	if m.genVideoErr != nil {
		return nil, m.genVideoErr
	}

	if opts.Duration == 0 {
		opts.Duration = 5.0
	}

	return &VideoGenerationResponse{
		Video:    []byte("mock_video_data"),
		Format:   "mp4",
		Duration: opts.Duration,
	}, nil
}

func (m *MockProvider) Rerank(ctx context.Context, model string, query string, documents []string) (*RerankResponse, error) {
	if m.rerankErr != nil {
		return nil, m.rerankErr
	}

	results := make([]RerankResult, len(documents))
	for i, doc := range documents {
		results[i] = RerankResult{
			Document: doc,
			Score:    1.0 - float64(i)*0.1,
			Index:    i,
		}
	}

	return &RerankResponse{
		Results: results,
		Usage: Usage{
			PromptTokens: len(query) / 4,
			TotalTokens:  len(query) / 4,
		},
	}, nil
}

func (m *MockProvider) Detect(ctx context.Context, model string, image []byte) (*DetectionResponse, error) {
	if m.detectErr != nil {
		return nil, m.detectErr
	}

	return &DetectionResponse{
		Detections: []Detection{
			{Label: "person", Confidence: 0.95, BBox: BBox{100, 100, 200, 300}},
			{Label: "car", Confidence: 0.87, BBox: BBox{350, 200, 150, 100}},
		},
		Model: model,
	}, nil
}

func (m *MockProvider) ListModels(ctx context.Context, modelType string) ([]InferenceModel, error) {
	if m.listModelsErr != nil {
		return nil, m.listModelsErr
	}

	models := []InferenceModel{
		{ID: "llama3", Name: "Llama 3", Type: "llm", Provider: "ollama", MaxTokens: 8192},
		{ID: "gpt-4", Name: "GPT-4", Type: "llm", Provider: "openai", MaxTokens: 8192},
		{ID: "whisper-large-v3", Name: "Whisper Large V3", Type: "asr", Provider: "ollama"},
		{ID: "tts-1", Name: "TTS 1", Type: "tts", Provider: "openai"},
		{ID: "text-embedding-3-small", Name: "Text Embedding 3 Small", Type: "embedding", Provider: "openai"},
		{ID: "dall-e-3", Name: "DALL-E 3", Type: "diffusion", Provider: "openai"},
		{ID: "stable-diffusion-xl", Name: "Stable Diffusion XL", Type: "diffusion", Provider: "local"},
	}

	if modelType != "" {
		var filtered []InferenceModel
		for _, m := range models {
			if m.Type == modelType {
				filtered = append(filtered, m)
			}
		}
		return filtered, nil
	}

	return models, nil
}

func (m *MockProvider) ListVoices(ctx context.Context, model string) ([]Voice, error) {
	if m.listVoicesErr != nil {
		return nil, m.listVoicesErr
	}

	voices := []Voice{
		{ID: "alloy", Name: "Alloy", Language: "en", Description: "Neutral and balanced"},
		{ID: "echo", Name: "Echo", Language: "en", Gender: "male", Description: "Warm and conversational"},
		{ID: "fable", Name: "Fable", Language: "en", Gender: "neutral", Description: "British accent"},
		{ID: "onyx", Name: "Onyx", Language: "en", Gender: "male", Description: "Deep and authoritative"},
		{ID: "nova", Name: "Nova", Language: "en", Gender: "female", Description: "Energetic and friendly"},
		{ID: "shimmer", Name: "Shimmer", Language: "en", Gender: "female", Description: "Soft and gentle"},
	}

	return voices, nil
}
