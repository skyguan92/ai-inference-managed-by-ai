package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/inference"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
)

// maxResponseSize caps the maximum response body read from backend inference
// services to prevent OOM from misbehaving endpoints.
const maxResponseSize = 10 * 1024 * 1024 // 10 MB

// Compile-time interface satisfaction check.
var _ inference.InferenceProvider = (*ProxyInferenceProvider)(nil)

// ProxyInferenceProvider implements inference.InferenceProvider by forwarding
// requests to running AIMA services (vLLM, Ollama, etc.).
type ProxyInferenceProvider struct {
	serviceStore service.ServiceStore
	modelStore   model.ModelStore
	httpClient   *http.Client
}

// NewProxyInferenceProvider creates a provider that proxies inference requests
// to locally running services discovered via the service and model stores.
func NewProxyInferenceProvider(serviceStore service.ServiceStore, modelStore model.ModelStore) *ProxyInferenceProvider {
	return &ProxyInferenceProvider{
		serviceStore: serviceStore,
		modelStore:   modelStore,
		httpClient:   &http.Client{Timeout: 5 * time.Minute},
	}
}

// resolveEndpoint finds a running service for the given model name and returns
// its endpoint URL. It searches models by name, then finds running services
// referencing that model's ID.
func (p *ProxyInferenceProvider) resolveEndpoint(ctx context.Context, modelName string) (string, error) {
	// First, try to find the model by name to get its ID
	models, _, err := p.modelStore.List(ctx, model.ModelFilter{})
	if err != nil {
		return "", fmt.Errorf("list models: %w", err)
	}

	var modelID string
	for _, m := range models {
		if strings.EqualFold(m.Name, modelName) || m.ID == modelName {
			modelID = m.ID
			break
		}
	}

	if modelID == "" {
		// Model not found by name — try treating the input as a model ID directly
		// and search for services referencing it
		modelID = modelName
	}

	// Find running services for this model
	svcs, _, err := p.serviceStore.List(ctx, service.ServiceFilter{
		Status:  service.ServiceStatusRunning,
		ModelID: modelID,
	})
	if err != nil {
		return "", fmt.Errorf("list services: %w", err)
	}

	if len(svcs) == 0 {
		return "", fmt.Errorf("no running services found for model %q", modelName)
	}

	svc := svcs[0]
	if len(svc.Endpoints) == 0 {
		return "", fmt.Errorf("service %q has no endpoints", svc.ID)
	}

	return svc.Endpoints[0], nil
}

// isOllamaEndpoint heuristically determines if an endpoint is Ollama (port 11434).
func isOllamaEndpoint(endpoint string) bool {
	return strings.Contains(endpoint, ":11434")
}

// --- OpenAI-compatible (vLLM) request/response types ---

type openAIChatRequest struct {
	Model       string             `json:"model"`
	Messages    []openAIChatMsg    `json:"messages"`
	Temperature *float64           `json:"temperature,omitempty"`
	MaxTokens   *int               `json:"max_tokens,omitempty"`
	TopP        *float64           `json:"top_p,omitempty"`
	Stop        []string           `json:"stop,omitempty"`
}

type openAIChatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Created int64  `json:"created"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
}

// --- Ollama request/response types ---

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaChatMsg `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  map[string]any  `json:"options,omitempty"`
}

type ollamaChatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Model   string `json:"model"`
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done               bool  `json:"done"`
	TotalDuration      int64 `json:"total_duration,omitempty"`
	PromptEvalCount    int   `json:"prompt_eval_count,omitempty"`
	EvalCount          int   `json:"eval_count,omitempty"`
}

// Chat sends a chat completion request to a running service.
func (p *ProxyInferenceProvider) Chat(ctx context.Context, modelName string, messages []inference.Message, opts inference.ChatOptions) (*inference.ChatResponse, error) {
	endpoint, err := p.resolveEndpoint(ctx, modelName)
	if err != nil {
		return nil, fmt.Errorf("inference.Chat: %w", err)
	}

	if isOllamaEndpoint(endpoint) {
		return p.chatOllama(ctx, endpoint, modelName, messages, opts)
	}
	// For vLLM endpoints, query /v1/models to get the actual model ID that vLLM knows.
	// vLLM names the model by its container mount path (e.g. "/models"), not by the AIMA model name.
	vllmModel := p.resolveVLLMModelName(ctx, endpoint, modelName)
	return p.chatOpenAI(ctx, endpoint, vllmModel, messages, opts)
}

// resolveVLLMModelName queries the vLLM /v1/models endpoint to get the actual
// model identifier. Falls back to modelName if the query fails.
func (p *ProxyInferenceProvider) resolveVLLMModelName(ctx context.Context, endpoint, fallback string) string {
	url := strings.TrimRight(endpoint, "/") + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fallback
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fallback
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fallback
	}
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1024*1024)).Decode(&result); err != nil {
		return fallback
	}
	if len(result.Data) > 0 {
		return result.Data[0].ID
	}
	return fallback
}

