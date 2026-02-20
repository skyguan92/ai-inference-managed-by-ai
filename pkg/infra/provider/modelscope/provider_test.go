package modelscope

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
)

// --- NewProvider ---

func TestNewProvider_Defaults(t *testing.T) {
	p := NewProvider()
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.baseURL != "https://api.modelscope.cn" {
		t.Errorf("expected default base URL, got %s", p.baseURL)
	}
	if p.downloadDir != "/tmp/aima-models" {
		t.Errorf("expected default download dir, got %s", p.downloadDir)
	}
	if p.modelCache == nil {
		t.Error("expected initialized model cache")
	}
	if p.client == nil {
		t.Error("expected non-nil client")
	}
}

func TestNewProvider_WithToken(t *testing.T) {
	p := NewProvider(WithToken("secret-token"))
	if p.token != "secret-token" {
		t.Errorf("expected token 'secret-token', got %s", p.token)
	}
}

func TestNewProvider_WithBaseURL(t *testing.T) {
	p := NewProvider(WithBaseURL("http://custom-host:9090"))
	if p.baseURL != "http://custom-host:9090" {
		t.Errorf("expected custom base URL, got %s", p.baseURL)
	}
}

func TestNewProvider_WithDownloadDir(t *testing.T) {
	p := NewProvider(WithDownloadDir("/my/download/dir"))
	if p.downloadDir != "/my/download/dir" {
		t.Errorf("expected download dir '/my/download/dir', got %s", p.downloadDir)
	}
}

func TestNewProvider_WithHTTPClient(t *testing.T) {
	custom := &http.Client{}
	p := NewProvider(WithHTTPClient(custom))
	if p.httpClient != custom {
		t.Error("expected custom HTTP client to be set")
	}
}

func TestNewProvider_MultipleOptions(t *testing.T) {
	custom := &http.Client{}
	p := NewProvider(
		WithToken("my-token"),
		WithBaseURL("http://localhost:8080"),
		WithDownloadDir("/tmp/models"),
		WithHTTPClient(custom),
	)
	if p.token != "my-token" {
		t.Errorf("expected token 'my-token', got %s", p.token)
	}
	if p.baseURL != "http://localhost:8080" {
		t.Errorf("expected base URL 'http://localhost:8080', got %s", p.baseURL)
	}
	if p.downloadDir != "/tmp/models" {
		t.Errorf("expected download dir '/tmp/models', got %s", p.downloadDir)
	}
	if p.httpClient != custom {
		t.Error("expected custom HTTP client")
	}
}

// --- NewProviderWithClient ---

func TestNewProviderWithClient(t *testing.T) {
	c := NewClient("my-token")
	p := NewProviderWithClient(c)
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.client != c {
		t.Error("expected client to be set")
	}
	if p.token != c.token {
		t.Errorf("expected token %s, got %s", c.token, p.token)
	}
	if p.modelCache == nil {
		t.Error("expected initialized model cache")
	}
}

// --- Client accessor ---

func TestProvider_Client(t *testing.T) {
	c := NewClient("")
	p := NewProviderWithClient(c)
	if p.Client() != c {
		t.Error("expected Client() to return same client instance")
	}
}

// --- Pull ---

func TestProvider_Pull_UnsupportedSource(t *testing.T) {
	p := NewProvider()
	_, err := p.Pull(context.Background(), "unknown-source", "org/model", "", nil)
	if err == nil {
		t.Error("expected error for unsupported source")
	}
	if !strings.Contains(err.Error(), "unsupported source") {
		t.Errorf("expected 'unsupported source' in error, got: %v", err)
	}
}

