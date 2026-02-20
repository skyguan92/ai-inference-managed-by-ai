package huggingface

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
)

func TestClient_New(t *testing.T) {
	t.Run("with token", func(t *testing.T) {
		client := NewClient("test-token")
		if client.baseURL != "https://huggingface.co" {
			t.Errorf("expected baseURL https://huggingface.co, got %s", client.baseURL)
		}
		if client.token != "test-token" {
			t.Errorf("expected token test-token, got %s", client.token)
		}
		if client.httpClient == nil {
			t.Error("expected httpClient to be initialized")
		}
	})

	t.Run("with base URL", func(t *testing.T) {
		client := NewClientWithBaseURL("https://custom.hf.co", "token")
		if client.baseURL != "https://custom.hf.co" {
			t.Errorf("expected baseURL https://custom.hf.co, got %s", client.baseURL)
		}
	})

	t.Run("set http client", func(t *testing.T) {
		client := NewClient("token")
		customHTTPClient := &http.Client{}
		client.SetHTTPClient(customHTTPClient)
		if client.httpClient != customHTTPClient {
			t.Error("expected httpClient to be set")
		}
	})
}

func TestClient_GetModelInfo(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockInfo := ModelInfo{
			ID:          "123",
			ModelID:     "test-org/test-model",
			Author:      "test-author",
			PipelineTag: "text-generation",
			Tags:        []string{"pytorch", "llm"},
			Downloads:   1000,
			Siblings: []Sibling{
				{Rfilename: "config.json"},
				{Rfilename: "model.safetensors", LFS: &LFSInfo{Size: 1000000}},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/models/test-org/test-model" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Header.Get("Authorization") != "Bearer test-token" {
				t.Errorf("expected Authorization header")
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(mockInfo)
		}))
		defer server.Close()

		client := NewClientWithBaseURL(server.URL, "test-token")
		client.SetHTTPClient(server.Client())

		info, err := client.GetModelInfo(context.Background(), "test-org/test-model")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.ModelID != "test-org/test-model" {
			t.Errorf("expected model ID test-org/test-model, got %s", info.ModelID)
		}
		if len(info.Siblings) != 2 {
			t.Errorf("expected 2 siblings, got %d", len(info.Siblings))
		}
	})

	t.Run("error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(ErrorResponse{Error: "Model not found"})
		}))
		defer server.Close()

		client := NewClientWithBaseURL(server.URL, "test-token")
		client.SetHTTPClient(server.Client())

		_, err := client.GetModelInfo(context.Background(), "nonexistent/model")
		if err == nil {
			t.Error("expected error for nonexistent model")
		}
		if !strings.Contains(err.Error(), "Model not found") {
			t.Errorf("expected 'Model not found' error, got: %v", err)
		}
	})
}

func TestClient_SearchModels(t *testing.T) {
	t.Run("success with query", func(t *testing.T) {
		mockResp := SearchResponse{
			Items: []ModelInfo{
				{ModelID: "model1", Downloads: 100},
				{ModelID: "model2", Downloads: 200},
			},
			TotalItems: 2,
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/models" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			query := r.URL.Query().Get("search")
			if query != "llama" {
				t.Errorf("expected search query 'llama', got %s", query)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(mockResp)
		}))
		defer server.Close()

		client := NewClientWithBaseURL(server.URL, "test-token")
		client.SetHTTPClient(server.Client())

		resp, err := client.SearchModels(context.Background(), "llama", nil, 10, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Items) != 2 {
			t.Errorf("expected 2 items, got %d", len(resp.Items))
		}
	})

	t.Run("success with filter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			filters := r.URL.Query()["filter"]
			found := false
			for _, f := range filters {
				if f == "task:text-generation" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected filter 'task:text-generation', got %v", filters)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(SearchResponse{})
		}))
		defer server.Close()

		client := NewClientWithBaseURL(server.URL, "test-token")
		client.SetHTTPClient(server.Client())

		filter := map[string]string{"task": "text-generation"}
		_, err := client.SearchModels(context.Background(), "", filter, 10, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestProvider_New(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		p := NewProvider()
		if p.baseURL != "https://huggingface.co" {
			t.Errorf("expected default baseURL, got %s", p.baseURL)
		}
		if p.downloadDir != "/tmp/aima-models" {
			t.Errorf("expected default downloadDir, got %s", p.downloadDir)
		}
		if p.client == nil {
			t.Error("expected client to be initialized")
		}
	})

	t.Run("with options", func(t *testing.T) {
		p := NewProvider(
			WithToken("test-token"),
			WithBaseURL("https://custom.hf.co"),
			WithDownloadDir("/custom/dir"),
		)
		if p.token != "test-token" {
			t.Errorf("expected token test-token, got %s", p.token)
		}
		if p.baseURL != "https://custom.hf.co" {
			t.Errorf("expected custom baseURL, got %s", p.baseURL)
		}
		if p.downloadDir != "/custom/dir" {
			t.Errorf("expected custom downloadDir, got %s", p.downloadDir)
		}
	})

	t.Run("with client", func(t *testing.T) {
		client := NewClient("custom-token")
		p := NewProviderWithClient(client)
		if p.client != client {
			t.Error("expected client to be set")
		}
	})
}