func (p *ProxyInferenceProvider) chatOpenAI(ctx context.Context, endpoint, modelName string, messages []inference.Message, opts inference.ChatOptions) (*inference.ChatResponse, error) {
	msgs := make([]openAIChatMsg, len(messages))
	for i, m := range messages {
		msgs[i] = openAIChatMsg{Role: m.Role, Content: m.Content}
	}

	req := openAIChatRequest{
		Model:       modelName,
		Messages:    msgs,
		Temperature: opts.Temperature,
		MaxTokens:   opts.MaxTokens,
		TopP:        opts.TopP,
		Stop:        opts.Stop,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(endpoint, "/") + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer local")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Warn("vLLM returned non-200", "status", resp.StatusCode, "body", string(respBody))
		return nil, fmt.Errorf("vLLM returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var oaiResp openAIChatResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	content := ""
	finishReason := ""
	if len(oaiResp.Choices) > 0 {
		content = oaiResp.Choices[0].Message.Content
		finishReason = oaiResp.Choices[0].FinishReason
	}

	return &inference.ChatResponse{
		Content:           content,
		FinishReason:      finishReason,
		Model:             oaiResp.Model,
		ID:                oaiResp.ID,
		Created:           oaiResp.Created,
		SystemFingerprint: oaiResp.SystemFingerprint,
		Usage: inference.Usage{
			PromptTokens:     oaiResp.Usage.PromptTokens,
			CompletionTokens: oaiResp.Usage.CompletionTokens,
			TotalTokens:      oaiResp.Usage.TotalTokens,
		},
	}, nil
}

func (p *ProxyInferenceProvider) chatOllama(ctx context.Context, endpoint, modelName string, messages []inference.Message, opts inference.ChatOptions) (*inference.ChatResponse, error) {
	msgs := make([]ollamaChatMsg, len(messages))
	for i, m := range messages {
		msgs[i] = ollamaChatMsg{Role: m.Role, Content: m.Content}
	}

	options := map[string]any{}
	if opts.Temperature != nil {
		options["temperature"] = *opts.Temperature
	}
	if opts.MaxTokens != nil {
		options["num_predict"] = *opts.MaxTokens
	}
	if opts.TopP != nil {
		options["top_p"] = *opts.TopP
	}
	if opts.TopK != nil {
		options["top_k"] = *opts.TopK
	}

	req := ollamaChatRequest{
		Model:    modelName,
		Messages: msgs,
		Stream:   false,
		Options:  options,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(endpoint, "/") + "/api/chat"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp ollamaChatResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &inference.ChatResponse{
		Content:      ollamaResp.Message.Content,
		FinishReason: "stop",
		Model:        ollamaResp.Model,
		Usage: inference.Usage{
			PromptTokens:     ollamaResp.PromptEvalCount,
			CompletionTokens: ollamaResp.EvalCount,
			TotalTokens:      ollamaResp.PromptEvalCount + ollamaResp.EvalCount,
		},
	}, nil
}

// Complete sends a text completion request to a running service.
func (p *ProxyInferenceProvider) Complete(ctx context.Context, modelName string, prompt string, opts inference.CompleteOptions) (*inference.CompletionResponse, error) {
	return nil, fmt.Errorf("inference.Complete: not yet implemented for running services")
}

func (p *ProxyInferenceProvider) Embed(ctx context.Context, modelName string, input []string) (*inference.EmbeddingResponse, error) {
	return nil, fmt.Errorf("inference.Embed: not yet implemented for running services")
}

func (p *ProxyInferenceProvider) Transcribe(ctx context.Context, modelName string, audio []byte, language string) (*inference.TranscriptionResponse, error) {
	return nil, fmt.Errorf("inference.Transcribe: not yet implemented for running services")
}

func (p *ProxyInferenceProvider) Synthesize(ctx context.Context, modelName string, text string, voice string) (*inference.AudioResponse, error) {
	return nil, fmt.Errorf("inference.Synthesize: not yet implemented for running services")
}

func (p *ProxyInferenceProvider) GenerateImage(ctx context.Context, modelName string, prompt string, opts inference.ImageOptions) (*inference.ImageGenerationResponse, error) {
	return nil, fmt.Errorf("inference.GenerateImage: not yet implemented for running services")
}

func (p *ProxyInferenceProvider) GenerateVideo(ctx context.Context, modelName string, prompt string, opts inference.VideoOptions) (*inference.VideoGenerationResponse, error) {
	return nil, fmt.Errorf("inference.GenerateVideo: not yet implemented for running services")
}

func (p *ProxyInferenceProvider) Rerank(ctx context.Context, modelName string, query string, documents []string) (*inference.RerankResponse, error) {
	return nil, fmt.Errorf("inference.Rerank: not yet implemented for running services")
}

func (p *ProxyInferenceProvider) Detect(ctx context.Context, modelName string, image []byte) (*inference.DetectionResponse, error) {
	return nil, fmt.Errorf("inference.Detect: not yet implemented for running services")
}

func (p *ProxyInferenceProvider) ListModels(ctx context.Context, modelType string) ([]inference.InferenceModel, error) {
	return nil, fmt.Errorf("inference.ListModels: not yet implemented for running services")
}

func (p *ProxyInferenceProvider) ListVoices(ctx context.Context, modelName string) ([]inference.Voice, error) {
	return nil, fmt.Errorf("inference.ListVoices: not yet implemented for running services")
}

func (p *ProxyInferenceProvider) ChatStream(ctx context.Context, modelName string, messages []inference.Message, opts inference.ChatOptions, stream chan<- inference.ChatStreamChunk) error {
	return fmt.Errorf("inference.ChatStream: not yet implemented for running services")
}

func (p *ProxyInferenceProvider) CompleteStream(ctx context.Context, modelName string, prompt string, opts inference.CompleteOptions, stream chan<- inference.CompleteStreamChunk) error {
	return fmt.Errorf("inference.CompleteStream: not yet implemented for running services")
}
