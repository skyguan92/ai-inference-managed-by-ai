package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/engine"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/inference"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
)

func TestProvider_New(t *testing.T) {
	t.Run("with default URL", func(t *testing.T) {
		p := NewProvider("")
		if p == nil {
			t.Fatal("expected provider, got nil")
		}
		if p.client == nil {
			t.Fatal("expected client, got nil")
		}
	})

	t.Run("with custom URL", func(t *testing.T) {
		p := NewProvider("http://custom:8080")
		if p == nil {
			t.Fatal("expected provider, got nil")
		}
		if p.client.baseURL != "http://custom:8080" {
			t.Errorf("expected baseURL http://custom:8080, got %s", p.client.baseURL)
		}
	})

	t.Run("with custom client", func(t *testing.T) {
		client := NewClient("http://test:1234")
		p := NewProviderWithClient(client)
		if p == nil {
			t.Fatal("expected provider, got nil")
		}
		if p.client.baseURL != "http://test:1234" {
			t.Errorf("expected baseURL http://test:1234, got %s", p.client.baseURL)
		}
	})
}

func TestProvider_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("expected path /api/chat, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		resp := ChatResponse{
			Model: "llama3",
			Message: &ChatMessage{
				Role:    "assistant",
				Content: "Hello, I am an AI assistant.",
			},
			Done:            true,
			PromptEvalCount: 10,
			EvalCount:       20,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetHTTPClient(server.Client())
	p := NewProviderWithClient(client)

	messages := []inference.Message{
		{Role: "user", Content: "Hello"},
	}
	opts := inference.ChatOptions{}

	resp, err := p.Chat(context.Background(), "llama3", messages, opts)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if resp.Content != "Hello, I am an AI assistant." {
		t.Errorf("expected content 'Hello, I am an AI assistant.', got %s", resp.Content)
	}
	if resp.Model != "llama3" {
		t.Errorf("expected model 'llama3', got %s", resp.Model)
	}
	if resp.Usage.PromptTokens != 10 {
		t.Errorf("expected prompt tokens 10, got %d", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 20 {
		t.Errorf("expected completion tokens 20, got %d", resp.Usage.CompletionTokens)
	}
}

func TestProvider_ChatWithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		if req.Options["temperature"] != 0.7 {
			t.Errorf("expected temperature 0.7, got %v", req.Options["temperature"])
		}
		if req.Options["num_predict"] != float64(100) {
			t.Errorf("expected num_predict 100, got %v", req.Options["num_predict"])
		}

		resp := ChatResponse{
			Model:     "llama3",
			Message:   &ChatMessage{Role: "assistant", Content: "OK"},
			Done:      true,
			EvalCount: 5,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetHTTPClient(server.Client())
	p := NewProviderWithClient(client)

	temp := 0.7
	maxTokens := 100
	opts := inference.ChatOptions{
		Temperature: &temp,
		MaxTokens:   &maxTokens,
	}

	_, err := p.Chat(context.Background(), "llama3", []inference.Message{{Role: "user", Content: "Hi"}}, opts)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
}

func TestProvider_Complete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("expected path /api/generate, got %s", r.URL.Path)
		}

		resp := GenerateResponse{
			Model:           "llama3",
			Response:        "This is a completion.",
			Done:            true,
			PromptEvalCount: 5,
			EvalCount:       10,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetHTTPClient(server.Client())
	p := NewProviderWithClient(client)

	opts := inference.CompleteOptions{}
	resp, err := p.Complete(context.Background(), "llama3", "Complete this:", opts)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if resp.Text != "This is a completion." {
		t.Errorf("expected text 'This is a completion.', got %s", resp.Text)
	}
	if resp.Usage.PromptTokens != 5 {
		t.Errorf("expected prompt tokens 5, got %d", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 10 {
		t.Errorf("expected completion tokens 10, got %d", resp.Usage.CompletionTokens)
	}
}

func TestProvider_Embed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			t.Errorf("expected path /api/embeddings, got %s", r.URL.Path)
		}

		resp := EmbeddingResponse{
			Embedding: []float64{0.1, 0.2, 0.3, 0.4, 0.5},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetHTTPClient(server.Client())
	p := NewProviderWithClient(client)

	resp, err := p.Embed(context.Background(), "nomic-embed-text", []string{"hello", "world"})
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(resp.Embeddings) != 2 {
		t.Errorf("expected 2 embeddings, got %d", len(resp.Embeddings))
	}
	if len(resp.Embeddings[0]) != 5 {
		t.Errorf("expected embedding length 5, got %d", len(resp.Embeddings[0]))
	}
}

func TestProvider_ListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("expected path /api/tags, got %s", r.URL.Path)
		}

		resp := ListModelsResponse{
			Models: []ModelInfo{
				{Name: "llama3:latest", Size: 4000000000},
				{Name: "nomic-embed-text:latest", Size: 274000000},
				{Name: "llava:latest", Size: 4500000000},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetHTTPClient(server.Client())
	p := NewProviderWithClient(client)

	models, err := p.ListModels(context.Background(), "")
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	if len(models) != 3 {
		t.Errorf("expected 3 models, got %d", len(models))
	}

	foundLLM := false
	foundEmbedding := false
	foundVLM := false
	for _, m := range models {
		if m.ID == "llama3:latest" && m.Type == "llm" {
			foundLLM = true
		}
		if m.ID == "nomic-embed-text:latest" && m.Type == "embedding" {
			foundEmbedding = true
		}
		if m.ID == "llava:latest" && m.Type == "vlm" {
			foundVLM = true
		}
	}

	if !foundLLM {
		t.Error("expected to find LLM model")
	}
	if !foundEmbedding {
		t.Error("expected to find embedding model")
	}
	if !foundVLM {
		t.Error("expected to find VLM model")
	}
}

func TestProvider_ListModelsFiltered(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ListModelsResponse{
			Models: []ModelInfo{
				{Name: "llama3:latest"},
				{Name: "nomic-embed-text:latest"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetHTTPClient(server.Client())
	p := NewProviderWithClient(client)

	models, err := p.ListModels(context.Background(), "embedding")
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	embeddingCount := 0
	for _, m := range models {
		if m.Type == "embedding" && m.ID != "" {
			embeddingCount++
		}
	}
	if embeddingCount != 1 {
		t.Errorf("expected 1 embedding model, got %d", embeddingCount)
	}
}

func TestProvider_Pull(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/pull" {
			t.Errorf("expected path /api/pull, got %s", r.URL.Path)
		}

		responses := []PullResponse{
			{Status: "pulling manifest", Total: 0, Completed: 0},
			{Status: "downloading", Total: 100, Completed: 50},
			{Status: "success", Total: 100, Completed: 100, Digest: "sha256:abc123"},
		}

		w.Header().Set("Content-Type", "application/x-ndjson")
		for _, resp := range responses {
			data, _ := json.Marshal(resp)
			_, _ = w.Write(data)
			_, _ = w.Write([]byte("\n"))
		}
		callCount++
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetHTTPClient(server.Client())
	p := NewProviderWithClient(client)

	progressCh := make(chan model.PullProgress, 10)
	m, err := p.Pull(context.Background(), "", "llama3", "latest", progressCh)
	close(progressCh)

	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	if m.Name != "llama3:latest" {
		t.Errorf("expected model name 'llama3:latest', got %s", m.Name)
	}
	if m.Status != model.StatusReady {
		t.Errorf("expected status ready, got %s", m.Status)
	}
	if m.Source != "ollama" {
		t.Errorf("expected source ollama, got %s", m.Source)
	}

	var progressCount int
	for range progressCh {
		progressCount++
	}
	if progressCount != 3 {
		t.Errorf("expected 3 progress updates, got %d", progressCount)
	}
}

func TestProvider_PullWithoutProgress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := PullResponse{Status: "success", Digest: "sha256:abc123"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetHTTPClient(server.Client())
	p := NewProviderWithClient(client)

	m, err := p.Pull(context.Background(), "", "llama3", "", nil)
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	if m.Status != model.StatusReady {
		t.Errorf("expected status ready, got %s", m.Status)
	}
}

func TestProvider_PullUnsupportedSource(t *testing.T) {
	p := NewProvider("")
	_, err := p.Pull(context.Background(), "huggingface", "model", "latest", nil)
	if err == nil {
		t.Error("expected error for unsupported source")
	}
}

func TestProvider_Search(t *testing.T) {
	p := NewProvider("")

	t.Run("search by query", func(t *testing.T) {
		results, err := p.Search(context.Background(), "llama", "", "", 0)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(results) == 0 {
			t.Error("expected results for 'llama' query")
		}

		for _, r := range results {
			if r.Source != "ollama" {
				t.Errorf("expected source ollama, got %s", r.Source)
			}
		}
	})

	t.Run("search by type", func(t *testing.T) {
		results, err := p.Search(context.Background(), "", "", model.ModelTypeEmbedding, 0)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		for _, r := range results {
			if r.Type != model.ModelTypeEmbedding {
				t.Errorf("expected embedding type, got %s", r.Type)
			}
		}
	})

	t.Run("search with limit", func(t *testing.T) {
		results, err := p.Search(context.Background(), "", "", "", 3)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(results) > 3 {
			t.Errorf("expected at most 3 results, got %d", len(results))
		}
	})
}

func TestProvider_GetFeatures(t *testing.T) {
	p := NewProvider("")

	features, err := p.GetFeatures(context.Background(), "ollama")
	if err != nil {
		t.Fatalf("GetFeatures failed: %v", err)
	}

	if !features.SupportsStreaming {
		t.Error("expected streaming support")
	}
	if !features.SupportsMultimodal {
		t.Error("expected multimodal support")
	}
	if !features.SupportsTools {
		t.Error("expected tools support")
	}
	if !features.SupportsEmbedding {
		t.Error("expected embedding support")
	}
	if features.MaxConcurrent != 10 {
		t.Errorf("expected max concurrent 10, got %d", features.MaxConcurrent)
	}
}

func TestProvider_UnsupportedOperations(t *testing.T) {
	p := NewProvider("")
	ctx := context.Background()

	t.Run("Transcribe", func(t *testing.T) {
		_, err := p.Transcribe(ctx, "whisper", []byte{}, "en")
		if err == nil {
			t.Error("expected error for transcribe")
		}
	})

	t.Run("Synthesize", func(t *testing.T) {
		_, err := p.Synthesize(ctx, "tts", "hello", "default")
		if err == nil {
			t.Error("expected error for synthesize")
		}
	})

	t.Run("GenerateImage", func(t *testing.T) {
		_, err := p.GenerateImage(ctx, "diffusion", "prompt", inference.ImageOptions{})
		if err == nil {
			t.Error("expected error for generate image")
		}
	})

	t.Run("GenerateVideo", func(t *testing.T) {
		_, err := p.GenerateVideo(ctx, "video", "prompt", inference.VideoOptions{})
		if err == nil {
			t.Error("expected error for generate video")
		}
	})

	t.Run("Rerank", func(t *testing.T) {
		_, err := p.Rerank(ctx, "rerank", "query", []string{"doc"})
		if err == nil {
			t.Error("expected error for rerank")
		}
	})

	t.Run("Detect", func(t *testing.T) {
		_, err := p.Detect(ctx, "detection", []byte{})
		if err == nil {
			t.Error("expected error for detect")
		}
	})

	t.Run("ListVoices", func(t *testing.T) {
		_, err := p.ListVoices(ctx, "tts")
		if err == nil {
			t.Error("expected error for list voices")
		}
	})
}

func TestProvider_Client(t *testing.T) {
	client := NewClient("http://test:1234")
	p := NewProviderWithClient(client)

	if p.Client() != client {
		t.Error("expected same client instance")
	}
}

func TestClient_ShowModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/show" {
			t.Errorf("expected path /api/show, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		resp := ShowModelResponse{
			License:   "MIT",
			Modelfile: "FROM llama3",
			Details: struct {
				Format            string `json:"format"`
				Family            string `json:"family"`
				ParameterSize     string `json:"parameter_size"`
				QuantizationLevel string `json:"quantization_level"`
			}{
				Format:            "gguf",
				Family:            "llama",
				ParameterSize:     "8B",
				QuantizationLevel: "Q4_0",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetHTTPClient(server.Client())

	resp, err := client.ShowModel(context.Background(), &ShowModelRequest{Name: "llama3"})
	if err != nil {
		t.Fatalf("ShowModel failed: %v", err)
	}

	if resp.License != "MIT" {
		t.Errorf("expected license MIT, got %s", resp.License)
	}
	if resp.Details.ParameterSize != "8B" {
		t.Errorf("expected parameter size 8B, got %s", resp.Details.ParameterSize)
	}
}

func TestClient_DeleteModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/delete" {
			t.Errorf("expected path /api/delete, got %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetHTTPClient(server.Client())

	err := client.DeleteModel(context.Background(), &DeleteModelRequest{Name: "llama3"})
	if err != nil {
		t.Fatalf("DeleteModel failed: %v", err)
	}
}

func TestClient_IsRunning(t *testing.T) {
	t.Run("running", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		client.SetHTTPClient(server.Client())

		if !client.IsRunning(context.Background()) {
			t.Error("expected IsRunning to return true")
		}
	})

	t.Run("not running", func(t *testing.T) {
		client := NewClient("http://localhost:59999")
		if client.IsRunning(context.Background()) {
			t.Error("expected IsRunning to return false for non-existent server")
		}
	})
}

func TestClient_ErrorHandling(t *testing.T) {
	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "internal server error"}`))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		client.SetHTTPClient(server.Client())

		_, err := client.Generate(context.Background(), &GenerateRequest{Model: "test"})
		if err == nil {
			t.Error("expected error for server error")
		}
		if !strings.Contains(err.Error(), "internal server error") {
			t.Errorf("expected error to contain 'internal server error', got %v", err)
		}
	})

	t.Run("server error without message", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{}`))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		client.SetHTTPClient(server.Client())

		_, err := client.Generate(context.Background(), &GenerateRequest{Model: "test"})
		if err == nil {
			t.Error("expected error for server error")
		}
	})

	t.Run("invalid json response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`invalid json`))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		client.SetHTTPClient(server.Client())

		_, err := client.Generate(context.Background(), &GenerateRequest{Model: "test"})
		if err == nil {
			t.Error("expected error for invalid json")
		}
	})
}

