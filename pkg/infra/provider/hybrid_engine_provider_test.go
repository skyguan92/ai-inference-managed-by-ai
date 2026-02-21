package provider

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/docker"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/engine"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
)

// mockModelStore is a test double for model.ModelStore
type mockModelStore struct {
	mu     sync.RWMutex
	models map[string]*model.Model
	getErr error
}

func newMockModelStore() *mockModelStore {
	return &mockModelStore{
		models: make(map[string]*model.Model),
	}
}

func (m *mockModelStore) Create(ctx context.Context, mod *model.Model) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.models[mod.ID] = mod
	return nil
}

func (m *mockModelStore) Get(ctx context.Context, id string) (*model.Model, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getErr != nil {
		return nil, m.getErr
	}
	mod, ok := m.models[id]
	if !ok {
		return nil, errors.New("model not found: " + id)
	}
	return mod, nil
}

func (m *mockModelStore) List(ctx context.Context, filter model.ModelFilter) ([]model.Model, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []model.Model
	for _, mod := range m.models {
		result = append(result, *mod)
	}
	return result, len(result), nil
}

func (m *mockModelStore) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.models, id)
	return nil
}

func (m *mockModelStore) Update(ctx context.Context, mod *model.Model) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.models[mod.ID] = mod
	return nil
}

// Helpers to add models in tests
func (m *mockModelStore) addModel(mod *model.Model) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.models[mod.ID] = mod
}

// ---- Tests for ResourceLimits and StartupConfig defaults ----

func TestGetDefaultResourceLimits(t *testing.T) {
	limits := getDefaultResourceLimits()

	tests := []struct {
		engine    string
		wantGPU   bool
		wantCPU   float64
		wantMemNZ bool // memory is non-zero
	}{
		{"vllm", true, 0, false},     // vllm: GPU true, CPU 0, memory "0"
		{"whisper", false, 2.0, true}, // whisper: GPU false, CPU 2.0, memory "4g"
		{"asr", false, 2.0, true},    // asr: GPU false, CPU 2.0, memory "4g"
		{"tts", false, 2.0, true},    // tts: GPU false, CPU 2.0, memory "4g"
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			l, ok := limits[tt.engine]
			if !ok {
				t.Fatalf("expected limits for engine %q, got none", tt.engine)
			}
			if l.GPU != tt.wantGPU {
				t.Errorf("engine %q: GPU=%v, want %v", tt.engine, l.GPU, tt.wantGPU)
			}
			if l.CPU != tt.wantCPU {
				t.Errorf("engine %q: CPU=%.1f, want %.1f", tt.engine, l.CPU, tt.wantCPU)
			}
			if tt.wantMemNZ && (l.Memory == "" || l.Memory == "0") {
				t.Errorf("engine %q: expected non-zero memory limit, got %q", tt.engine, l.Memory)
			}
		})
	}
}

func TestGetDefaultResourceLimits_EnvOverride(t *testing.T) {
	// Test environment variable overrides
	t.Run("memory override", func(t *testing.T) {
		t.Setenv("AIMA_WHISPER_MEMORY", "8g")

		limits := getDefaultResourceLimits()
		if limits["whisper"].Memory != "8g" {
			t.Errorf("expected memory '8g', got %q", limits["whisper"].Memory)
		}
	})

	t.Run("cpu override", func(t *testing.T) {
		t.Setenv("AIMA_WHISPER_CPU", "4.0")

		limits := getDefaultResourceLimits()
		if limits["whisper"].CPU != 4.0 {
			t.Errorf("expected CPU 4.0, got %.1f", limits["whisper"].CPU)
		}
	})

	t.Run("gpu override true", func(t *testing.T) {
		t.Setenv("AIMA_WHISPER_GPU", "true")

		limits := getDefaultResourceLimits()
		if !limits["whisper"].GPU {
			t.Error("expected GPU=true after override")
		}
	})

	t.Run("gpu override 1", func(t *testing.T) {
		t.Setenv("AIMA_WHISPER_GPU", "1")

		limits := getDefaultResourceLimits()
		if !limits["whisper"].GPU {
			t.Error("expected GPU=true for value '1'")
		}
	})

	t.Run("gpu override false", func(t *testing.T) {
		t.Setenv("AIMA_VLLM_GPU", "false")

		limits := getDefaultResourceLimits()
		if limits["vllm"].GPU {
			t.Error("expected GPU=false after override")
		}
	})

	t.Run("invalid cpu override ignored", func(t *testing.T) {
		t.Setenv("AIMA_WHISPER_CPU", "not-a-number")

		// Should not panic, should keep default
		limits := getDefaultResourceLimits()
		if limits["whisper"].CPU != 2.0 {
			t.Errorf("expected default CPU 2.0 for invalid override, got %.1f", limits["whisper"].CPU)
		}
	})
}