func TestProvider_Pull(t *testing.T) {
	t.Run("unsupported source", func(t *testing.T) {
		p := NewProvider()
		_, err := p.Pull(context.Background(), "ollama", "test/model", "", nil)
		if err == nil {
			t.Error("expected error for unsupported source")
		}
		if !strings.Contains(err.Error(), "unsupported source") {
			t.Errorf("expected unsupported source error, got: %v", err)
		}
	})

	t.Run("success pull gguf model", func(t *testing.T) {
		mockInfo := ModelInfo{
			ModelID:     "test-org/test-model",
			PipelineTag: "text-generation",
			Tags:        []string{"gguf"},
			Siblings: []Sibling{
				{Rfilename: "config.json"},
				{Rfilename: "model.gguf", LFS: &LFSInfo{Size: 100}},
			},
		}

		fileContent := []byte("test model content")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/models/test-org/test-model":
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(mockInfo)
			case "/test-org/test-model/resolve/main/model.gguf":
				w.Header().Set("Content-Length", "18")
				_, _ = w.Write(fileContent)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		tmpDir := t.TempDir()
		p := NewProvider(
			WithBaseURL(server.URL),
			WithDownloadDir(tmpDir),
		)
		p.client.SetHTTPClient(server.Client())

		m, err := p.Pull(context.Background(), "huggingface", "test-org/test-model", "", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Status != model.StatusReady {
			t.Errorf("expected status ready, got %s", m.Status)
		}
		if m.Source != "huggingface" {
			t.Errorf("expected source huggingface, got %s", m.Source)
		}
		if m.Format != model.FormatGGUF {
			t.Errorf("expected format gguf, got %s", m.Format)
		}
	})

	t.Run("pull with custom revision", func(t *testing.T) {
		mockInfo := ModelInfo{
			ModelID:     "test-org/test-model",
			PipelineTag: "text-generation",
			Siblings: []Sibling{
				{Rfilename: "config.json"},
				{Rfilename: "model.safetensors", LFS: &LFSInfo{Size: 100}},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/models/test-org/test-model" {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(mockInfo)
				return
			}
			if strings.Contains(r.URL.Path, "/resolve/v1.0/") {
				w.Header().Set("Content-Length", "8")
				_, _ = w.Write([]byte("testfile"))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		tmpDir := t.TempDir()
		p := NewProvider(
			WithBaseURL(server.URL),
			WithDownloadDir(tmpDir),
		)
		p.client.SetHTTPClient(server.Client())

		m, err := p.Pull(context.Background(), "hf", "test-org/test-model", "v1.0", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Status != model.StatusReady {
			t.Errorf("expected status ready, got %s", m.Status)
		}
	})
}

func TestProvider_Search(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockResp := SearchResponse{
			Items: []ModelInfo{
				{
					ModelID:     "test-org/model1",
					PipelineTag: "text-generation",
					Downloads:   1000,
					Tags:        []string{"pytorch"},
					CardData:    map[string]any{"description": "Test model 1"},
				},
				{
					ModelID:     "test-org/model2",
					PipelineTag: "feature-extraction",
					Downloads:   500,
					Tags:        []string{"sentence-transformers"},
					CardData:    map[string]any{"summary": "Test embedding model"},
				},
			},
			TotalItems: 2,
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(mockResp)
		}))
		defer server.Close()

		p := NewProvider(WithBaseURL(server.URL))
		p.client.SetHTTPClient(server.Client())

		results, err := p.Search(context.Background(), "test", "", "", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}
		if results[0].Source != "huggingface" {
			t.Errorf("expected source huggingface, got %s", results[0].Source)
		}
	})

	t.Run("with model type filter", func(t *testing.T) {
		mockResp := SearchResponse{
			Items: []ModelInfo{
				{
					ModelID:     "test-org/embedding-model",
					PipelineTag: "feature-extraction",
					Downloads:   100,
					Tags:        []string{"embeddings"},
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(mockResp)
		}))
		defer server.Close()

		p := NewProvider(WithBaseURL(server.URL))
		p.client.SetHTTPClient(server.Client())

		results, err := p.Search(context.Background(), "", "", model.ModelTypeEmbedding, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("expected 1 result, got %d", len(results))
		}
		if results[0].Type != model.ModelTypeEmbedding {
			t.Errorf("expected type embedding, got %s", results[0].Type)
		}
	})
}