func TestClient_StreamingError(t *testing.T) {
	t.Run("server error in streaming", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": "bad request"}`))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		client.SetHTTPClient(server.Client())

		err := client.Pull(context.Background(), &PullRequest{Name: "test"}, func(resp *PullResponse) error {
			return nil
		})
		if err == nil {
			t.Error("expected error for streaming server error")
		}
	})

	t.Run("handler error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/x-ndjson")
			_, _ = w.Write([]byte(`{"status":"pulling"}` + "\n"))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		client.SetHTTPClient(server.Client())

		expectedErr := errors.New("handler error")
		err := client.Pull(context.Background(), &PullRequest{Name: "test"}, func(resp *PullResponse) error {
			return expectedErr
		})
		if err != expectedErr {
			t.Errorf("expected handler error, got %v", err)
		}
	})

	t.Run("invalid json in stream", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/x-ndjson")
			_, _ = w.Write([]byte(`invalid json` + "\n"))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		client.SetHTTPClient(server.Client())

		err := client.Pull(context.Background(), &PullRequest{Name: "test"}, func(resp *PullResponse) error {
			return nil
		})
		if err == nil {
			t.Error("expected error for invalid json in stream")
		}
	})
}

func TestProvider_CompleteWithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req GenerateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		if req.Options["temperature"] != 0.5 {
			t.Errorf("expected temperature 0.5, got %v", req.Options["temperature"])
		}
		if req.Options["num_predict"] != float64(50) {
			t.Errorf("expected num_predict 50, got %v", req.Options["num_predict"])
		}
		if req.Options["top_p"] != 0.9 {
			t.Errorf("expected top_p 0.9, got %v", req.Options["top_p"])
		}

		resp := GenerateResponse{
			Model:           "llama3",
			Response:        "completion",
			Done:            true,
			PromptEvalCount: 5,
			EvalCount:       10,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetHTTPClient(server.Client())
	p := NewProviderWithClient(client)

	temp := 0.5
	maxTokens := 50
	topP := 0.9
	opts := inference.CompleteOptions{
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		TopP:        &topP,
		Stop:        []string{"END"},
	}

	resp, err := p.Complete(context.Background(), "llama3", "prompt", opts)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if resp.Text != "completion" {
		t.Errorf("expected text 'completion', got %s", resp.Text)
	}
}