func TestGetDefaultStartupConfigs(t *testing.T) {
	configs := getDefaultStartupConfigs()

	tests := []struct {
		engine         string
		wantMaxRetries int
		wantHealthPath string
	}{
		{"vllm", 5, "/health"},
		{"whisper", 3, "/"},
		{"asr", 3, "/"},
		{"tts", 3, "/health"},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			cfg, ok := configs[tt.engine]
			if !ok {
				t.Fatalf("expected config for engine %q, got none", tt.engine)
			}
			if cfg.MaxRetries != tt.wantMaxRetries {
				t.Errorf("engine %q: MaxRetries=%d, want %d", tt.engine, cfg.MaxRetries, tt.wantMaxRetries)
			}
			if cfg.HealthCheckURL != tt.wantHealthPath {
				t.Errorf("engine %q: HealthCheckURL=%q, want %q", tt.engine, cfg.HealthCheckURL, tt.wantHealthPath)
			}
			if cfg.StartupTimeout <= 0 {
				t.Errorf("engine %q: expected positive StartupTimeout, got %v", tt.engine, cfg.StartupTimeout)
			}
			if cfg.RetryInterval <= 0 {
				t.Errorf("engine %q: expected positive RetryInterval, got %v", tt.engine, cfg.RetryInterval)
			}
		})
	}
}

// ---- Tests for NewHybridEngineProvider ----

func TestNewHybridEngineProvider(t *testing.T) {
	store := newMockModelStore()
	p := NewHybridEngineProvider(store)

	if p == nil {
		t.Fatal("expected provider, got nil")
	}
	if p.containers == nil {
		t.Error("expected initialized containers map")
	}
	if p.nativeProcesses == nil {
		t.Error("expected initialized nativeProcesses map")
	}
	if p.serviceInfo == nil {
		t.Error("expected initialized serviceInfo map")
	}
	if p.resourceLimits == nil {
		t.Error("expected initialized resourceLimits map")
	}
	if p.startupConfigs == nil {
		t.Error("expected initialized startupConfigs map")
	}
	if p.modelStore != store {
		t.Error("expected modelStore to match provided store")
	}
}

// ---- Tests for GetFeatures ----

func TestHybridEngineProvider_GetFeatures(t *testing.T) {
	store := newMockModelStore()
	p := NewHybridEngineProvider(store)
	ctx := context.Background()

	tests := []struct {
		name                string
		engineName          string
		wantStreaming        bool
		wantBatch           bool
		wantTools           bool
		wantMaxConcurrent   int
		wantMaxContextLen   int
	}{
		{
			name:              "vllm features",
			engineName:        "vllm",
			wantStreaming:     true,
			wantBatch:         true,
			wantTools:         true,
			wantMaxConcurrent: 100,
			wantMaxContextLen: 128000,
		},
		{
			name:              "whisper features",
			engineName:        "whisper",
			wantStreaming:     false,
			wantMaxConcurrent: 10,
		},
		{
			name:              "asr features (alias)",
			engineName:        "asr",
			wantStreaming:     false,
			wantMaxConcurrent: 10,
		},
		{
			name:              "tts features",
			engineName:        "tts",
			wantStreaming:     false,
			wantMaxConcurrent: 10,
		},
		{
			name:              "unknown engine default features",
			engineName:        "unknown-engine",
			wantStreaming:     true,
			wantMaxConcurrent: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features, err := p.GetFeatures(ctx, tt.engineName)
			if err != nil {
				t.Fatalf("GetFeatures failed: %v", err)
			}
			if features == nil {
				t.Fatal("expected features, got nil")
			}
			if features.SupportsStreaming != tt.wantStreaming {
				t.Errorf("SupportsStreaming=%v, want %v", features.SupportsStreaming, tt.wantStreaming)
			}
			if features.MaxConcurrent != tt.wantMaxConcurrent {
				t.Errorf("MaxConcurrent=%d, want %d", features.MaxConcurrent, tt.wantMaxConcurrent)
			}
			if tt.wantMaxContextLen > 0 && features.MaxContextLength != tt.wantMaxContextLen {
				t.Errorf("MaxContextLength=%d, want %d", features.MaxContextLength, tt.wantMaxContextLen)
			}
			if tt.wantBatch && !features.SupportsBatch {
				t.Error("expected SupportsBatch=true")
			}
			if tt.wantTools && !features.SupportsTools {
				t.Error("expected SupportsTools=true")
			}
		})
	}
}

// ---- Tests for getDefaultPort ----