func TestProvider_DetectModelType(t *testing.T) {
	tests := []struct {
		name     string
		info     *ModelInfo
		expected string
	}{
		{
			name:     "text-generation pipeline",
			info:     &ModelInfo{PipelineTag: "text-generation"},
			expected: "llm",
		},
		{
			name:     "image-text-to-text pipeline",
			info:     &ModelInfo{PipelineTag: "image-text-to-text"},
			expected: "vlm",
		},
		{
			name:     "automatic-speech-recognition pipeline",
			info:     &ModelInfo{PipelineTag: "automatic-speech-recognition"},
			expected: "asr",
		},
		{
			name:     "text-to-speech pipeline",
			info:     &ModelInfo{PipelineTag: "text-to-speech"},
			expected: "tts",
		},
		{
			name:     "feature-extraction pipeline",
			info:     &ModelInfo{PipelineTag: "feature-extraction"},
			expected: "embedding",
		},
		{
			name:     "text-to-image pipeline",
			info:     &ModelInfo{PipelineTag: "text-to-image"},
			expected: "diffusion",
		},
		{
			name:     "text-to-video pipeline",
			info:     &ModelInfo{PipelineTag: "text-to-video"},
			expected: "video_gen",
		},
		{
			name:     "object-detection pipeline",
			info:     &ModelInfo{PipelineTag: "object-detection"},
			expected: "detection",
		},
		{
			name:     "reranking pipeline",
			info:     &ModelInfo{PipelineTag: "reranking"},
			expected: "rerank",
		},
		{
			name:     "tag-based llm",
			info:     &ModelInfo{Tags: []string{"causal-lm", "pytorch"}},
			expected: "llm",
		},
		{
			name:     "tag-based vlm",
			info:     &ModelInfo{Tags: []string{"vision-language"}},
			expected: "vlm",
		},
		{
			name:     "tag-based asr",
			info:     &ModelInfo{Tags: []string{"speech-recognition"}},
			expected: "asr",
		},
		{
			name:     "tag-based diffusion",
			info:     &ModelInfo{Tags: []string{"stable-diffusion"}},
			expected: "diffusion",
		},
		{
			name:     "default to llm",
			info:     &ModelInfo{},
			expected: "llm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectModelType(tt.info)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestClient_GetFileURL(t *testing.T) {
	client := NewClient("token")

	tests := []struct {
		name     string
		repoID   string
		filename string
		revision string
		expected string
	}{
		{
			name:     "default revision",
			repoID:   "org/model",
			filename: "config.json",
			revision: "",
			expected: "https://huggingface.co/org/model/resolve/main/config.json",
		},
		{
			name:     "custom revision",
			repoID:   "org/model",
			filename: "model.gguf",
			revision: "v1.0",
			expected: "https://huggingface.co/org/model/resolve/v1.0/model.gguf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := client.GetFileURL(tt.repoID, tt.filename, tt.revision)
			if url != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, url)
			}
		})
	}
}