func TestProvider_ChatWithAllOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		if req.Options["top_k"] != float64(40) {
			t.Errorf("expected top_k 40, got %v", req.Options["top_k"])
		}
		if req.Options["frequency_penalty"] != float64(0.5) {
			t.Errorf("expected frequency_penalty 0.5, got %v", req.Options["frequency_penalty"])
		}
		if req.Options["presence_penalty"] != float64(0.3) {
			t.Errorf("expected presence_penalty 0.3, got %v", req.Options["presence_penalty"])
		}

		resp := ChatResponse{
			Model:   "llama3",
			Message: &ChatMessage{Role: "assistant", Content: "response"},
			Done:    true,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetHTTPClient(server.Client())
	p := NewProviderWithClient(client)

	topK := 40
	freqPenalty := 0.5
	presPenalty := 0.3
	opts := inference.ChatOptions{
		TopK:             &topK,
		FrequencyPenalty: &freqPenalty,
		PresencePenalty:  &presPenalty,
		Stop:             []string{"END"},
	}

	_, err := p.Chat(context.Background(), "llama3", []inference.Message{{Role: "user", Content: "hi"}}, opts)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
}

func TestProvider_ChatError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "model not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetHTTPClient(server.Client())
	p := NewProviderWithClient(client)

	_, err := p.Chat(context.Background(), "nonexistent", []inference.Message{}, inference.ChatOptions{})
	if err == nil {
		t.Error("expected error for chat failure")
	}
}

