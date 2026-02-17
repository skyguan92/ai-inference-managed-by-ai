package ollama

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/engine"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/inference"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
)

type Provider struct {
	client     *Client
	baseURL    string
	processes  map[string]*exec.Cmd
	mu         sync.RWMutex
	modelCache map[string]*model.Model
}

func NewProvider(baseURL string) *Provider {
	return &Provider{
		client:     NewClient(baseURL),
		baseURL:    baseURL,
		processes:  make(map[string]*exec.Cmd),
		modelCache: make(map[string]*model.Model),
	}
}

func NewProviderWithClient(client *Client) *Provider {
	return &Provider{
		client:     client,
		processes:  make(map[string]*exec.Cmd),
		modelCache: make(map[string]*model.Model),
	}
}

func (p *Provider) Client() *Client {
	return p.client
}

func (p *Provider) Pull(ctx context.Context, source, repo, tag string, progressCh chan<- model.PullProgress) (*model.Model, error) {
	if source != "" && source != "ollama" {
		return nil, fmt.Errorf("unsupported source: %s", source)
	}

	modelName := repo
	if tag != "" && tag != "latest" {
		modelName = repo + ":" + tag
	} else if !strings.Contains(repo, ":") {
		modelName = repo + ":latest"
	}

	req := &PullRequest{
		Name:   modelName,
		Stream: progressCh != nil,
	}

	now := time.Now().Unix()
	m := &model.Model{
		ID:        "model-" + uuid.New().String()[:8],
		Name:      modelName,
		Type:      model.ModelTypeLLM,
		Format:    model.FormatGGUF,
		Status:    model.StatusPulling,
		Source:    "ollama",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if progressCh != nil {
		err := p.client.Pull(ctx, req, func(resp *PullResponse) error {
			var progress float64
			if resp.Total > 0 {
				progress = float64(resp.Completed) / float64(resp.Total) * 100
			}

			progressCh <- model.PullProgress{
				ModelID:    m.ID,
				Status:     resp.Status,
				Progress:   progress,
				BytesTotal: resp.Total,
				BytesDone:  resp.Completed,
			}

			if resp.Status == "success" {
				m.Status = model.StatusReady
				m.Size = resp.Total
				m.Checksum = resp.Digest
			}

			return nil
		})
		if err != nil {
			m.Status = model.StatusError
			return nil, fmt.Errorf("pull model %s: %w", modelName, err)
		}
	} else {
		if err := p.client.Pull(ctx, req, nil); err != nil {
			m.Status = model.StatusError
			return nil, fmt.Errorf("pull model %s: %w", modelName, err)
		}
		m.Status = model.StatusReady
	}

	m.UpdatedAt = time.Now().Unix()
	p.mu.Lock()
	p.modelCache[modelName] = m
	p.mu.Unlock()

	return m, nil
}

func (p *Provider) Search(ctx context.Context, query string, source string, modelType model.ModelType, limit int) ([]model.ModelSearchResult, error) {
	ollamaModels := []struct {
		name        string
		description string
		modelType   model.ModelType
		downloads   int
	}{
		{"llama3", "Meta Llama 3 - latest generation of Llama models", model.ModelTypeLLM, 1000000},
		{"llama3.1", "Meta Llama 3.1 - improved version with longer context", model.ModelTypeLLM, 800000},
		{"mistral", "Mistral 7B - efficient and powerful language model", model.ModelTypeLLM, 600000},
		{"mixtral", "Mixtral 8x7B - mixture of experts model", model.ModelTypeLLM, 400000},
		{"codellama", "Code Llama - specialized for code generation", model.ModelTypeLLM, 300000},
		{"phi3", "Microsoft Phi-3 - small but capable model", model.ModelTypeLLM, 200000},
		{"gemma", "Google Gemma - open model from Google", model.ModelTypeLLM, 250000},
		{"qwen2", "Alibaba Qwen2 - multilingual model", model.ModelTypeLLM, 180000},
		{"llava", "LLaVA - vision-language model", model.ModelTypeVLM, 150000},
		{"nomic-embed-text", "Nomic embedding model", model.ModelTypeEmbedding, 100000},
		{"mxbai-embed-large", "Large embedding model", model.ModelTypeEmbedding, 80000},
		{"whisper", "OpenAI Whisper - speech recognition", model.ModelTypeASR, 200000},
	}

	var results []model.ModelSearchResult
	queryLower := strings.ToLower(query)

	for _, m := range ollamaModels {
		if query != "" {
			if !strings.Contains(strings.ToLower(m.name), queryLower) &&
				!strings.Contains(strings.ToLower(m.description), queryLower) {
				continue
			}
		}

		if modelType != "" && m.modelType != modelType {
			continue
		}

		results = append(results, model.ModelSearchResult{
			ID:          m.name,
			Name:        m.name,
			Type:        m.modelType,
			Source:      "ollama",
			Description: m.description,
			Downloads:   m.downloads,
		})

		if limit > 0 && len(results) >= limit {
			break
		}
	}

	return results, nil
}

func (p *Provider) ImportLocal(ctx context.Context, path string, autoDetect bool) (*model.Model, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("path does not exist: %s", path)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat path: %w", err)
	}

	var modelName string
	var modelfile string

	if fileInfo.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("read directory: %w", err)
		}
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".gguf") {
				modelName = strings.TrimSuffix(entry.Name(), ".gguf")
				modelfile = filepath.Join(path, entry.Name())
				break
			}
		}
		if modelName == "" {
			return nil, fmt.Errorf("no GGUF file found in directory")
		}
	} else {
		modelName = strings.TrimSuffix(filepath.Base(path), ".gguf")
		modelfile = path
	}

	now := time.Now().Unix()
	m := &model.Model{
		ID:        "model-" + uuid.New().String()[:8],
		Name:      modelName,
		Type:      model.ModelTypeLLM,
		Format:    model.FormatGGUF,
		Status:    model.StatusReady,
		Source:    "local",
		Path:      path,
		Size:      fileInfo.Size(),
		CreatedAt: now,
		UpdatedAt: now,
	}

	absPath, err := filepath.Abs(modelfile)
	if err != nil {
		return m, nil
	}

	cmd := exec.CommandContext(ctx, "ollama", "create", modelName, "-f", absPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("create model with ollama: %w, output: %s", err, string(output))
	}

	m.Source = "ollama"
	return m, nil
}

