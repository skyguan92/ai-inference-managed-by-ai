package huggingface

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
)

type Provider struct {
	client      *Client
	token       string
	baseURL     string
	httpClient  *http.Client
	downloadDir string
	mu          sync.RWMutex
	modelCache  map[string]*model.Model
}

type ProviderOption func(*Provider)

func WithToken(token string) ProviderOption {
	return func(p *Provider) {
		p.token = token
	}
}

func WithBaseURL(baseURL string) ProviderOption {
	return func(p *Provider) {
		p.baseURL = baseURL
	}
}

func WithHTTPClient(httpClient *http.Client) ProviderOption {
	return func(p *Provider) {
		p.httpClient = httpClient
	}
}

func WithDownloadDir(dir string) ProviderOption {
	return func(p *Provider) {
		p.downloadDir = dir
	}
}

func NewProvider(opts ...ProviderOption) *Provider {
	p := &Provider{
		baseURL:     "https://huggingface.co",
		downloadDir: "/tmp/aima-models",
		modelCache:  make(map[string]*model.Model),
		httpClient: &http.Client{
			Timeout: 30 * time.Minute,
		},
	}

	for _, opt := range opts {
		opt(p)
	}

	p.client = NewClientWithBaseURL(p.baseURL, p.token)
	if p.httpClient != nil {
		p.client.SetHTTPClient(p.httpClient)
	}

	return p
}

func NewProviderWithClient(client *Client) *Provider {
	return &Provider{
		client:      client,
		baseURL:     client.baseURL,
		token:       client.token,
		downloadDir: "/tmp/aima-models",
		modelCache:  make(map[string]*model.Model),
		httpClient:  client.httpClient,
	}
}

func (p *Provider) Client() *Client {
	return p.client
}

func (p *Provider) Pull(ctx context.Context, source, repo, tag string, progressCh chan<- model.PullProgress) (*model.Model, error) {
	if source != "" && source != "huggingface" && source != "hf" {
		return nil, fmt.Errorf("unsupported source: %s", source)
	}

	revision := tag
	if revision == "" {
		revision = "main"
	}

	info, err := p.client.GetModelInfo(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("get model info for %s: %w", repo, err)
	}

	now := time.Now().Unix()
	m := &model.Model{
		ID:        "model-" + uuid.New().String()[:8],
		Name:      info.ModelID,
		Source:    "huggingface",
		Status:    model.StatusPulling,
		CreatedAt: now,
		UpdatedAt: now,
		Type:      model.ModelType(DetectModelType(info)),
	}

	downloadDir := filepath.Join(p.downloadDir, strings.ReplaceAll(repo, "/", "_"))
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return nil, fmt.Errorf("create download directory: %w", err)
	}

	var totalSize int64
	var downloadedSize int64
	var filesToDownload []Sibling

	for _, sibling := range info.Siblings {
		filename := sibling.Rfilename
		ext := filepath.Ext(filename)
		switch ext {
		case ".gguf", ".safetensors", ".onnx", ".bin", ".pt", ".pth":
			filesToDownload = append(filesToDownload, sibling)
			if sibling.LFS != nil {
				totalSize += sibling.LFS.Size
			}
		}
	}

	if len(filesToDownload) == 0 {
		for _, sibling := range info.Siblings {
			if strings.Contains(sibling.Rfilename, ".json") ||
				strings.Contains(sibling.Rfilename, "config") ||
				strings.Contains(sibling.Rfilename, "tokenizer") {
				filesToDownload = append(filesToDownload, sibling)
			}
		}
	}

	if len(filesToDownload) == 0 {
		return nil, fmt.Errorf("no downloadable model files found in repository")
	}

	for _, file := range filesToDownload {
		filename := file.Rfilename
		destPath := filepath.Join(downloadDir, filepath.Base(filename))

		if progressCh != nil {
			progressCh <- model.PullProgress{
				ModelID: m.ID,
				Status:  fmt.Sprintf("downloading %s", filename),
			}
		}

		fileProgressCh := make(chan int64, 10)
		var fileDownloaded int64
		var fileTotal int64

		go func() {
			for progress := range fileProgressCh {
				if progressCh != nil {
					progressCh <- model.PullProgress{
						ModelID:    m.ID,
						Status:     fmt.Sprintf("downloading %s", filename),
						Progress:   float64(downloadedSize+progress) / float64(totalSize) * 100,
						BytesTotal: totalSize,
						BytesDone:  downloadedSize + progress,
					}
				}
			}
		}()

		fileTotal, err = p.downloadFile(ctx, repo, filename, revision, destPath, fileProgressCh)
		if err != nil {
			m.Status = model.StatusError
			if progressCh != nil {
				progressCh <- model.PullProgress{
					ModelID: m.ID,
					Status:  "error",
					Error:   err.Error(),
				}
			}
			return nil, fmt.Errorf("download file %s: %w", filename, err)
		}

		fileDownloaded = fileTotal
		downloadedSize += fileDownloaded
		close(fileProgressCh)
	}

	if len(filesToDownload) > 0 {
		mainFile := filesToDownload[0].Rfilename
		m.Format = model.ModelFormat(GetModelFormat(mainFile))
		if m.Format == "" {
			m.Format = model.FormatSafetensors
		}
	}

	m.Path = downloadDir
	m.Size = downloadedSize
	m.Status = model.StatusReady
	m.UpdatedAt = time.Now().Unix()

	if info.Safetensors != nil && info.Safetensors.Total > 0 {
		m.Checksum = fmt.Sprintf("safetensors:%d", info.Safetensors.Total)
	}

	p.mu.Lock()
	p.modelCache[repo] = m
	p.mu.Unlock()

	if progressCh != nil {
		progressCh <- model.PullProgress{
			ModelID:  m.ID,
			Status:   "completed",
			Progress: 100,
		}
	}

	return m, nil
}