func TestProvider_CompleteError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "model not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetHTTPClient(server.Client())
	p := NewProviderWithClient(client)

	_, err := p.Complete(context.Background(), "nonexistent", "prompt", inference.CompleteOptions{})
	if err == nil {
		t.Error("expected error for complete failure")
	}
}

func TestProvider_EmbedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "model not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetHTTPClient(server.Client())
	p := NewProviderWithClient(client)

	_, err := p.Embed(context.Background(), "nonexistent", []string{"test"})
	if err == nil {
		t.Error("expected error for embed failure")
	}
}

func TestProvider_ListModelsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetHTTPClient(server.Client())
	p := NewProviderWithClient(client)

	_, err := p.ListModels(context.Background(), "")
	if err == nil {
		t.Error("expected error for list models failure")
	}
}

func TestProvider_Verify(t *testing.T) {
	t.Run("model exists", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := ShowModelResponse{
				Details: struct {
					Format            string `json:"format"`
					Family            string `json:"family"`
					ParameterSize     string `json:"parameter_size"`
					QuantizationLevel string `json:"quantization_level"`
				}{
					Format:        "gguf",
					ParameterSize: "8B",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		client.SetHTTPClient(server.Client())
		p := NewProviderWithClient(client)

		result, err := p.Verify(context.Background(), "llama3", "")
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}
		if !result.Valid {
			t.Error("expected valid result")
		}
	})

	t.Run("model not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "model not found"}`))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		client.SetHTTPClient(server.Client())
		p := NewProviderWithClient(client)

		result, err := p.Verify(context.Background(), "nonexistent", "")
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}
		if result.Valid {
			t.Error("expected invalid result for non-existent model")
		}
	})

	t.Run("with cached model ID", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req ShowModelRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.Name != "llama3:latest" {
				t.Errorf("expected model name 'llama3:latest', got %s", req.Name)
			}
			resp := ShowModelResponse{}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		client.SetHTTPClient(server.Client())
		p := NewProviderWithClient(client)

		p.mu.Lock()
		p.modelCache["llama3:latest"] = &model.Model{ID: "model-abc123", Name: "llama3:latest"}
		p.mu.Unlock()

		result, err := p.Verify(context.Background(), "model-abc123", "")
		if err != nil {
			t.Fatalf("Verify failed: %v", err)
		}
		if !result.Valid {
			t.Error("expected valid result")
		}
	})
}