func (p *Provider) Verify(ctx context.Context, modelID string, checksum string) (*model.VerificationResult, error) {
	modelName := modelID
	if strings.HasPrefix(modelID, "model-") {
		p.mu.RLock()
		for name, m := range p.modelCache {
			if m.ID == modelID {
				modelName = name
				break
			}
		}
		p.mu.RUnlock()
	}

	resp, err := p.client.ShowModel(ctx, &ShowModelRequest{Name: modelName})
	if err != nil {
		return &model.VerificationResult{
			Valid:  false,
			Issues: []string{fmt.Sprintf("model not found: %s", modelName)},
		}, nil
	}

	issues := []string{}

	if checksum != "" && resp.Details.ParameterSize != "" {
	}

	if len(issues) == 0 {
		return &model.VerificationResult{Valid: true}, nil
	}

	return &model.VerificationResult{
		Valid:  false,
		Issues: issues,
	}, nil
}

func (p *Provider) EstimateResources(ctx context.Context, modelID string) (*model.ModelRequirements, error) {
	modelName := modelID
	if strings.HasPrefix(modelID, "model-") {
		p.mu.RLock()
		for name, m := range p.modelCache {
			if m.ID == modelID {
				modelName = name
				break
			}
		}
		p.mu.RUnlock()
	}

	resp, err := p.client.ShowModel(ctx, &ShowModelRequest{Name: modelName})
	if err != nil {
		return nil, fmt.Errorf("get model info: %w", err)
	}

	paramSize := parseParameterSize(resp.Details.ParameterSize)

	var memMin, memRec int64
	switch {
	case paramSize >= 70:
		memMin = 40 * 1024 * 1024 * 1024
		memRec = 80 * 1024 * 1024 * 1024
	case paramSize >= 30:
		memMin = 20 * 1024 * 1024 * 1024
		memRec = 48 * 1024 * 1024 * 1024
	case paramSize >= 13:
		memMin = 8 * 1024 * 1024 * 1024
		memRec = 16 * 1024 * 1024 * 1024
	case paramSize >= 7:
		memMin = 4 * 1024 * 1024 * 1024
		memRec = 8 * 1024 * 1024 * 1024
	default:
		memMin = 2 * 1024 * 1024 * 1024
		memRec = 4 * 1024 * 1024 * 1024
	}

	if resp.Details.QuantizationLevel != "" && resp.Details.QuantizationLevel != "F16" && resp.Details.QuantizationLevel != "F32" {
		memMin = memMin / 3
		memRec = memRec / 3
	}

	return &model.ModelRequirements{
		MemoryMin:         memMin,
		MemoryRecommended: memRec,
		GPUMemory:         memMin,
	}, nil
}