func TestHybridEngineProvider_getDefaultPort(t *testing.T) {
	p := NewHybridEngineProvider(newMockModelStore())

	tests := []struct {
		engineType string
		wantPort   int
	}{
		{"vllm", 8000},
		{"whisper", 8001},
		{"asr", 8001},
		{"tts", 8002},
		{"unknown", 8080},
		{"", 8080},
	}

	for _, tt := range tests {
		t.Run(tt.engineType, func(t *testing.T) {
			port := p.getDefaultPort(tt.engineType)
			if port != tt.wantPort {
				t.Errorf("getDefaultPort(%q)=%d, want %d", tt.engineType, port, tt.wantPort)
			}
		})
	}
}

// ---- Tests for getEngineTypeForModel ----

func TestHybridEngineProvider_getEngineTypeForModel(t *testing.T) {
	p := NewHybridEngineProvider(newMockModelStore())

	tests := []struct {
		modelType  model.ModelType
		wantEngine string
	}{
		{model.ModelTypeLLM, "vllm"},
		{model.ModelTypeVLM, "vllm"},
		{model.ModelTypeASR, "whisper"},
		{model.ModelTypeTTS, "tts"},
		{model.ModelTypeEmbedding, "vllm"},  // fallback to vllm
		{model.ModelTypeDiffusion, "vllm"},  // fallback to vllm
		{model.ModelTypeDetection, "vllm"},  // fallback to vllm
	}

	for _, tt := range tests {
		t.Run(string(tt.modelType), func(t *testing.T) {
			engineType := p.getEngineTypeForModel(tt.modelType)
			if engineType != tt.wantEngine {
				t.Errorf("getEngineTypeForModel(%q)=%q, want %q", tt.modelType, engineType, tt.wantEngine)
			}
		})
	}
}

// ---- Tests for getDockerImages ----

func TestHybridEngineProvider_getDockerImages(t *testing.T) {
	p := NewHybridEngineProvider(newMockModelStore())

	tests := []struct {
		name             string
		engineName       string
		version          string
		wantMinImages    int
		wantContains     string // must be in one of the images
	}{
		{
			name:          "vllm images",
			engineName:    "vllm",
			version:       "",
			wantMinImages: 3,
			wantContains:  "vllm",
		},
		{
			name:          "whisper images",
			engineName:    "whisper",
			version:       "latest",
			wantMinImages: 1,
			wantContains:  "",
		},
		{
			name:          "asr images",
			engineName:    "asr",
			version:       "",
			wantMinImages: 1,
		},
		{
			name:          "tts images",
			engineName:    "tts",
			version:       "",
			wantMinImages: 1,
		},
		{
			name:          "unknown engine uses default format",
			engineName:    "myengine",
			version:       "v1.0",
			wantMinImages: 1,
			wantContains:  "myengine:v1.0",
		},
		{
			name:          "unknown engine uses latest when version empty",
			engineName:    "myengine",
			version:       "",
			wantMinImages: 1,
			wantContains:  "myengine:latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			images := p.getDockerImages(tt.engineName, tt.version)
			if len(images) < tt.wantMinImages {
				t.Errorf("expected at least %d images, got %d", tt.wantMinImages, len(images))
			}
			if tt.wantContains != "" {
				found := false
				for _, img := range images {
					if strings.Contains(img, tt.wantContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected one image to contain %q, got %v", tt.wantContains, images)
				}
			}
		})
	}
}

// ---- Tests for validateModelPath ----

func TestValidateModelPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid absolute path",
			path:    "/models/my-model",
			wantErr: false,
		},
		{
			name:    "valid absolute path with subdirectories",
			path:    "/home/user/models/llm/qwen3",
			wantErr: false,
		},
		{
			name:    "path with semicolon (shell injection)",
			path:    "/models/test;rm -rf /",
			wantErr: true,
		},
		{
			name:    "path with ampersand",
			path:    "/models/test&bad",
			wantErr: true,
		},
		{
			name:    "path with pipe",
			path:    "/models/test|cat",
			wantErr: true,
		},
		{
			name:    "path with backtick",
			path:    "/models/test`cmd`",
			wantErr: true,
		},
		{
			name:    "path with dollar sign",
			path:    "/models/$HOME",
			wantErr: true,
		},
		{
			name:    "path with parenthesis",
			path:    "/models/(test)",
			wantErr: true,
		},
		{
			name:    "path with angle brackets",
			path:    "/models/<test>",
			wantErr: true,
		},
		{
			name:    "path with backslash",
			path:    "/models\\test",
			wantErr: true,
		},
		{
			name:    "path with single quote",
			path:    "/models/'test'",
			wantErr: true,
		},
		{
			name:    "path with double quote",
			path:    `/models/"test"`,
			wantErr: true,
		},
		{
			name:    "path with newline",
			path:    "/models/test\necho",
			wantErr: true,
		},
		{
			name:    "path with carriage return",
			path:    "/models/test\recmd",
			wantErr: true,
		},
		{
			name:    "path with directory traversal",
			path:    "/models/../etc/passwd",
			wantErr: true,
		},
		{
			name:    "relative path",
			path:    "models/test",
			wantErr: true,
		},
		{
			name:    "empty path (no leading slash)",
			path:    "test",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateModelPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateModelPath(%q): got err=%v, wantErr=%v", tt.path, err, tt.wantErr)
			}
		})
	}
}