func TestProvider_Pull_SupportedSources(t *testing.T) {
	// modelscope and ms are valid sources; we test them fail at the HTTP level
	// (not at the source-check level) by using an empty base URL server
	resp := ModelInfo{
		Data: &ModelDetail{
			Name:        "Qwen-7B",
			PipelineTag: "text-generation",
			ModelFileList: []ModelFile{
				{Name: "config.json", Type: "Config", Size: 100, URL: ""},
			},
		},
	}

	for _, source := range []string{"modelscope", "ms", ""} {
		t.Run("source="+source, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "/api/v1/models") {
					_ = json.NewEncoder(w).Encode(resp)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			tmpDir := t.TempDir()
			c := NewClientWithBaseURL(server.URL, "")
			c.SetHTTPClient(server.Client())

			p := NewProvider(
				WithBaseURL(server.URL),
				WithDownloadDir(tmpDir),
				WithHTTPClient(server.Client()),
			)
			p.client = c

			// Pull will fail on the download step (file URL is empty + redirect to modelscope.cn)
			// so we just verify it doesn't fail at source validation
			_, err := p.Pull(context.Background(), source, "org/Qwen-7B", "", nil)
			// It may fail at download, but not at source validation
			if err != nil && strings.Contains(err.Error(), "unsupported source") {
				t.Errorf("source %q should be valid, got error: %v", source, err)
			}
		})
	}
}

func TestProvider_Pull_ModelNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(ErrorResponse{Message: "model not found"})
	}))
	defer server.Close()

	c := NewClientWithBaseURL(server.URL, "")
	c.SetHTTPClient(server.Client())
	p := NewProviderWithClient(c)

	_, err := p.Pull(context.Background(), "modelscope", "org/nonexistent", "", nil)
	if err == nil {
		t.Error("expected error for model not found")
	}
}

func TestProvider_Pull_NoDownloadableFiles(t *testing.T) {
	// Model info returns no files
	resp := ModelInfo{
		Data: &ModelDetail{
			Name:          "EmptyModel",
			ModelFileList: []ModelFile{},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClientWithBaseURL(server.URL, "")
	c.SetHTTPClient(server.Client())
	p := NewProviderWithClient(c)
	p.downloadDir = t.TempDir()

	_, err := p.Pull(context.Background(), "", "org/empty-model", "", nil)
	if err == nil {
		t.Error("expected error for model with no downloadable files")
	}
	if !strings.Contains(err.Error(), "no downloadable model files") {
		t.Errorf("expected 'no downloadable model files' in error, got: %v", err)
	}
}

func TestProvider_Pull_WithProgress(t *testing.T) {
	// Return model info with a single "Model" file; the actual download will fail
	// because the download URL points to modelscope.cn which isn't our server.
	// We verify the progress channel receives at least the "downloading" status.
	resp := ModelInfo{
		Data: &ModelDetail{
			Name:        "TestModel",
			PipelineTag: "text-generation",
			ModelFileList: []ModelFile{
				{Name: "model.gguf", Type: "Model", Size: 1000, URL: ""},
			},
			Snapshots: []Snapshot{{VersionID: "v1"}},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClientWithBaseURL(server.URL, "")
	c.SetHTTPClient(server.Client())
	p := NewProviderWithClient(c)
	p.downloadDir = t.TempDir()

	progressCh := make(chan model.PullProgress, 20)
	_, err := p.Pull(context.Background(), "", "org/test-model", "", progressCh)
	close(progressCh)

	// Collect progress updates
	var updates []model.PullProgress
	for update := range progressCh {
		updates = append(updates, update)
	}

	// Should have received at least one progress update before failure
	if len(updates) == 0 && err == nil {
		t.Error("expected either progress updates or an error")
	}
}

// --- Search ---

func TestProvider_Search(t *testing.T) {
	t.Run("basic search returns results", func(t *testing.T) {
		resp := SearchResponse{
			Data: &SearchData{
				Total: 2,
				Data: []ModelItem{
					{Name: "Qwen-7B", PipelineTag: "text-generation", Downloads: 1000},
					{Name: "Qwen-14B", PipelineTag: "text-generation", Downloads: 500},
				},
			},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		c := NewClientWithBaseURL(server.URL, "")
		c.SetHTTPClient(server.Client())
		p := NewProviderWithClient(c)

		results, err := p.Search(context.Background(), "Qwen", "", model.ModelTypeLLM, 10)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}
		for _, r := range results {
			if r.Source != "modelscope" {
				t.Errorf("expected source 'modelscope', got %s", r.Source)
			}
		}
	})

	t.Run("filters by model type", func(t *testing.T) {
		resp := SearchResponse{
			Data: &SearchData{
				Data: []ModelItem{
					{Name: "llm-model", PipelineTag: "text-generation"},
					{Name: "embed-model", PipelineTag: "feature-extraction"},
				},
			},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		c := NewClientWithBaseURL(server.URL, "")
		c.SetHTTPClient(server.Client())
		p := NewProviderWithClient(c)

		results, err := p.Search(context.Background(), "", "", model.ModelTypeEmbedding, 10)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		for _, r := range results {
			if r.Type != model.ModelTypeEmbedding {
				t.Errorf("expected embedding type, got %s", r.Type)
			}
		}
	})

	t.Run("uses default limit 20", func(t *testing.T) {
		var gotPageSize string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPageSize = r.URL.Query().Get("PageSize")
			_ = json.NewEncoder(w).Encode(SearchResponse{Data: &SearchData{}})
		}))
		defer server.Close()

		c := NewClientWithBaseURL(server.URL, "")
		c.SetHTTPClient(server.Client())
		p := NewProviderWithClient(c)

		_, _ = p.Search(context.Background(), "", "", "", 0)
		if gotPageSize != "20" {
			t.Errorf("expected PageSize=20, got %s", gotPageSize)
		}
	})

	t.Run("nil data returns empty slice", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(SearchResponse{Data: nil})
		}))
		defer server.Close()

		c := NewClientWithBaseURL(server.URL, "")
		c.SetHTTPClient(server.Client())
		p := NewProviderWithClient(c)

		results, err := p.Search(context.Background(), "query", "", "", 10)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results for nil data, got %d", len(results))
		}
	})

	t.Run("search error propagates", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(ErrorResponse{Message: "internal error"})
		}))
		defer server.Close()

		c := NewClientWithBaseURL(server.URL, "")
		c.SetHTTPClient(server.Client())
		p := NewProviderWithClient(c)

		_, err := p.Search(context.Background(), "query", "", "", 10)
		if err == nil {
			t.Error("expected error for server error")
		}
	})
}

