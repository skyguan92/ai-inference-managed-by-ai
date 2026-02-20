package modelscope

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- NewClient ---

func TestNewClient(t *testing.T) {
	t.Run("sets default base URL", func(t *testing.T) {
		c := NewClient("my-token")
		if c.baseURL != "https://api.modelscope.cn" {
			t.Errorf("expected default base URL, got %s", c.baseURL)
		}
	})

	t.Run("stores token", func(t *testing.T) {
		c := NewClient("test-token-123")
		if c.token != "test-token-123" {
			t.Errorf("expected token 'test-token-123', got %s", c.token)
		}
	})

	t.Run("empty token", func(t *testing.T) {
		c := NewClient("")
		if c == nil {
			t.Fatal("expected non-nil client")
		}
		if c.token != "" {
			t.Errorf("expected empty token, got %s", c.token)
		}
	})
}

func TestNewClientWithBaseURL(t *testing.T) {
	c := NewClientWithBaseURL("http://custom-host:8080", "token-abc")
	if c.baseURL != "http://custom-host:8080" {
		t.Errorf("expected base URL 'http://custom-host:8080', got %s", c.baseURL)
	}
	if c.token != "token-abc" {
		t.Errorf("expected token 'token-abc', got %s", c.token)
	}
}

func TestClient_SetHTTPClient(t *testing.T) {
	c := NewClient("token")
	custom := &http.Client{}
	c.SetHTTPClient(custom)
	if c.httpClient != custom {
		t.Error("expected custom HTTP client to be set")
	}
}

// --- GetModelInfo ---

func TestClient_GetModelInfo(t *testing.T) {
	t.Run("successful response", func(t *testing.T) {
		resp := ModelInfo{
			Code:    "0",
			Message: "OK",
			Data: &ModelDetail{
				ID:          "model-001",
				Name:        "Qwen-7B",
				OriginalName: "qwen/Qwen-7B",
				PipelineTag: "text-generation",
				ModelType:   "LLM",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/api/v1/models/qwen/Qwen-7B") {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		c := NewClientWithBaseURL(server.URL, "")
		c.SetHTTPClient(server.Client())

		info, err := c.GetModelInfo(context.Background(), "qwen/Qwen-7B")
		if err != nil {
			t.Fatalf("GetModelInfo failed: %v", err)
		}
		if info.Data.Name != "Qwen-7B" {
			t.Errorf("expected name 'Qwen-7B', got %s", info.Data.Name)
		}
	})

	t.Run("nil data returns error", func(t *testing.T) {
		resp := ModelInfo{Code: "0", Data: nil}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		c := NewClientWithBaseURL(server.URL, "")
		c.SetHTTPClient(server.Client())

		_, err := c.GetModelInfo(context.Background(), "org/model")
		if err == nil {
			t.Error("expected error when Data is nil")
		}
		if !strings.Contains(err.Error(), "model not found") {
			t.Errorf("expected 'model not found' in error, got: %v", err)
		}
	})

	t.Run("server error with message", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(ErrorResponse{
				Code:    "404",
				Message: "Model does not exist",
			})
		}))
		defer server.Close()

		c := NewClientWithBaseURL(server.URL, "")
		c.SetHTTPClient(server.Client())

		_, err := c.GetModelInfo(context.Background(), "org/nonexistent")
		if err == nil {
			t.Error("expected error for 404")
		}
		if !strings.Contains(err.Error(), "Model does not exist") {
			t.Errorf("expected error message in response, got: %v", err)
		}
	})

	t.Run("server error with error code only", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(ErrorResponse{Code: "INVALID_REQUEST"})
		}))
		defer server.Close()

		c := NewClientWithBaseURL(server.URL, "")
		c.SetHTTPClient(server.Client())

		_, err := c.GetModelInfo(context.Background(), "org/model")
		if err == nil {
			t.Error("expected error for bad request")
		}
	})

	t.Run("sends authorization header when token is set", func(t *testing.T) {
		var gotAuth string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			_ = json.NewEncoder(w).Encode(ModelInfo{Data: &ModelDetail{Name: "model"}})
		}))
		defer server.Close()

		c := NewClientWithBaseURL(server.URL, "my-secret-token")
		c.SetHTTPClient(server.Client())

		_, _ = c.GetModelInfo(context.Background(), "org/model")
		if gotAuth != "Bearer my-secret-token" {
			t.Errorf("expected 'Bearer my-secret-token', got %q", gotAuth)
		}
	})

	t.Run("no authorization header when token is empty", func(t *testing.T) {
		var gotAuth string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			_ = json.NewEncoder(w).Encode(ModelInfo{Data: &ModelDetail{Name: "model"}})
		}))
		defer server.Close()

		c := NewClientWithBaseURL(server.URL, "")
		c.SetHTTPClient(server.Client())

		_, _ = c.GetModelInfo(context.Background(), "org/model")
		if gotAuth != "" {
			t.Errorf("expected no Authorization header, got %q", gotAuth)
		}
	})
}