// ---- Tests for buildDockerCommand ----

func TestHybridEngineProvider_buildDockerCommand(t *testing.T) {
	p := NewHybridEngineProvider(newMockModelStore())

	tests := []struct {
		name        string
		engineType  string
		image       string
		config      map[string]any
		port        int
		wantNil     bool
		wantContain string
	}{
		{
			name:       "vllm qwen3-omni-server image returns nil cmd",
			engineType: "vllm",
			image:      "aima-qwen3-omni-server:latest",
			config:     map[string]any{},
			port:       8000,
			wantNil:    true,
		},
		{
			name:        "vllm zhiwen-vllm image",
			engineType:  "vllm",
			image:       "zhiwen-vllm:0128",
			config:      map[string]any{},
			port:        8000,
			wantNil:     false,
			wantContain: "vllm",
		},
		{
			name:        "vllm default image",
			engineType:  "vllm",
			image:       "vllm/vllm-openai:v0.15.0",
			config:      map[string]any{},
			port:        8000,
			wantNil:     false,
			wantContain: "--model",
		},
		{
			name:        "vllm with custom gpu_memory_utilization",
			engineType:  "vllm",
			image:       "vllm/vllm-openai:v0.15.0",
			config:      map[string]any{"gpu_memory_utilization": 0.8},
			port:        8000,
			wantNil:     false,
			wantContain: "0.80",
		},
		{
			name:        "asr qujing-glm image",
			engineType:  "asr",
			image:       "qujing-glm-asr-nano:latest",
			config:      map[string]any{},
			port:        8001,
			wantNil:     false,
			wantContain: "uvicorn",
		},
		{
			name:        "asr funasr official image",
			engineType:  "asr",
			image:       "registry.cn-hangzhou.aliyuncs.com/funasr_repo/funasr:latest",
			config:      map[string]any{},
			port:        8001,
			wantNil:     false,
			wantContain: "/bin/bash",
		},
		{
			name:        "whisper funasr image (not qujing)",
			engineType:  "whisper",
			image:       "some-other-whisper:latest",
			config:      map[string]any{},
			port:        8001,
			wantNil:     false,
			wantContain: "/bin/bash",
		},
		{
			name:        "tts qujing image",
			engineType:  "tts",
			image:       "qujing-qwen3-tts:latest",
			config:      map[string]any{},
			port:        8002,
			wantNil:     false,
			wantContain: "uvicorn",
		},
		{
			name:        "tts official coqui image",
			engineType:  "tts",
			image:       "ghcr.io/coqui-ai/tts:latest",
			config:      map[string]any{},
			port:        8002,
			wantNil:     false,
			wantContain: "python",
		},
		{
			name:        "unknown engine type",
			engineType:  "custom",
			image:       "custom-engine:latest",
			config:      map[string]any{},
			port:        8080,
			wantNil:     false,
			wantContain: "--port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := p.buildDockerCommand(tt.engineType, tt.image, tt.config, tt.port)

			if tt.wantNil {
				if cmd != nil {
					t.Errorf("expected nil command, got %v", cmd)
				}
				return
			}

			if cmd == nil {
				t.Fatal("expected non-nil command, got nil")
			}
			if tt.wantContain != "" {
				found := false
				for _, arg := range cmd {
					if strings.Contains(arg, tt.wantContain) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("command args %v don't contain %q", cmd, tt.wantContain)
				}
			}
		})
	}
}

// ---- Tests for Stop (docker and native) ----

func TestHybridEngineProvider_Stop_NoProcess(t *testing.T) {
	p := NewHybridEngineProvider(newMockModelStore())
	ctx := context.Background()

	// Stop when no container or process exists should succeed
	result, err := p.Stop(ctx, "vllm", false, 10)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true for stop with no process")
	}
}