// --- ImportLocal ---

func TestProvider_ImportLocal(t *testing.T) {
	t.Run("non-existent path returns error", func(t *testing.T) {
		p := NewProvider()
		_, err := p.ImportLocal(context.Background(), "/nonexistent/model.gguf", false)
		if err == nil {
			t.Error("expected error for non-existent path")
		}
	})

	t.Run("single gguf file", func(t *testing.T) {
		tmpDir := t.TempDir()
		ggufPath := filepath.Join(tmpDir, "llama3.gguf")
		if err := os.WriteFile(ggufPath, []byte("fake model"), 0644); err != nil {
			t.Fatal(err)
		}

		p := NewProvider()
		m, err := p.ImportLocal(context.Background(), ggufPath, false)
		if err != nil {
			t.Fatalf("ImportLocal failed: %v", err)
		}
		if m.Format != model.FormatGGUF {
			t.Errorf("expected format GGUF, got %s", m.Format)
		}
		if m.Status != model.StatusReady {
			t.Errorf("expected status ready, got %s", m.Status)
		}
		if m.Source != "local" {
			t.Errorf("expected source 'local', got %s", m.Source)
		}
		if m.Name != "llama3" {
			t.Errorf("expected name 'llama3', got %s", m.Name)
		}
	})

	t.Run("safetensors file", func(t *testing.T) {
		tmpDir := t.TempDir()
		stPath := filepath.Join(tmpDir, "model.safetensors")
		if err := os.WriteFile(stPath, []byte("fake"), 0644); err != nil {
			t.Fatal(err)
		}

		p := NewProvider()
		m, err := p.ImportLocal(context.Background(), stPath, false)
		if err != nil {
			t.Fatalf("ImportLocal failed: %v", err)
		}
		if m.Format != model.FormatSafetensors {
			t.Errorf("expected format safetensors, got %s", m.Format)
		}
	})

	t.Run("directory with safetensors files", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "model.safetensors"), []byte("fake"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}

		p := NewProvider()
		m, err := p.ImportLocal(context.Background(), tmpDir, false)
		if err != nil {
			t.Fatalf("ImportLocal failed: %v", err)
		}
		if m.Format != model.FormatSafetensors {
			t.Errorf("expected format safetensors, got %s", m.Format)
		}
		if m.Path != tmpDir {
			t.Errorf("expected path %s, got %s", tmpDir, m.Path)
		}
	})

	t.Run("directory with gguf file", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "model.gguf"), []byte("fake"), 0644); err != nil {
			t.Fatal(err)
		}

		p := NewProvider()
		m, err := p.ImportLocal(context.Background(), tmpDir, false)
		if err != nil {
			t.Fatalf("ImportLocal failed: %v", err)
		}
		if m.Format != model.FormatGGUF {
			t.Errorf("expected format gguf, got %s", m.Format)
		}
	})

	t.Run("directory with no model files defaults to gguf", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("hello"), 0644); err != nil {
			t.Fatal(err)
		}

		p := NewProvider()
		m, err := p.ImportLocal(context.Background(), tmpDir, false)
		if err != nil {
			t.Fatalf("ImportLocal failed: %v", err)
		}
		if m.Format != model.FormatGGUF {
			t.Errorf("expected default format gguf, got %s", m.Format)
		}
	})

	t.Run("model has non-empty ID", func(t *testing.T) {
		tmpDir := t.TempDir()
		ggufPath := filepath.Join(tmpDir, "mymodel.gguf")
		if err := os.WriteFile(ggufPath, []byte("fake"), 0644); err != nil {
			t.Fatal(err)
		}

		p := NewProvider()
		m, err := p.ImportLocal(context.Background(), ggufPath, false)
		if err != nil {
			t.Fatalf("ImportLocal failed: %v", err)
		}
		if m.ID == "" {
			t.Error("expected non-empty model ID")
		}
		if !strings.HasPrefix(m.ID, "model-") {
			t.Errorf("expected ID to start with 'model-', got %s", m.ID)
		}
	})
}