// --- SearchModels ---

func TestClient_SearchModels(t *testing.T) {
	t.Run("basic search", func(t *testing.T) {
		resp := SearchResponse{
			Code:    "0",
			Message: "OK",
			Data: &SearchData{
				Total:    2,
				Page:     1,
				PageSize: 20,
				Data: []ModelItem{
					{ID: "1", Name: "Qwen-7B", PipelineTag: "text-generation"},
					{ID: "2", Name: "Qwen-14B", PipelineTag: "text-generation"},
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			query := r.URL.Query()
			if query.Get("Search") != "Qwen" {
				t.Errorf("expected Search=Qwen, got %s", query.Get("Search"))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		c := NewClientWithBaseURL(server.URL, "")
		c.SetHTTPClient(server.Client())

		result, err := c.SearchModels(context.Background(), "Qwen", "", "", 20, 0)
		if err != nil {
			t.Fatalf("SearchModels failed: %v", err)
		}
		if result.Data.Total != 2 {
			t.Errorf("expected 2 total, got %d", result.Data.Total)
		}
		if len(result.Data.Data) != 2 {
			t.Errorf("expected 2 models, got %d", len(result.Data.Data))
		}
	})

	t.Run("with limit and offset", func(t *testing.T) {
		var gotPageSize, gotOffset string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPageSize = r.URL.Query().Get("PageSize")
			gotOffset = r.URL.Query().Get("Offset")
			_ = json.NewEncoder(w).Encode(SearchResponse{Data: &SearchData{}})
		}))
		defer server.Close()

		c := NewClientWithBaseURL(server.URL, "")
		c.SetHTTPClient(server.Client())

		_, _ = c.SearchModels(context.Background(), "", "", "", 10, 5)
		if gotPageSize != "10" {
			t.Errorf("expected PageSize=10, got %s", gotPageSize)
		}
		if gotOffset != "5" {
			t.Errorf("expected Offset=5, got %s", gotOffset)
		}
	})

	t.Run("default page size when limit is zero", func(t *testing.T) {
		var gotPageSize string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPageSize = r.URL.Query().Get("PageSize")
			_ = json.NewEncoder(w).Encode(SearchResponse{Data: &SearchData{}})
		}))
		defer server.Close()

		c := NewClientWithBaseURL(server.URL, "")
		c.SetHTTPClient(server.Client())

		_, _ = c.SearchModels(context.Background(), "", "", "", 0, 0)
		if gotPageSize != "20" {
			t.Errorf("expected default PageSize=20, got %s", gotPageSize)
		}
	})

	t.Run("with model type filter", func(t *testing.T) {
		var gotModelType string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotModelType = r.URL.Query().Get("ModelType")
			_ = json.NewEncoder(w).Encode(SearchResponse{Data: &SearchData{}})
		}))
		defer server.Close()

		c := NewClientWithBaseURL(server.URL, "")
		c.SetHTTPClient(server.Client())

		_, _ = c.SearchModels(context.Background(), "", "LLM", "", 10, 0)
		if gotModelType != "LLM" {
			t.Errorf("expected ModelType=LLM, got %s", gotModelType)
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"Code":"500","Message":"internal error"}`))
		}))
		defer server.Close()

		c := NewClientWithBaseURL(server.URL, "")
		c.SetHTTPClient(server.Client())

		_, err := c.SearchModels(context.Background(), "query", "", "", 10, 0)
		if err == nil {
			t.Error("expected error for server error")
		}
	})
}

// --- GetDownloadURL ---

func TestClient_GetDownloadURL(t *testing.T) {
	t.Run("file with URL field", func(t *testing.T) {
		resp := ModelInfo{
			Data: &ModelDetail{
				Name: "MyModel",
				ModelFileList: []ModelFile{
					{Name: "model.safetensors", URL: "https://cdn.example.com/model.safetensors"},
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		c := NewClientWithBaseURL(server.URL, "")
		c.SetHTTPClient(server.Client())

		url, err := c.GetDownloadURL(context.Background(), "org/model", "model.safetensors", "v1")
		if err != nil {
			t.Fatalf("GetDownloadURL failed: %v", err)
		}
		if url != "https://cdn.example.com/model.safetensors" {
			t.Errorf("expected CDN URL, got %s", url)
		}
	})

	t.Run("file without URL uses constructed URL", func(t *testing.T) {
		resp := ModelInfo{
			Data: &ModelDetail{
				Name: "MyModel",
				ModelFileList: []ModelFile{
					{Name: "model.gguf", URL: ""},
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		c := NewClientWithBaseURL(server.URL, "")
		c.SetHTTPClient(server.Client())

		url, err := c.GetDownloadURL(context.Background(), "org/my-model", "model.gguf", "version1")
		if err != nil {
			t.Fatalf("GetDownloadURL failed: %v", err)
		}
		if !strings.Contains(url, "org/my-model") {
			t.Errorf("expected URL to contain model path, got %s", url)
		}
		if !strings.Contains(url, "model.gguf") {
			t.Errorf("expected URL to contain filename, got %s", url)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		resp := ModelInfo{
			Data: &ModelDetail{
				ModelFileList: []ModelFile{
					{Name: "other-file.bin"},
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		c := NewClientWithBaseURL(server.URL, "")
		c.SetHTTPClient(server.Client())

		_, err := c.GetDownloadURL(context.Background(), "org/model", "missing-file.gguf", "v1")
		if err == nil {
			t.Error("expected error for missing file")
		}
		if !strings.Contains(err.Error(), "file not found") {
			t.Errorf("expected 'file not found' in error, got: %v", err)
		}
	})
}

// --- DownloadFile ---

func TestClient_DownloadFile_Error(t *testing.T) {
	t.Run("download server error", func(t *testing.T) {
		// First request: GetModelInfo (to resolve URL)
		// Second request: actual download (returns 403)
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				// GetModelInfo response
				_ = json.NewEncoder(w).Encode(ModelInfo{
					Data: &ModelDetail{
						ModelFileList: []ModelFile{
							{Name: "model.gguf", URL: ""},
						},
					},
				})
				return
			}
			// Download request - simulate error
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("forbidden"))
		}))
		defer server.Close()

		c := NewClientWithBaseURL(server.URL, "")
		c.SetHTTPClient(server.Client())

		// GetDownloadURL will construct a URL pointing to modelscope.cn, not our server.
		// So DownloadFile with a nil progressFn:
		// Use a server that always returns 403 for all requests
		c2 := NewClientWithBaseURL(server.URL, "")
		c2.SetHTTPClient(server.Client())

		// Override GetDownloadURL to return the server URL directly
		// We can test the error path by making GetModelInfo return a file
		// with a URL that points to our error server
		errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/v1/models") {
				_ = json.NewEncoder(w).Encode(ModelInfo{
					Data: &ModelDetail{
						ModelFileList: []ModelFile{
							// URL points to the same error server's /download path
						},
					},
				})
				return
			}
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("access denied"))
		}))
		defer errorServer.Close()

		// Easiest test: supply a broken model info response that causes GetDownloadURL to fail
		brokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(ErrorResponse{Message: "not found"})
		}))
		defer brokenServer.Close()

		c3 := NewClientWithBaseURL(brokenServer.URL, "")
		c3.SetHTTPClient(brokenServer.Client())

		_, _, err := c3.DownloadFile(context.Background(), "org/model", "model.gguf", "v1", nil)
		if err == nil {
			t.Error("expected error for download with broken model info")
		}
	})
}

// --- GetModelFormat ---

func TestGetModelFormat(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"model.gguf", "gguf"},
		{"model.safetensors", "safetensors"},
		{"model.onnx", "onnx"},
		{"model.bin", "bin"},
		{"model.pt", "pytorch"},
		{"model.pth", "pytorch"},
		{"model.pdparams", "paddle"},
		{"model.unknown", ""},
		{"", ""},
		{"no-extension", ""},
		{"model.GGUF", ""},     // case-sensitive
		{"model.safetensors.bak", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := GetModelFormat(tt.filename)
			if got != tt.expected {
				t.Errorf("GetModelFormat(%q) = %q, want %q", tt.filename, got, tt.expected)
			}
		})
	}
}

// --- DetectModelType ---

func TestDetectModelType(t *testing.T) {
	tests := []struct {
		name        string
		pipelineTag string
		modelType   string
		expected    string
	}{
		// PipelineTag takes precedence
		{"text-generation LLM", "text-generation", "", "llm"},
		{"text2text-generation LLM", "text2text-generation", "", "llm"},
		{"chat LLM", "chat", "", "llm"},
		{"image-text-to-text VLM", "image-text-to-text", "", "vlm"},
		{"visual-question-answering VLM", "visual-question-answering", "", "vlm"},
		{"multimodal VLM", "multimodal", "", "vlm"},
		{"asr pipeline", "automatic-speech-recognition", "", "asr"},
		{"tts pipeline", "text-to-speech", "", "tts"},
		{"embedding pipeline", "feature-extraction", "", "embedding"},
		{"diffusion pipeline", "text-to-image", "", "diffusion"},
		{"video-gen pipeline", "text-to-video", "", "video_gen"},
		{"detection pipeline", "object-detection", "", "detection"},
		{"rerank pipeline", "text-ranking", "", "rerank"},
		// ModelType fallback when PipelineTag empty
		{"LLM model type", "", "LLM", "llm"},
		{"VLM model type", "", "VLM", "vlm"},
		{"ASR model type", "", "ASR", "asr"},
		{"TTS model type", "", "TTS", "tts"},
		{"Embedding model type", "", "Embedding", "embedding"},
		{"Diffusion model type", "", "Diffusion", "diffusion"},
		{"VideoGen model type", "", "VideoGen", "video_gen"},
		{"Detection model type", "", "Detection", "detection"},
		{"Rerank model type", "", "Rerank", "rerank"},
		// Default: unknown → llm
		{"unknown falls back to llm", "unknown-task", "UnknownModel", "llm"},
		{"all empty falls back to llm", "", "", "llm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectModelType(tt.pipelineTag, tt.modelType)
			if got != tt.expected {
				t.Errorf("DetectModelType(%q, %q) = %q, want %q", tt.pipelineTag, tt.modelType, got, tt.expected)
			}
		})
	}
}

// --- progressReader ---

func TestProgressReader(t *testing.T) {
	t.Run("tracks progress", func(t *testing.T) {
		data := []byte("hello world test data")
		var lastDownloaded, lastTotal int64

		pr := &progressReader{
			reader: nopCloser(data),
			total:  int64(len(data)),
			progressFn: func(downloaded, total int64) {
				lastDownloaded = downloaded
				lastTotal = total
			},
		}

		buf := make([]byte, 5)
		n, _ := pr.Read(buf)
		if n != 5 {
			t.Errorf("expected to read 5 bytes, got %d", n)
		}
		if lastDownloaded != 5 {
			t.Errorf("expected downloaded=5, got %d", lastDownloaded)
		}
		if lastTotal != int64(len(data)) {
			t.Errorf("expected total=%d, got %d", len(data), lastTotal)
		}
	})

	t.Run("nil progressFn is safe", func(t *testing.T) {
		pr := &progressReader{
			reader:     nopCloser([]byte("data")),
			total:      4,
			progressFn: nil,
		}
		buf := make([]byte, 4)
		_, err := pr.Read(buf)
		if err != nil && err.Error() != "EOF" {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// helper: wrap []byte as io.ReadCloser
type bytesReadCloser struct {
	data   []byte
	offset int
}

func nopCloser(data []byte) *bytesReadCloser {
	return &bytesReadCloser{data: data}
}

func (b *bytesReadCloser) Read(p []byte) (int, error) {
	if b.offset >= len(b.data) {
		return 0, nil
	}
	n := copy(p, b.data[b.offset:])
	b.offset += n
	return n, nil
}

func (b *bytesReadCloser) Close() error { return nil }