func TestHybridEngineProvider_Stop_DockerContainer(t *testing.T) {
	p := NewHybridEngineProvider(newMockModelStore())
	ctx := context.Background()

	// Inject a fake container ID (won't actually call Docker - will get an error since docker is not running in tests)
	p.mu.Lock()
	p.containers["vllm"] = "abc123containerid"
	p.mu.Unlock()

	// The stop will try to contact docker, which will fail, but that's expected behavior in unit tests
	// We verify that the container was attempted to be stopped and removed on success
	// Since docker is not available in tests, we just ensure it doesn't panic
	_, _ = p.Stop(ctx, "vllm", false, 1)
}

func TestHybridEngineProvider_Stop_ConcurrentSafety(t *testing.T) {
	// Use mock Docker client to avoid data race inside Docker SDK's
	// API version negotiation (negotiateAPIVersionPing) when multiple
	// goroutines hit the real client concurrently.
	p := newHybridEngineProviderWithClient(newMockModelStore(), docker.NewMockClient())
	ctx := context.Background()

	var wg sync.WaitGroup
	// Concurrently call Stop to verify no data races
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = p.Stop(ctx, "vllm", false, 1)
		}()
	}
	wg.Wait()
}

// ---- Tests for Install ----

func TestHybridEngineProvider_Install_FallbackBestEffort(t *testing.T) {
	p := NewHybridEngineProvider(newMockModelStore())
	ctx := context.Background()

	// When neither docker nor native binary is available, Install still returns success (best-effort mode)
	result, err := p.Install(ctx, "nonexistent-engine-xyz", "")
	if err != nil {
		t.Fatalf("Install should not return error in best-effort mode: %v", err)
	}
	if !result.Success {
		t.Error("expected Success=true in best-effort mode")
	}
}

// ---- Tests for concurrent access to containers and nativeProcesses maps ----

func TestHybridEngineProvider_ConcurrentMapAccess(t *testing.T) {
	p := newHybridEngineProviderWithClient(newMockModelStore(), docker.NewMockClient())

	var wg sync.WaitGroup

	// Concurrent reads and writes to containers
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			engineName := "engine-test"
			// Concurrent write
			p.mu.Lock()
			p.containers[engineName] = "container-id"
			p.mu.Unlock()

			// Concurrent read
			p.mu.RLock()
			_ = p.containers[engineName]
			p.mu.RUnlock()
		}(i)
	}

	// Concurrent reads and writes to nativeProcesses
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			p.mu.Lock()
			delete(p.nativeProcesses, "engine-test")
			p.mu.Unlock()

			p.mu.RLock()
			_ = p.nativeProcesses["engine-test"]
			p.mu.RUnlock()
		}(i)
	}

	wg.Wait()
}

// ---- Tests for HybridServiceProvider ----

func TestNewHybridServiceProvider(t *testing.T) {
	store := newMockModelStore()
	p := NewHybridServiceProvider(store, service.NewMemoryStore())

	if p == nil {
		t.Fatal("expected provider, got nil")
	}
	if p.hybridProvider == nil {
		t.Error("expected initialized hybridProvider")
	}
	if p.portCounter != 8000 {
		t.Errorf("expected portCounter=8000, got %d", p.portCounter)
	}
	if p.startupOrder == nil {
		t.Error("expected initialized startupOrder slice")
	}
}

func TestHybridServiceProvider_Create(t *testing.T) {
	store := newMockModelStore()
	store.addModel(&model.Model{
		ID:   "model-llm-001",
		Name: "my-llm",
		Type: model.ModelTypeLLM,
		Path: "/models/my-llm",
	})
	store.addModel(&model.Model{
		ID:   "model-asr-001",
		Name: "my-asr",
		Type: model.ModelTypeASR,
		Path: "/models/my-asr",
	})
	store.addModel(&model.Model{
		ID:   "model-tts-001",
		Name: "my-tts",
		Type: model.ModelTypeTTS,
		Path: "/models/my-tts",
	})

	p := NewHybridServiceProvider(store, service.NewMemoryStore())
	ctx := context.Background()

	tests := []struct {
		name          string
		modelID       string
		resourceClass service.ResourceClass
		replicas      int
		wantEngine    string
		wantGPU       bool
	}{
		{
			name:          "LLM model uses vllm engine with GPU",
			modelID:       "model-llm-001",
			resourceClass: service.ResourceClassLarge,
			replicas:      1,
			wantEngine:    "vllm",
			wantGPU:       true,
		},
		{
			name:          "ASR model uses whisper engine without GPU",
			modelID:       "model-asr-001",
			resourceClass: service.ResourceClassSmall,
			replicas:      1,
			wantEngine:    "whisper",
			wantGPU:       false,
		},
		{
			name:          "TTS model uses tts engine without GPU",
			modelID:       "model-tts-001",
			resourceClass: service.ResourceClassSmall,
			replicas:      1,
			wantEngine:    "tts",
			wantGPU:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := p.Create(ctx, tt.modelID, tt.resourceClass, tt.replicas, false)
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}
			if svc == nil {
				t.Fatal("expected service, got nil")
			}
			if svc.ModelID != tt.modelID {
				t.Errorf("ModelID=%q, want %q", svc.ModelID, tt.modelID)
			}
			if svc.Replicas != tt.replicas {
				t.Errorf("Replicas=%d, want %d", svc.Replicas, tt.replicas)
			}
			if svc.Status != service.ServiceStatusCreating {
				t.Errorf("Status=%q, want ServiceStatusCreating", svc.Status)
			}
			if len(svc.Endpoints) == 0 {
				t.Error("expected at least one endpoint")
			}
			if svc.Config == nil {
				t.Error("expected config map")
			}
			engineType, _ := svc.Config["engine_type"].(string)
			if engineType != tt.wantEngine {
				t.Errorf("engine_type=%q, want %q", engineType, tt.wantEngine)
			}
			gpuVal, _ := svc.Config["gpu"].(bool)
			if gpuVal != tt.wantGPU {
				t.Errorf("gpu=%v, want %v", gpuVal, tt.wantGPU)
			}
		})
	}
}