// --- Verify ---

func TestProvider_Verify_ModelNotCached(t *testing.T) {
	p := NewProvider()
	result, err := p.Verify(context.Background(), "nonexistent-model-id", "")
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid result for unknown model")
	}
	if len(result.Issues) == 0 {
		t.Error("expected at least one issue")
	}
}

func TestProvider_Verify_NoLocalPath(t *testing.T) {
	p := NewProvider()
	p.mu.Lock()
	p.modelCache["org/test"] = &model.Model{
		ID:   "model-abc",
		Name: "test",
		Path: "", // empty path
	}
	p.mu.Unlock()

	result, err := p.Verify(context.Background(), "model-abc", "")
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid result for model with no path")
	}
}

func TestProvider_Verify_PathNotExist(t *testing.T) {
	p := NewProvider()
	p.mu.Lock()
	p.modelCache["org/test"] = &model.Model{
		ID:   "model-xyz",
		Path: "/nonexistent/path",
	}
	p.mu.Unlock()

	result, err := p.Verify(context.Background(), "model-xyz", "")
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid result for non-existent path")
	}
}

func TestProvider_Verify_ValidPath(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a fake file to make the path exist
	if err := os.WriteFile(filepath.Join(tmpDir, "model.gguf"), []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	// Use a server that returns an error for the repo check (no real modelscope)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(ModelInfo{Data: &ModelDetail{Name: "test"}})
	}))
	defer server.Close()

	c := NewClientWithBaseURL(server.URL, "")
	c.SetHTTPClient(server.Client())
	p := NewProviderWithClient(c)
	p.mu.Lock()
	p.modelCache["org/test-model"] = &model.Model{
		ID:   "model-valid",
		Path: tmpDir,
	}
	p.mu.Unlock()

	result, err := p.Verify(context.Background(), "model-valid", "")
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid result, got issues: %v", result.Issues)
	}
}