func parseParameterSize(size string) float64 {
	if size == "" {
		return 0
	}
	size = strings.ToLower(strings.TrimSpace(size))
	size = strings.TrimSuffix(size, "b")
	size = strings.TrimSuffix(size, "m")

	if val, err := strconv.ParseFloat(size, 64); err == nil {
		if strings.Contains(strings.ToLower(size), "m") {
			return val / 1000
		}
		return val
	}
	return 0
}

func (p *Provider) Start(ctx context.Context, name string, config map[string]any) (*engine.StartResult, error) {
	if !p.client.IsRunning(ctx) {
		cmd := exec.CommandContext(ctx, "ollama", "serve")
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("start ollama server: %w", err)
		}

		p.mu.Lock()
		p.processes["ollama"] = cmd
		p.mu.Unlock()

		for i := 0; i < 30; i++ {
			time.Sleep(500 * time.Millisecond)
			if p.client.IsRunning(ctx) {
				break
			}
		}
	}

	pid := "ollama-server"
	if p.client.IsRunning(ctx) {
		return &engine.StartResult{
			ProcessID: pid,
			Status:    engine.EngineStatusRunning,
		}, nil
	}

	return nil, fmt.Errorf("failed to start ollama server")
}

func (p *Provider) Stop(ctx context.Context, name string, force bool, timeout int) (*engine.StopResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	cmd, exists := p.processes[name]
	if !exists {
		return &engine.StopResult{Success: true}, nil
	}

	if force {
		if err := cmd.Process.Kill(); err != nil {
			return nil, fmt.Errorf("kill ollama process: %w", err)
		}
	} else {
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			return nil, fmt.Errorf("interrupt ollama process: %w", err)
		}

		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-time.After(time.Duration(timeout) * time.Second):
			if err := cmd.Process.Kill(); err != nil {
				return nil, fmt.Errorf("kill ollama process after timeout: %w", err)
			}
		case <-done:
		}
	}

	delete(p.processes, name)
	return &engine.StopResult{Success: true}, nil
}

func (p *Provider) Install(ctx context.Context, name string, version string) (*engine.InstallResult, error) {
	_, err := exec.LookPath("ollama")
	if err == nil {
		path, _ := exec.LookPath("ollama")
		return &engine.InstallResult{
			Success: true,
			Path:    path,
		}, nil
	}

	return nil, fmt.Errorf("ollama not found, please install from https://ollama.ai")
}

func (p *Provider) GetFeatures(ctx context.Context, name string) (*engine.EngineFeatures, error) {
	return &engine.EngineFeatures{
		SupportsStreaming:    true,
		SupportsBatch:        false,
		SupportsMultimodal:   true,
		SupportsTools:        true,
		SupportsEmbedding:    true,
		MaxConcurrent:        10,
		MaxContextLength:     128000,
		MaxBatchSize:         1,
		SupportsGPULayers:    true,
		SupportsQuantization: true,
	}, nil
}

func (p *Provider) Chat(ctx context.Context, modelName string, messages []inference.Message, opts inference.ChatOptions) (*inference.ChatResponse, error) {
	chatMsgs := make([]ChatMessage, len(messages))
	for i, m := range messages {
		chatMsgs[i] = ChatMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	req := &ChatRequest{
		Model:    modelName,
		Messages: chatMsgs,
		Stream:   false,
		Options:  make(map[string]any),
	}

	if opts.Temperature != nil {
		req.Options["temperature"] = *opts.Temperature
	}
	if opts.MaxTokens != nil {
		req.Options["num_predict"] = *opts.MaxTokens
	}
	if opts.TopP != nil {
		req.Options["top_p"] = *opts.TopP
	}
	if opts.TopK != nil {
		req.Options["top_k"] = *opts.TopK
	}
	if opts.FrequencyPenalty != nil {
		req.Options["frequency_penalty"] = *opts.FrequencyPenalty
	}
	if opts.PresencePenalty != nil {
		req.Options["presence_penalty"] = *opts.PresencePenalty
	}
	if len(opts.Stop) > 0 {
		req.Options["stop"] = opts.Stop
	}

	resp, err := p.client.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("chat: %w", err)
	}

	content := ""
	if resp.Message != nil {
		content = resp.Message.Content
	}

	return &inference.ChatResponse{
		Content:      content,
		FinishReason: "stop",
		Usage: inference.Usage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
		},
		Model:   resp.Model,
		ID:      "chatcmpl-" + uuid.New().String()[:8],
		Created: time.Now().Unix(),
	}, nil
}