func TestHybridServiceProvider_Create_ModelNotFound(t *testing.T) {
	store := newMockModelStore()
	p := NewHybridServiceProvider(store, service.NewMemoryStore())
	ctx := context.Background()

	_, err := p.Create(ctx, "nonexistent-model-id", service.ResourceClassSmall, 1, false)
	if err == nil {
		t.Error("expected error for missing model")
	}
}

func TestHybridServiceProvider_Create_PortIncrement(t *testing.T) {
	store := newMockModelStore()
	store.addModel(&model.Model{ID: "m1", Name: "model1", Type: model.ModelTypeLLM})
	store.addModel(&model.Model{ID: "m2", Name: "model2", Type: model.ModelTypeLLM})

	p := NewHybridServiceProvider(store, service.NewMemoryStore())
	ctx := context.Background()

	svc1, err := p.Create(ctx, "m1", service.ResourceClassMedium, 1, false)
	if err != nil {
		t.Fatalf("Create 1 failed: %v", err)
	}
	svc2, err := p.Create(ctx, "m2", service.ResourceClassMedium, 1, false)
	if err != nil {
		t.Fatalf("Create 2 failed: %v", err)
	}

	// Port should increment between calls
	if len(svc1.Endpoints) == 0 || len(svc2.Endpoints) == 0 {
		t.Fatal("both services should have endpoints")
	}
	if svc1.Endpoints[0] == svc2.Endpoints[0] {
		t.Error("expected different endpoints (different ports) for different services")
	}
}

func TestHybridServiceProvider_Stop(t *testing.T) {
	store := newMockModelStore()
	p := NewHybridServiceProvider(store, service.NewMemoryStore())
	ctx := context.Background()

	// Stop for a non-existent service should not panic
	err := p.Stop(ctx, "svc-vllm-model-abc", false)
	// Error or nil depends on docker availability; just verify no panic
	_ = err
}

func TestHybridServiceProvider_Scale(t *testing.T) {
	store := newMockModelStore()
	p := NewHybridServiceProvider(store, service.NewMemoryStore())
	ctx := context.Background()

	err := p.Scale(ctx, "svc-any", 3)
	if err == nil {
		t.Error("expected error: scale not yet implemented")
	}
	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("expected 'not yet implemented' in error, got %q", err.Error())
	}
}

func TestHybridServiceProvider_GetMetrics(t *testing.T) {
	store := newMockModelStore()
	p := NewHybridServiceProvider(store, service.NewMemoryStore())
	ctx := context.Background()

	metrics, err := p.GetMetrics(ctx, "any-service-id")
	if err != nil {
		t.Fatalf("GetMetrics failed: %v", err)
	}
	if metrics == nil {
		t.Error("expected metrics struct, got nil")
	}
}