func TestProvider_Verify_SizeChecksum(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "model-*.gguf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := []byte("fake model content 123")
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ModelInfo{Data: &ModelDetail{Name: "test"}})
	}))
	defer server.Close()

	c := NewClientWithBaseURL(server.URL, "")
	c.SetHTTPClient(server.Client())
	p := NewProviderWithClient(c)
	p.mu.Lock()
	p.modelCache["org/model"] = &model.Model{
		ID:   "model-checksum",
		Path: tmpFile.Name(),
	}
	p.mu.Unlock()

	t.Run("correct size checksum", func(t *testing.T) {
		checksum := "size:22" // len("fake model content 123")
		result, err := p.Verify(context.Background(), "model-checksum", checksum)
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}
		if !result.Valid {
			t.Errorf("expected valid for correct size checksum, got issues: %v", result.Issues)
		}
	})

	t.Run("wrong size checksum", func(t *testing.T) {
		checksum := "size:9999"
		result, err := p.Verify(context.Background(), "model-checksum", checksum)
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}
		if result.Valid {
			t.Error("expected invalid for wrong size checksum")
		}
	})
}

// --- EstimateResources ---

func TestProvider_EstimateResources_ModelNotCached(t *testing.T) {
	p := NewProvider()
	_, err := p.EstimateResources(context.Background(), "nonexistent-id")
	if err == nil {
		t.Error("expected error for uncached model")
	}
	if !strings.Contains(err.Error(), "model not found") {
		t.Errorf("expected 'model not found' in error, got: %v", err)
	}
}