func TestProvider_EstimateResources(t *testing.T) {
	tests := []struct {
		name         string
		paramSize    string
		quantization string
		wantMemMin   int64
	}{
		{"70B model", "70B", "Q4_0", 40 * 1024 * 1024 * 1024 / 3},
		{"30B model", "30B", "Q4_0", 20 * 1024 * 1024 * 1024 / 3},
		{"13B model", "13B", "Q4_0", 8 * 1024 * 1024 * 1024 / 3},
		{"7B model", "7B", "Q4_0", 4 * 1024 * 1024 * 1024 / 3},
		{"small model", "3B", "Q4_0", 2 * 1024 * 1024 * 1024 / 3},
		{"no quantization", "7B", "F16", 4 * 1024 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				resp := ShowModelResponse{
					Details: struct {
						Format            string `json:"format"`
						Family            string `json:"family"`
						ParameterSize     string `json:"parameter_size"`
						QuantizationLevel string `json:"quantization_level"`
					}{
						ParameterSize:     tt.paramSize,
						QuantizationLevel: tt.quantization,
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			client := NewClient(server.URL)
			client.SetHTTPClient(server.Client())
			p := NewProviderWithClient(client)

			req, err := p.EstimateResources(context.Background(), "llama3")
			if err != nil {
				t.Fatalf("EstimateResources failed: %v", err)
			}
			if req.MemoryMin != tt.wantMemMin {
				t.Errorf("expected memory min %d, got %d", tt.wantMemMin, req.MemoryMin)
			}
		})
	}

	t.Run("with cached model ID", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req ShowModelRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if req.Name != "llama3:latest" {
				t.Errorf("expected model name 'llama3:latest', got %s", req.Name)
			}
			resp := ShowModelResponse{
				Details: struct {
					Format            string `json:"format"`
					Family            string `json:"family"`
					ParameterSize     string `json:"parameter_size"`
					QuantizationLevel string `json:"quantization_level"`
				}{
					ParameterSize: "7B",
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		client.SetHTTPClient(server.Client())
		p := NewProviderWithClient(client)

		p.mu.Lock()
		p.modelCache["llama3:latest"] = &model.Model{ID: "model-abc123", Name: "llama3:latest"}
		p.mu.Unlock()

		_, err := p.EstimateResources(context.Background(), "model-abc123")
		if err != nil {
			t.Fatalf("EstimateResources failed: %v", err)
		}
	})

	t.Run("error from API", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		client.SetHTTPClient(server.Client())
		p := NewProviderWithClient(client)

		_, err := p.EstimateResources(context.Background(), "llama3")
		if err == nil {
			t.Error("expected error for API failure")
		}
	})
}

func TestParseParameterSize(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"", 0},
		{"7B", 7},
		{"13B", 13},
		{"70B", 70},
		{"7b", 7},
		{"7", 7},
		{"invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseParameterSize(tt.input)
			if result != tt.expected {
				t.Errorf("parseParameterSize(%s) = %f, expected %f", tt.input, result, tt.expected)
			}
		})
	}
}