func TestHybridServiceProvider_GetRecommendation(t *testing.T) {
	store := newMockModelStore()
	store.addModel(&model.Model{
		ID:   "llm-model",
		Name: "qwen3",
		Type: model.ModelTypeLLM,
	})
	store.addModel(&model.Model{
		ID:   "asr-model",
		Name: "whisper-base",
		Type: model.ModelTypeASR,
	})
	store.addModel(&model.Model{
		ID:   "tts-model",
		Name: "kokoro-tts",
		Type: model.ModelTypeTTS,
	})

	p := NewHybridServiceProvider(store, service.NewMemoryStore())
	ctx := context.Background()

	tests := []struct {
		name           string
		modelID        string
		wantEngine     string
		wantDevice     string
		wantClass      service.ResourceClass
	}{
		{
			name:       "LLM recommendation",
			modelID:    "llm-model",
			wantEngine: "vllm",
			wantDevice: "gpu",
			wantClass:  service.ResourceClassLarge,
		},
		{
			name:       "ASR recommendation",
			modelID:    "asr-model",
			wantEngine: "whisper",
			wantDevice: "cpu",
			wantClass:  service.ResourceClassSmall,
		},
		{
			name:       "TTS recommendation",
			modelID:    "tts-model",
			wantEngine: "tts",
			wantDevice: "cpu",
			wantClass:  service.ResourceClassSmall,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, err := p.GetRecommendation(ctx, tt.modelID, "")
			if err != nil {
				t.Fatalf("GetRecommendation failed: %v", err)
			}
			if rec.EngineType != tt.wantEngine {
				t.Errorf("EngineType=%q, want %q", rec.EngineType, tt.wantEngine)
			}
			if rec.DeviceType != tt.wantDevice {
				t.Errorf("DeviceType=%q, want %q", rec.DeviceType, tt.wantDevice)
			}
			if rec.ResourceClass != tt.wantClass {
				t.Errorf("ResourceClass=%q, want %q", rec.ResourceClass, tt.wantClass)
			}
			if rec.Replicas != 1 {
				t.Errorf("Replicas=%d, want 1", rec.Replicas)
			}
			if rec.Reason == "" {
				t.Error("expected non-empty Reason")
			}
		})
	}
}

func TestHybridServiceProvider_GetRecommendation_ModelNotFound(t *testing.T) {
	store := newMockModelStore()
	p := NewHybridServiceProvider(store, service.NewMemoryStore())
	ctx := context.Background()

	_, err := p.GetRecommendation(ctx, "does-not-exist", "")
	if err == nil {
		t.Error("expected error for non-existent model")
	}
}

func TestHybridServiceProvider_IsRunning_NoInfo(t *testing.T) {
	store := newMockModelStore()
	p := NewHybridServiceProvider(store, service.NewMemoryStore())
	ctx := context.Background()

	// serviceInfo is empty, so IsRunning should return false
	running := p.IsRunning(ctx, "svc-any")
	if running {
		t.Error("expected IsRunning=false when no service info")
	}
}

func TestHybridServiceProvider_IsRunning_WithInfo(t *testing.T) {
	store := newMockModelStore()
	p := NewHybridServiceProvider(store, service.NewMemoryStore())
	ctx := context.Background()

	// Inject service info with short ProcessID (not a Docker container ID)
	p.hybridProvider.mu.Lock()
	p.hybridProvider.serviceInfo["svc-test"] = &ServiceInfo{
		ServiceID: "svc-test",
		ModelID:   "model1",
		Engine:    "vllm",
		Port:      8000,
		ProcessID: "12345", // short PID, not Docker
	}
	p.hybridProvider.mu.Unlock()

	// For native processes with a short ProcessID, IsRunning returns true (simplified check)
	running := p.IsRunning(ctx, "svc-test")
	if !running {
		t.Error("expected IsRunning=true for native process (simplified check)")
	}
}

func TestHybridServiceProvider_IsRunning_EmptyProcessID(t *testing.T) {
	store := newMockModelStore()
	p := NewHybridServiceProvider(store, service.NewMemoryStore())
	ctx := context.Background()

	p.hybridProvider.mu.Lock()
	p.hybridProvider.serviceInfo["svc-empty"] = &ServiceInfo{
		ServiceID: "svc-empty",
		ProcessID: "", // empty
	}
	p.hybridProvider.mu.Unlock()

	running := p.IsRunning(ctx, "svc-empty")
	if running {
		t.Error("expected IsRunning=false for empty ProcessID")
	}
}

func TestHybridServiceProvider_GetEngineProvider(t *testing.T) {
	store := newMockModelStore()
	p := NewHybridServiceProvider(store, service.NewMemoryStore())

	ep := p.GetEngineProvider()
	if ep == nil {
		t.Error("expected non-nil engine provider")
	}
	// Verify it implements engine.EngineProvider
	var _ engine.EngineProvider = ep
}

// ---- Tests for interface compliance ----

func TestHybridEngineProvider_ImplementsInterface(t *testing.T) {
	store := newMockModelStore()
	p := NewHybridEngineProvider(store)
	var _ engine.EngineProvider = p
}

func TestHybridServiceProvider_ImplementsInterface(t *testing.T) {
	store := newMockModelStore()
	p := NewHybridServiceProvider(store, service.NewMemoryStore())
	var _ service.ServiceProvider = p
}

// ---- Tests for ResourceLimits struct ----