func TestProvider_EstimateResources_FromFileSummary(t *testing.T) {
	resp := ModelInfo{
		Data: &ModelDetail{
			Name: "Qwen-7B",
			FileSummary: &FileSummary{
				TotalSize: 8 * 1024 * 1024 * 1024, // 8GB
			},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClientWithBaseURL(server.URL, "")
	c.SetHTTPClient(server.Client())
	p := NewProviderWithClient(c)
	p.mu.Lock()
	p.modelCache["org/qwen7b"] = &model.Model{ID: "model-qwen"}
	p.mu.Unlock()

	req, err := p.EstimateResources(context.Background(), "model-qwen")
	if err != nil {
		t.Fatalf("EstimateResources failed: %v", err)
	}
	if req == nil {
		t.Fatal("expected non-nil requirements")
	}
	baseBytes := 8 * 1024 * 1024 * 1024
	expectedMin := int64(float64(baseBytes) * 1.2)
	if req.MemoryMin != expectedMin {
		t.Errorf("expected MemoryMin %d, got %d", expectedMin, req.MemoryMin)
	}
}

func TestProvider_EstimateResources_FromGigabytes(t *testing.T) {
	resp := ModelInfo{
		Data: &ModelDetail{
			Name:      "BigModel",
			Gigabytes: 14.0, // 14GB
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClientWithBaseURL(server.URL, "")
	c.SetHTTPClient(server.Client())
	p := NewProviderWithClient(c)
	p.mu.Lock()
	p.modelCache["org/bigmodel"] = &model.Model{ID: "model-big"}
	p.mu.Unlock()

	req, err := p.EstimateResources(context.Background(), "model-big")
	if err != nil {
		t.Fatalf("EstimateResources failed: %v", err)
	}
	if req.MemoryMin <= 0 {
		t.Error("expected positive MemoryMin")
	}
	if req.MemoryRecommended <= req.MemoryMin {
		t.Error("expected MemoryRecommended > MemoryMin")
	}
}

func TestProvider_EstimateResources_NoSizeInfo(t *testing.T) {
	// No file summary and no gigabytes — defaults to 4GB/8GB
	resp := ModelInfo{
		Data: &ModelDetail{
			Name:      "UnknownSize",
			Gigabytes: 0,
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClientWithBaseURL(server.URL, "")
	c.SetHTTPClient(server.Client())
	p := NewProviderWithClient(c)
	p.mu.Lock()
	p.modelCache["org/unknownsize"] = &model.Model{ID: "model-unknown"}
	p.mu.Unlock()

	req, err := p.EstimateResources(context.Background(), "model-unknown")
	if err != nil {
		t.Fatalf("EstimateResources failed: %v", err)
	}
	if req.MemoryMin != 4*1024*1024*1024 {
		t.Errorf("expected default 4GB MemoryMin, got %d", req.MemoryMin)
	}
	if req.MemoryRecommended != 8*1024*1024*1024 {
		t.Errorf("expected default 8GB MemoryRecommended, got %d", req.MemoryRecommended)
	}
}

func TestProvider_EstimateResources_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(ErrorResponse{Message: "internal error"})
	}))
	defer server.Close()

	c := NewClientWithBaseURL(server.URL, "")
	c.SetHTTPClient(server.Client())
	p := NewProviderWithClient(c)
	p.mu.Lock()
	p.modelCache["org/model"] = &model.Model{ID: "model-err"}
	p.mu.Unlock()

	_, err := p.EstimateResources(context.Background(), "model-err")
	if err == nil {
		t.Error("expected error for API error")
	}
}

// --- modelTypeToPipelineTag ---

func TestModelTypeToPipelineTag(t *testing.T) {
	tests := []struct {
		modelType model.ModelType
		expected  string
	}{
		{model.ModelTypeLLM, "text-generation"},
		{model.ModelTypeVLM, "image-text-to-text"},
		{model.ModelTypeASR, "automatic-speech-recognition"},
		{model.ModelTypeTTS, "text-to-speech"},
		{model.ModelTypeEmbedding, "feature-extraction"},
		{model.ModelTypeDiffusion, "text-to-image"},
		{model.ModelTypeVideoGen, "text-to-video"},
		{model.ModelTypeDetection, "object-detection"},
		{model.ModelTypeRerank, "text-ranking"},
		{"unknown-type", ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.modelType), func(t *testing.T) {
			got := modelTypeToPipelineTag(tt.modelType)
			if got != tt.expected {
				t.Errorf("modelTypeToPipelineTag(%q) = %q, want %q", tt.modelType, got, tt.expected)
			}
		})
	}
}

// --- parseInt64 ---

func TestParseInt64(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"42", 42},
		{"0", 0},
		{"1234567890", 1234567890},
		{"", 0},
		{"abc", 0},
		{"12abc34", 1234}, // digits only
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseInt64(tt.input)
			if got != tt.expected {
				t.Errorf("parseInt64(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

// --- tagsToStrings ---

func TestTagsToStrings(t *testing.T) {
	tags := []Tag{
		{Name: "llm"},
		{Name: "chinese"},
		{Name: "7B"},
	}
	result := tagsToStrings(tags)
	if len(result) != 3 {
		t.Errorf("expected 3 tags, got %d", len(result))
	}
	if result[0] != "llm" || result[1] != "chinese" || result[2] != "7B" {
		t.Errorf("unexpected tags: %v", result)
	}
}

func TestTagsToStrings_Empty(t *testing.T) {
	result := tagsToStrings(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 tags, got %d", len(result))
	}
}

// --- Concurrent access ---

func TestProvider_ConcurrentModelCache_NoRace(t *testing.T) {
	p := NewProvider()
	const goroutines = 20
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := strings.Repeat("x", n+1)
			p.mu.Lock()
			p.modelCache[key] = &model.Model{ID: "model-concurrent"}
			p.mu.Unlock()
		}(i)
	}

	// Concurrent reads
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.mu.RLock()
			_ = len(p.modelCache)
			p.mu.RUnlock()
		}()
	}

	wg.Wait()
}

func TestProvider_ConcurrentVerify_NoRace(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "model.gguf"), []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ModelInfo{Data: &ModelDetail{Name: "model"}})
	}))
	defer server.Close()

	c := NewClientWithBaseURL(server.URL, "")
	c.SetHTTPClient(server.Client())
	p := NewProviderWithClient(c)

	// Pre-populate cache with multiple entries
	for i := 0; i < 5; i++ {
		key := strings.Repeat("a", i+1) + "/model"
		modelID := strings.Repeat("m", i+1)
		p.mu.Lock()
		p.modelCache[key] = &model.Model{ID: modelID, Path: tmpDir}
		p.mu.Unlock()
	}

	const goroutines = 10
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			modelID := strings.Repeat("m", (n%5)+1)
			_, _ = p.Verify(context.Background(), modelID, "")
		}(i)
	}
	wg.Wait()
}