func TestProvider_PullWithModelName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := PullResponse{Status: "success", Digest: "sha256:abc"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetHTTPClient(server.Client())
	p := NewProviderWithClient(client)

	m, err := p.Pull(context.Background(), "", "llama3", "v2", nil)
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}
	if m.Name != "llama3:v2" {
		t.Errorf("expected name 'llama3:v2', got %s", m.Name)
	}
}

func TestProvider_PullError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "pull failed"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetHTTPClient(server.Client())
	p := NewProviderWithClient(client)

	t.Run("without progress", func(t *testing.T) {
		_, err := p.Pull(context.Background(), "", "llama3", "", nil)
		if err == nil {
			t.Error("expected error for pull failure")
		}
	})

	t.Run("with progress", func(t *testing.T) {
		progressCh := make(chan model.PullProgress, 10)
		_, err := p.Pull(context.Background(), "", "llama3", "", progressCh)
		close(progressCh)
		if err == nil {
			t.Error("expected error for pull failure")
		}
	})
}

func TestProvider_Install(t *testing.T) {
	p := NewProvider("")
	result, err := p.Install(context.Background(), "ollama", "")
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.Path == "" {
		t.Error("expected non-empty path")
	}
}

func TestProvider_Stop(t *testing.T) {
	t.Run("no process", func(t *testing.T) {
		p := NewProvider("")
		result, err := p.Stop(context.Background(), "ollama", false, 5)
		if err != nil {
			t.Fatalf("Stop failed: %v", err)
		}
		if !result.Success {
			t.Error("expected success")
		}
	})
}

func TestProvider_Start(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetHTTPClient(server.Client())
	p := NewProviderWithClient(client)

	result, err := p.Start(context.Background(), "ollama", nil)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if result.Status != engine.EngineStatusRunning {
		t.Errorf("expected status running, got %s", result.Status)
	}
}

func TestProvider_ImportLocal(t *testing.T) {
	t.Run("file not found", func(t *testing.T) {
		p := NewProvider("")
		_, err := p.ImportLocal(context.Background(), "/nonexistent/path.gguf", false)
		if err == nil {
			t.Error("expected error for non-existent path")
		}
	})

	t.Run("stat error on unreadable dir", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "ollama-test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		ggufFile := filepath.Join(tmpDir, "test.gguf")
		if err := os.WriteFile(ggufFile, []byte("fake gguf"), 0644); err != nil {
			t.Fatal(err)
		}

		p := NewProvider("")
		m, err := p.ImportLocal(context.Background(), ggufFile, false)
		if err == nil {
			if m != nil {
				t.Error("expected error for ImportLocal with ollama command")
			}
		}
	})
}