func TestGetModelFormat(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"model.gguf", "gguf"},
		{"model.safetensors", "safetensors"},
		{"model.onnx", "onnx"},
		{"model.engine", "tensorrt"},
		{"model.plan", "tensorrt"},
		{"model.bin", "pytorch"},
		{"model.pt", "pytorch"},
		{"model.pth", "pytorch"},
		{"unknown.txt", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := GetModelFormat(tt.filename)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestClient_DownloadFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		content := []byte("test file content for download")
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/resolve/main/model.gguf") {
				w.Header().Set("Content-Length", "30")
				_, _ = w.Write(content)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := NewClientWithBaseURL(server.URL, "token")
		client.SetHTTPClient(server.Client())

		var progressCalls int
		reader, size, err := client.DownloadFile(context.Background(), "org/model", "model.gguf", "main", func(downloaded, total int64) {
			progressCalls++
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = reader.Close() }()

		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("unexpected read error: %v", err)
		}
		if string(data) != string(content) {
			t.Errorf("content mismatch")
		}
		if size != 30 {
			t.Errorf("expected size 30, got %d", size)
		}
		if progressCalls == 0 {
			t.Error("expected progress callback to be called")
		}
	})

	t.Run("error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("access denied"))
		}))
		defer server.Close()

		client := NewClientWithBaseURL(server.URL, "token")
		client.SetHTTPClient(server.Client())

		_, _, err := client.DownloadFile(context.Background(), "org/model", "model.gguf", "main", nil)
		if err == nil {
			t.Error("expected error for forbidden access")
		}
	})
}

func TestProvider_ImportLocal(t *testing.T) {
	t.Run("import directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		modelFile := filepath.Join(tmpDir, "test-model.safetensors")
		if err := os.WriteFile(modelFile, []byte("fake model"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		p := NewProvider()
		m, err := p.ImportLocal(context.Background(), tmpDir, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Name != "test-model" {
			t.Errorf("expected name test-model, got %s", m.Name)
		}
		if m.Format != model.FormatSafetensors {
			t.Errorf("expected format safetensors, got %s", m.Format)
		}
		if m.Source != "local" {
			t.Errorf("expected source local, got %s", m.Source)
		}
	})

	t.Run("import file", func(t *testing.T) {
		tmpDir := t.TempDir()
		modelFile := filepath.Join(tmpDir, "my-model.gguf")
		if err := os.WriteFile(modelFile, []byte("fake gguf"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		p := NewProvider()
		m, err := p.ImportLocal(context.Background(), modelFile, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Name != "my-model" {
			t.Errorf("expected name my-model, got %s", m.Name)
		}
		if m.Format != model.FormatGGUF {
			t.Errorf("expected format gguf, got %s", m.Format)
		}
	})

	t.Run("nonexistent path", func(t *testing.T) {
		p := NewProvider()
		_, err := p.ImportLocal(context.Background(), "/nonexistent/path", true)
		if err == nil {
			t.Error("expected error for nonexistent path")
		}
	})
}

func TestProvider_Verify(t *testing.T) {
	t.Run("model not in cache", func(t *testing.T) {
		p := NewProvider()
		result, err := p.Verify(context.Background(), "nonexistent-id", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Valid {
			t.Error("expected invalid result for nonexistent model")
		}
		if len(result.Issues) == 0 {
			t.Error("expected issues for nonexistent model")
		}
	})

	t.Run("model in cache with valid path", func(t *testing.T) {
		tmpDir := t.TempDir()
		modelPath := filepath.Join(tmpDir, "model.gguf")
		if err := os.WriteFile(modelPath, []byte("fake model"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		mockInfo := ModelInfo{
			ModelID: "test-org/test-model",
			Siblings: []Sibling{
				{Rfilename: "model.gguf", LFS: &LFSInfo{Size: 10}},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(mockInfo)
		}))
		defer server.Close()

		p := NewProvider(WithBaseURL(server.URL))
		p.client.SetHTTPClient(server.Client())

		p.mu.Lock()
		p.modelCache["test-org/test-model"] = &model.Model{
			ID:     "model-12345",
			Name:   "test-model",
			Path:   modelPath,
			Status: model.StatusReady,
		}
		p.mu.Unlock()

		result, err := p.Verify(context.Background(), "model-12345", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Valid {
			t.Errorf("expected valid result, got issues: %v", result.Issues)
		}
	})
}