func (p *Provider) Complete(ctx context.Context, modelName string, prompt string, opts inference.CompleteOptions) (*inference.CompletionResponse, error) {
	req := &GenerateRequest{
		Model:   modelName,
		Prompt:  prompt,
		Stream:  false,
		Options: make(map[string]any),
	}

	if opts.Temperature != nil {
		req.Options["temperature"] = *opts.Temperature
	}
	if opts.MaxTokens != nil {
		req.Options["num_predict"] = *opts.MaxTokens
	}
	if opts.TopP != nil {
		req.Options["top_p"] = *opts.TopP
	}
	if len(opts.Stop) > 0 {
		req.Options["stop"] = opts.Stop
	}

	resp, err := p.client.Generate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("generate: %w", err)
	}

	return &inference.CompletionResponse{
		Text:         resp.Response,
		FinishReason: "stop",
		Usage: inference.Usage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
		},
	}, nil
}

func (p *Provider) Embed(ctx context.Context, modelName string, input []string) (*inference.EmbeddingResponse, error) {
	embeddings := make([][]float64, len(input))
	totalTokens := 0

	for i, text := range input {
		req := &EmbeddingRequest{
			Model:  modelName,
			Prompt: text,
		}

		resp, err := p.client.Embedding(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("embedding for input %d: %w", i, err)
		}

		embeddings[i] = resp.Embedding
		totalTokens += len(text) / 4
	}

	return &inference.EmbeddingResponse{
		Embeddings: embeddings,
		Usage: inference.Usage{
			PromptTokens: totalTokens,
			TotalTokens:  totalTokens,
		},
	}, nil
}

func (p *Provider) Transcribe(ctx context.Context, modelName string, audio []byte, language string) (*inference.TranscriptionResponse, error) {
	return nil, fmt.Errorf("transcription not supported by ollama provider, use whisper directly")
}

func (p *Provider) Synthesize(ctx context.Context, modelName string, text string, voice string) (*inference.AudioResponse, error) {
	return nil, fmt.Errorf("speech synthesis not supported by ollama provider")
}

func (p *Provider) GenerateImage(ctx context.Context, modelName string, prompt string, opts inference.ImageOptions) (*inference.ImageGenerationResponse, error) {
	return nil, fmt.Errorf("image generation not supported by ollama provider")
}

func (p *Provider) GenerateVideo(ctx context.Context, modelName string, prompt string, opts inference.VideoOptions) (*inference.VideoGenerationResponse, error) {
	return nil, fmt.Errorf("video generation not supported by ollama provider")
}

func (p *Provider) Rerank(ctx context.Context, modelName string, query string, documents []string) (*inference.RerankResponse, error) {
	return nil, fmt.Errorf("reranking not supported by ollama provider")
}

func (p *Provider) Detect(ctx context.Context, modelName string, image []byte) (*inference.DetectionResponse, error) {
	return nil, fmt.Errorf("object detection not supported by ollama provider")
}

func (p *Provider) ListModels(ctx context.Context, modelType string) ([]inference.InferenceModel, error) {
	resp, err := p.client.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}

	models := make([]inference.InferenceModel, len(resp.Models))
	for i, m := range resp.Models {
		typ := "llm"
		if strings.Contains(m.Name, "embed") {
			typ = "embedding"
		} else if strings.Contains(m.Name, "whisper") {
			typ = "asr"
		} else if strings.Contains(m.Name, "llava") || strings.Contains(m.Name, "bakllava") {
			typ = "vlm"
		}

		if modelType != "" && typ != modelType {
			continue
		}

		models[i] = inference.InferenceModel{
			ID:         m.Name,
			Name:       m.Name,
			Type:       typ,
			Provider:   "ollama",
			MaxTokens:  8192,
			Modalities: []string{"text"},
		}
	}

	return models, nil
}

func (p *Provider) ListVoices(ctx context.Context, modelName string) ([]inference.Voice, error) {
	return nil, fmt.Errorf("voice listing not supported by ollama provider")
}