func TestResourceLimits_Fields(t *testing.T) {
	rl := ResourceLimits{
		Memory:    "4g",
		CPU:       2.0,
		GPU:       true,
		GPUMemory: "10g",
	}

	if rl.Memory != "4g" {
		t.Errorf("expected Memory '4g', got %q", rl.Memory)
	}
	if rl.CPU != 2.0 {
		t.Errorf("expected CPU 2.0, got %f", rl.CPU)
	}
	if !rl.GPU {
		t.Error("expected GPU=true")
	}
	if rl.GPUMemory != "10g" {
		t.Errorf("expected GPUMemory '10g', got %q", rl.GPUMemory)
	}
}

// ---- Tests for StartupConfig struct ----

func TestStartupConfig_Fields(t *testing.T) {
	cfg := StartupConfig{
		MaxRetries:     5,
		RetryInterval:  10 * time.Second,
		StartupTimeout: 60 * time.Second,
		HealthCheckURL: "/health",
	}

	if cfg.MaxRetries != 5 {
		t.Errorf("expected MaxRetries 5, got %d", cfg.MaxRetries)
	}
	if cfg.RetryInterval != 10*time.Second {
		t.Errorf("expected RetryInterval 10s, got %v", cfg.RetryInterval)
	}
	if cfg.StartupTimeout != 60*time.Second {
		t.Errorf("expected StartupTimeout 60s, got %v", cfg.StartupTimeout)
	}
	if cfg.HealthCheckURL != "/health" {
		t.Errorf("expected HealthCheckURL '/health', got %q", cfg.HealthCheckURL)
	}
}

// ---- Tests for ServiceInfo struct ----

func TestServiceInfo_Fields(t *testing.T) {
	info := &ServiceInfo{
		ServiceID: "svc-vllm-001",
		ModelID:   "model-001",
		Engine:    "vllm",
		Port:      8000,
		UseGPU:    true,
		ProcessID: "abc123",
		Endpoint:  "http://localhost:8000",
	}

	if info.ServiceID != "svc-vllm-001" {
		t.Errorf("ServiceID=%q, want 'svc-vllm-001'", info.ServiceID)
	}
	if info.Port != 8000 {
		t.Errorf("Port=%d, want 8000", info.Port)
	}
	if !info.UseGPU {
		t.Error("expected UseGPU=true")
	}
}

// ---- Concurrent tests for HybridServiceProvider ----

func TestHybridServiceProvider_ConcurrentCreate(t *testing.T) {
	store := newMockModelStore()
	store.addModel(&model.Model{
		ID:   "llm-concurrent",
		Name: "test-llm",
		Type: model.ModelTypeLLM,
	})

	p := NewHybridServiceProvider(store, service.NewMemoryStore())
	ctx := context.Background()

	const numGoroutines = 20
	var wg sync.WaitGroup
	errs := make(chan error, numGoroutines)
	svcs := make(chan *service.ModelService, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc, err := p.Create(ctx, "llm-concurrent", service.ResourceClassSmall, 1, false)
			if err != nil {
				errs <- err
				return
			}
			svcs <- svc
		}()
	}
	wg.Wait()
	close(errs)
	close(svcs)

	for err := range errs {
		t.Errorf("concurrent Create failed: %v", err)
	}

	// Verify each goroutine received a unique port — the mutex in Create()
	// must guarantee that portCounter reads and increments are atomic.
	ports := make(map[int]bool)
	for svc := range svcs {
		port, ok := svc.Config["port"].(int)
		if !ok {
			t.Errorf("service %s has non-int port in Config: %T", svc.ID, svc.Config["port"])
			continue
		}
		if ports[port] {
			t.Errorf("duplicate port %d assigned to service %s", port, svc.ID)
		}
		ports[port] = true
	}
	if len(ports) != numGoroutines {
		t.Errorf("expected %d unique ports, got %d", numGoroutines, len(ports))
	}
}

func TestHybridServiceProvider_Start_InvalidServiceID(t *testing.T) {
	store := newMockModelStore()
	p := NewHybridServiceProvider(store, service.NewMemoryStore())
	ctx := context.Background()

	// Service ID with too few parts
	err := p.Start(ctx, "svc")
	if err == nil {
		t.Error("expected error for invalid service ID")
	}
}

func TestHybridServiceProvider_StartAsync_ModelNotFound(t *testing.T) {
	store := newMockModelStore()
	p := NewHybridServiceProvider(store, service.NewMemoryStore())
	ctx := context.Background()

	// Valid format but model doesn't exist
	err := p.StartAsync(ctx, "svc-vllm-missing-model", false)
	if err == nil {
		t.Error("expected error when model not found")
	}
}