func (p *Provider) downloadFile(ctx context.Context, repo, filename, revision, destPath string, progressCh chan<- int64) (int64, error) {
	reader, _, err := p.client.DownloadFile(ctx, repo, filename, revision, func(downloaded, total int64) {
		if progressCh != nil {
			progressCh <- downloaded
		}
	})
	if err != nil {
		return 0, err
	}
	defer reader.Close()

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return 0, fmt.Errorf("create directory: %w", err)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return 0, fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	written, err := io.Copy(file, reader)
	if err != nil {
		return 0, fmt.Errorf("write file: %w", err)
	}

	return written, nil
}

func (p *Provider) Search(ctx context.Context, query string, source string, modelType model.ModelType, limit int) ([]model.ModelSearchResult, error) {
	filter := make(map[string]string)
	if modelType != "" {
		task := modelTypeToPipelineTag(modelType)
		if task != "" {
			filter["task"] = task
		}
	}

	if limit <= 0 {
		limit = 20
	}

	resp, err := p.client.SearchModels(ctx, query, filter, limit, 0)
	if err != nil {
		return nil, fmt.Errorf("search models: %w", err)
	}

	results := make([]model.ModelSearchResult, 0, len(resp.Items))
	for _, item := range resp.Items {
		typ := model.ModelTypeLLM
		if detected := DetectModelType(&item); detected != "" {
			typ = model.ModelType(detected)
		}

		if modelType != "" && typ != modelType {
			continue
		}

		description := ""
		if cardData, ok := item.CardData["description"].(string); ok {
			description = cardData
		} else if cardData, ok := item.CardData["summary"].(string); ok {
			description = cardData
		}

		results = append(results, model.ModelSearchResult{
			ID:          item.ModelID,
			Name:        item.ModelID,
			Type:        typ,
			Source:      "huggingface",
			Description: description,
			Downloads:   item.Downloads,
			Tags:        item.Tags,
		})
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
	var detectedFormat model.ModelFormat
	var detectedType model.ModelType = model.ModelTypeLLM

	if fileInfo.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, fmt.Errorf("read directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			switch {
			case strings.HasSuffix(name, ".safetensors"):
				modelName = strings.TrimSuffix(name, ".safetensors")
				detectedFormat = model.FormatSafetensors
				break
			case strings.HasSuffix(name, ".gguf"):
				modelName = strings.TrimSuffix(name, ".gguf")
				detectedFormat = model.FormatGGUF
				break
			case strings.HasSuffix(name, ".onnx"):
				modelName = strings.TrimSuffix(name, ".onnx")
				detectedFormat = model.FormatONNX
				break
			}
		}

		if modelName == "" {
			modelName = filepath.Base(path)
		}
	} else {
		modelName = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		detectedFormat = model.ModelFormat(GetModelFormat(filepath.Base(path)))
	}

	if detectedFormat == "" {
		detectedFormat = model.FormatSafetensors
	}

	now := time.Now().Unix()
	m := &model.Model{
		ID:        "model-" + uuid.New().String()[:8],
		Name:      modelName,
		Type:      detectedType,
		Format:    detectedFormat,
		Status:    model.StatusReady,
		Source:    "local",
		Path:      path,
		Size:      fileInfo.Size(),
		CreatedAt: now,
		UpdatedAt: now,
	}

	return m, nil
}

func (p *Provider) Verify(ctx context.Context, modelID string, checksum string) (*model.VerificationResult, error) {
	var m *model.Model
	var repo string

	p.mu.RLock()
	for r, cached := range p.modelCache {
		if cached.ID == modelID {
			m = cached
			repo = r
			break
		}
	}
	p.mu.RUnlock()

	if m == nil {
		return &model.VerificationResult{
			Valid:  false,
			Issues: []string{fmt.Sprintf("model not found: %s", modelID)},
		}, nil
	}

	issues := []string{}

	if m.Path == "" {
		return &model.VerificationResult{
			Valid:  false,
			Issues: []string{"model has no local path"},
		}, nil
	}

	if _, err := os.Stat(m.Path); os.IsNotExist(err) {
		return &model.VerificationResult{
			Valid:  false,
			Issues: []string{fmt.Sprintf("model path does not exist: %s", m.Path)},
		}, nil
	}

	if checksum != "" {
		verified, err := p.verifyChecksum(m.Path, checksum)
		if err != nil {
			issues = append(issues, fmt.Sprintf("checksum verification failed: %v", err))
		} else if !verified {
			issues = append(issues, "checksum mismatch")
		}
	}

	if repo != "" {
		_, err := p.client.GetModelInfo(ctx, repo)
		if err != nil {
			issues = append(issues, "repository not accessible on HuggingFace Hub")
		}
	}

	if len(issues) == 0 {
		return &model.VerificationResult{Valid: true}, nil
	}

	return &model.VerificationResult{
		Valid:  false,
		Issues: issues,
	}, nil
}

func (p *Provider) verifyChecksum(path, expectedChecksum string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	if strings.HasPrefix(expectedChecksum, "size:") {
		expectedSize := parseInt64(strings.TrimPrefix(expectedChecksum, "size:"))
		return fileInfo.Size() == expectedSize, nil
	}

	if strings.HasPrefix(expectedChecksum, "safetensors:") {
		return true, nil
	}

	if strings.HasPrefix(expectedChecksum, "sha256:") {
		expectedHash := strings.TrimPrefix(expectedChecksum, "sha256:")
		hash, err := p.calculateFileHash(path)
		if err != nil {
			return false, err
		}
		return hash == expectedHash, nil
	}

	return true, nil
}

func (p *Provider) calculateFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (p *Provider) EstimateResources(ctx context.Context, modelID string) (*model.ModelRequirements, error) {
	var repo string

	p.mu.RLock()
	for r, m := range p.modelCache {
		if m.ID == modelID {
			repo = r
			break
		}
	}
	p.mu.RUnlock()

	if repo == "" {
		return nil, fmt.Errorf("model not found: %s", modelID)
	}

	info, err := p.client.GetModelInfo(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("get model info: %w", err)
	}

	var totalParams int64
	if info.Safetensors != nil && info.Safetensors.Total > 0 {
		totalParams = info.Safetensors.Total
	}

	var totalSize int64
	for _, sibling := range info.Siblings {
		if sibling.LFS != nil {
			totalSize += sibling.LFS.Size
		}
	}

	var memMin, memRec int64

	if totalSize > 0 {
		memMin = int64(float64(totalSize) * 1.2)
		memRec = int64(float64(totalSize) * 1.5)
	} else if totalParams > 0 {
		bytesPerParam := int64(2)
		if hasQuantizationTag(info.Tags) {
			bytesPerParam = 1
		}
		memMin = totalParams * bytesPerParam
		memRec = int64(float64(memMin) * 1.3)
	} else {
		memMin = 4 * 1024 * 1024 * 1024
		memRec = 8 * 1024 * 1024 * 1024
	}

	return &model.ModelRequirements{
		MemoryMin:         memMin,
		MemoryRecommended: memRec,
		GPUMemory:         memMin,
	}, nil
}

func modelTypeToPipelineTag(mt model.ModelType) string {
	switch mt {
	case model.ModelTypeLLM:
		return "text-generation"
	case model.ModelTypeVLM:
		return "image-text-to-text"
	case model.ModelTypeASR:
		return "automatic-speech-recognition"
	case model.ModelTypeTTS:
		return "text-to-speech"
	case model.ModelTypeEmbedding:
		return "feature-extraction"
	case model.ModelTypeDiffusion:
		return "text-to-image"
	case model.ModelTypeVideoGen:
		return "text-to-video"
	case model.ModelTypeDetection:
		return "object-detection"
	case model.ModelTypeRerank:
		return "text-ranking"
	default:
		return ""
	}
}

func hasQuantizationTag(tags []string) bool {
	for _, tag := range tags {
		if strings.Contains(tag, "quantized") ||
			strings.Contains(tag, "4bit") ||
			strings.Contains(tag, "8bit") ||
			strings.Contains(tag, "gptq") ||
			strings.Contains(tag, "awq") ||
			strings.Contains(tag, "gguf") {
			return true
		}
	}
	return false
}

func parseInt64(s string) int64 {
	var result int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int64(c-'0')
		}
	}
	return result
}
