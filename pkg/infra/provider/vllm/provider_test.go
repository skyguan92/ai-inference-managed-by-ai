package vllm

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
)

// --- mockModelStore ---

type mockModelStore struct {
	mu     sync.RWMutex
	models map[string]*model.Model
}

func newMockModelStore() *mockModelStore {
	return &mockModelStore{models: make(map[string]*model.Model)}
}

func (m *mockModelStore) Create(ctx context.Context, mdl *model.Model) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.models[mdl.ID] = mdl
	return nil
}

func (m *mockModelStore) Get(ctx context.Context, id string) (*model.Model, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mdl, ok := m.models[id]
	if !ok {
		return nil, model.ErrModelNotFound
	}
	return mdl, nil
}

func (m *mockModelStore) List(ctx context.Context, filter model.ModelFilter) ([]model.Model, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []model.Model
	for _, mdl := range m.models {
		result = append(result, *mdl)
	}
	return result, len(result), nil
}

func (m *mockModelStore) Update(ctx context.Context, mdl *model.Model) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.models[mdl.ID] = mdl
	return nil
}

func (m *mockModelStore) Delete(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.models, id)
	return nil
}

// --- NewProvider ---

func TestNewProvider_NotNil(t *testing.T) {
	p := NewProvider(nil)
	if p == nil {
		t.Fatal("expected non-nil Provider")
	}
}

func TestNewProvider_MapsInitialized(t *testing.T) {
	p := NewProvider(nil)
	if p.processes == nil {
		t.Error("expected processes map to be initialized")
	}
	if p.services == nil {
		t.Error("expected services map to be initialized")
	}
}

func TestNewProvider_WithModelStore(t *testing.T) {
	store := newMockModelStore()
	p := NewProvider(store)
	if p.modelStore != store {
		t.Error("expected modelStore to be set")
	}
}

// --- NewServiceProvider ---

func TestNewServiceProvider_NotNil(t *testing.T) {
	sp := NewServiceProvider(nil)
	if sp == nil {
		t.Fatal("expected non-nil ServiceProvider")
	}
	if sp.provider == nil {
		t.Error("expected inner provider to be set")
	}
}

func TestNewServiceProvider_WithModelStore(t *testing.T) {
	store := newMockModelStore()
	sp := NewServiceProvider(store)
	if sp.provider.modelStore != store {
		t.Error("expected inner provider to have model store")
	}
}

func TestNewServiceProvider_WithNonStoreInterface(t *testing.T) {
	// Passing an interface that does NOT implement model.ModelStore should result
	// in a nil inner modelStore (the type assertion fails silently).
	sp := NewServiceProvider("not a model store")
	if sp == nil {
		t.Fatal("expected non-nil ServiceProvider")
	}
	if sp.provider.modelStore != nil {
		t.Error("expected nil modelStore for invalid input")
	}
}

func TestServiceProvider_ImplementsInterface(t *testing.T) {
	var _ service.ServiceProvider = (*ServiceProvider)(nil)
}

// --- Create ---

func TestProvider_Create_WithNilModelStore_UsesDefaultPath(t *testing.T) {
	p := NewProvider(nil)
	svc, err := p.Create(context.Background(), "llama3-8b", service.ResourceClassMedium, 1, false)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.ModelID != "llama3-8b" {
		t.Errorf("expected ModelID 'llama3-8b', got %s", svc.ModelID)
	}
	if !strings.HasPrefix(svc.ID, "svc-vllm-") {
		t.Errorf("expected service ID to start with 'svc-vllm-', got %s", svc.ID)
	}
	if len(svc.Endpoints) == 0 {
		t.Error("expected at least one endpoint")
	}
	if !strings.HasPrefix(svc.Endpoints[0], "http://localhost:") {
		t.Errorf("expected endpoint to start with 'http://localhost:', got %s", svc.Endpoints[0])
	}
	if svc.Status != service.ServiceStatusCreating {
		t.Errorf("expected status creating, got %s", svc.Status)
	}
}

func TestProvider_Create_WithModelStore_UsesModelPath(t *testing.T) {
	store := newMockModelStore()
	_ = store.Create(context.Background(), &model.Model{
		ID:   "model-001",
		Name: "llama3",
		Path: "/data/models/llama3",
	})

	p := NewProvider(store)
	svc, err := p.Create(context.Background(), "model-001", service.ResourceClassLarge, 2, false)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if svc.ModelID != "model-001" {
		t.Errorf("expected ModelID 'model-001', got %s", svc.ModelID)
	}
	if svc.Replicas != 2 {
		t.Errorf("expected replicas 2, got %d", svc.Replicas)
	}

	// Verify ServiceInfo was stored
	p.mu.RLock()
	info, exists := p.services[svc.ID]
	p.mu.RUnlock()
	if !exists {
		t.Error("expected service info to be stored in provider")
	}
	if info.ModelPath != "/data/models/llama3" {
		t.Errorf("expected model path '/data/models/llama3', got %s", info.ModelPath)
	}
}

func TestProvider_Create_ModelNotFound_ReturnsError(t *testing.T) {
	store := newMockModelStore() // empty store
	p := NewProvider(store)

	_, err := p.Create(context.Background(), "nonexistent-model", service.ResourceClassSmall, 1, false)
	if err == nil {
		t.Error("expected error for model not found")
	}
	if !strings.Contains(err.Error(), "model not found") {
		t.Errorf("expected 'model not found' in error, got: %v", err)
	}
}

func TestProvider_Create_UniqueServiceIDs(t *testing.T) {
	p := NewProvider(nil)
	ids := make(map[string]struct{})
	for i := 0; i < 10; i++ {
		svc, err := p.Create(context.Background(), "model-x", service.ResourceClassSmall, 1, false)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if _, exists := ids[svc.ID]; exists {
			t.Errorf("duplicate service ID: %s", svc.ID)
		}
		ids[svc.ID] = struct{}{}
	}
}

// --- allocatePort ---

func TestProvider_AllocatePort_Default(t *testing.T) {
	p := NewProvider(nil)
	port := p.allocatePort()
	if port != 8000 {
		t.Errorf("expected default port 8000 for empty provider, got %d", port)
	}
}

func TestProvider_AllocatePort_Increments(t *testing.T) {
	p := NewProvider(nil)

	// Occupy port 8000
	p.mu.Lock()
	p.services["svc-a"] = &ServiceInfo{Port: 8000}
	p.mu.Unlock()

	port := p.allocatePort()
	if port != 8001 {
		t.Errorf("expected port 8001 after 8000 is taken, got %d", port)
	}
}

func TestProvider_AllocatePort_SkipsOccupied(t *testing.T) {
	p := NewProvider(nil)
	p.mu.Lock()
	p.services["svc-a"] = &ServiceInfo{Port: 8000}
	p.services["svc-b"] = &ServiceInfo{Port: 8001}
	p.services["svc-c"] = &ServiceInfo{Port: 8002}
	p.mu.Unlock()

	port := p.allocatePort()
	if port != 8003 {
		t.Errorf("expected port 8003, got %d", port)
	}
}

// --- buildServeArgs ---

func TestProvider_BuildServeArgs_BasicPort(t *testing.T) {
	p := NewProvider(nil)
	info := &ServiceInfo{Port: 8080}
	args := p.buildServeArgs(info)

	portIdx := -1
	for i, a := range args {
		if a == "--port" {
			portIdx = i
			break
		}
	}
	if portIdx < 0 {
		t.Fatalf("expected --port in args, got: %v", args)
	}
	if args[portIdx+1] != "8080" {
		t.Errorf("expected port value '8080', got '%s'", args[portIdx+1])
	}
}

func TestProvider_BuildServeArgs_SingleGPUNoTensorParallel(t *testing.T) {
	p := NewProvider(nil)
	info := &ServiceInfo{Port: 8000, GPUs: []int{0}}
	args := p.buildServeArgs(info)

	for _, a := range args {
		if a == "--tensor-parallel-size" {
			t.Error("expected no --tensor-parallel-size for single GPU")
		}
	}
}

func TestProvider_BuildServeArgs_MultiGPUAddsTensorParallel(t *testing.T) {
	p := NewProvider(nil)
	info := &ServiceInfo{Port: 8000, GPUs: []int{0, 1, 2, 3}}
	args := p.buildServeArgs(info)

	tpIdx := -1
	for i, a := range args {
		if a == "--tensor-parallel-size" {
			tpIdx = i
			break
		}
	}
	if tpIdx < 0 {
		t.Fatalf("expected --tensor-parallel-size in args for 4 GPUs, got: %v", args)
	}
	if args[tpIdx+1] != "4" {
		t.Errorf("expected tensor-parallel-size '4', got '%s'", args[tpIdx+1])
	}
}

func TestProvider_BuildServeArgs_GPUMemoryUtilization(t *testing.T) {
	p := NewProvider(nil)
	info := &ServiceInfo{Port: 8000}
	args := p.buildServeArgs(info)

	found := false
	for i, a := range args {
		if a == "--gpu-memory-utilization" {
			found = true
			if args[i+1] != "0.9" {
				t.Errorf("expected gpu-memory-utilization '0.9', got '%s'", args[i+1])
			}
			break
		}
	}
	if !found {
		t.Errorf("expected --gpu-memory-utilization in args, got: %v", args)
	}
}

// --- Stop ---

func TestProvider_Stop_ServiceNotRunning(t *testing.T) {
	p := NewProvider(nil)
	err := p.Stop(context.Background(), "nonexistent-svc", true)
	if err == nil {
		t.Error("expected error for stopping non-existent service")
	}
	if !strings.Contains(err.Error(), "service not running") {
		t.Errorf("expected 'service not running' in error, got: %v", err)
	}
}

// --- Scale ---

func TestProvider_Scale_ReturnsNotImplemented(t *testing.T) {
	p := NewProvider(nil)
	err := p.Scale(context.Background(), "svc-123", 3)
	if err == nil {
		t.Error("expected error for unimplemented Scale")
	}
}

// --- GetRecommendation ---

func TestProvider_GetRecommendation_ReturnsLargeClass(t *testing.T) {
	p := NewProvider(nil)
	rec, err := p.GetRecommendation(context.Background(), "llama3-70b", "")
	if err != nil {
		t.Fatalf("GetRecommendation failed: %v", err)
	}
	if rec == nil {
		t.Fatal("expected non-nil recommendation")
	}
	if rec.ResourceClass != service.ResourceClassLarge {
		t.Errorf("expected ResourceClassLarge, got %s", rec.ResourceClass)
	}
	if rec.Replicas != 1 {
		t.Errorf("expected 1 replica, got %d", rec.Replicas)
	}
	if rec.ExpectedThroughput <= 0 {
		t.Errorf("expected positive throughput, got %f", rec.ExpectedThroughput)
	}
}

// --- SetGPUDevices ---

func TestProvider_SetGPUDevices_ServiceExists(t *testing.T) {
	p := NewProvider(nil)
	p.mu.Lock()
	p.services["svc-x"] = &ServiceInfo{ServiceID: "svc-x"}
	p.mu.Unlock()

	err := p.SetGPUDevices("svc-x", []int{0, 1})
	if err != nil {
		t.Fatalf("SetGPUDevices failed: %v", err)
	}

	p.mu.RLock()
	defer p.mu.RUnlock()
	info := p.services["svc-x"]
	if len(info.GPUs) != 2 || info.GPUs[0] != 0 || info.GPUs[1] != 1 {
		t.Errorf("expected GPUs [0, 1], got %v", info.GPUs)
	}
}

func TestProvider_SetGPUDevices_ServiceNotFound(t *testing.T) {
	p := NewProvider(nil)
	err := p.SetGPUDevices("nonexistent", []int{0})
	if err == nil {
		t.Error("expected error for non-existent service")
	}
}

// --- GetServiceInfo ---

func TestProvider_GetServiceInfo_Found(t *testing.T) {
	p := NewProvider(nil)
	p.mu.Lock()
	p.services["svc-y"] = &ServiceInfo{
		ServiceID: "svc-y",
		ModelID:   "model-abc",
		Port:      8080,
		Status:    "running",
	}
	p.mu.Unlock()

	info, err := p.GetServiceInfo("svc-y")
	if err != nil {
		t.Fatalf("GetServiceInfo failed: %v", err)
	}
	if info.ServiceID != "svc-y" {
		t.Errorf("expected ServiceID 'svc-y', got %s", info.ServiceID)
	}
	if info.Port != 8080 {
		t.Errorf("expected Port 8080, got %d", info.Port)
	}
}

func TestProvider_GetServiceInfo_NotFound(t *testing.T) {
	p := NewProvider(nil)
	_, err := p.GetServiceInfo("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent service")
	}
}

// --- GetMetrics ---

func TestProvider_GetMetrics_ServiceNotFound(t *testing.T) {
	p := NewProvider(nil)
	_, err := p.GetMetrics(context.Background(), "nonexistent-svc")
	if err == nil {
		t.Error("expected error for non-existent service")
	}
	if !strings.Contains(err.Error(), "service not found") {
		t.Errorf("expected 'service not found' in error, got: %v", err)
	}
}

func TestProvider_GetMetrics_ServiceExists_EndpointDown(t *testing.T) {
	// When the metrics endpoint is unreachable, scrapeMetrics returns zeros (not error).
	p := NewProvider(nil)
	p.mu.Lock()
	p.services["svc-metrics"] = &ServiceInfo{
		ServiceID: "svc-metrics",
		Port:      59998, // port nothing is listening on
	}
	p.mu.Unlock()

	metrics, err := p.GetMetrics(context.Background(), "svc-metrics")
	if err != nil {
		t.Fatalf("GetMetrics failed: %v", err)
	}
	if metrics == nil {
		t.Fatal("expected non-nil metrics even when endpoint is down")
	}
}

// --- isRunning ---

func TestProvider_IsRunning_NoProcess(t *testing.T) {
	p := NewProvider(nil)
	if p.isRunning("nonexistent") {
		t.Error("expected isRunning to be false for unknown service")
	}
}

// --- parsePromLine ---

func TestParsePromLine_BasicMetric(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantName  string
		wantValue float64
		wantOK    bool
	}{
		{
			name:      "simple metric",
			line:      "vllm:request_success_total 42",
			wantName:  "vllm:request_success_total",
			wantValue: 42,
			wantOK:    true,
		},
		{
			name:      "metric with labels",
			line:      `vllm:e2e_request_latency_seconds_sum{model="llama3"} 1.5`,
			wantName:  "vllm:e2e_request_latency_seconds_sum",
			wantValue: 1.5,
			wantOK:    true,
		},
		{
			name:      "metric with timestamp",
			line:      "vllm:request_failure_total 7 1620000000000",
			wantName:  "vllm:request_failure_total",
			wantValue: 7,
			wantOK:    true,
		},
		{
			name:      "+Inf sentinel",
			line:      `vllm:latency_bucket{le="+Inf"} +Inf`,
			wantName:  "vllm:latency_bucket",
			wantValue: 0,
			wantOK:    true,
		},
		{
			name:      "NaN sentinel",
			line:      "some_metric NaN",
			wantName:  "some_metric",
			wantValue: 0,
			wantOK:    true,
		},
		{
			name:   "comment line",
			line:   "# HELP vllm:foo A metric",
			wantOK: false,
		},
		{
			name:   "empty line",
			line:   "",
			wantOK: false,
		},
		{
			name:   "no space separator",
			line:   "nospace",
			wantOK: false,
		},
		{
			name:   "invalid float value",
			line:   "metric_name notafloat",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, value, ok := parsePromLine(tt.line)
			if ok != tt.wantOK {
				t.Errorf("expected ok=%v, got ok=%v (name=%q, value=%f)", tt.wantOK, ok, name, value)
				return
			}
			if !tt.wantOK {
				return
			}
			if name != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, name)
			}
			if value != tt.wantValue {
				t.Errorf("expected value %f, got %f", tt.wantValue, value)
			}
		})
	}
}

// --- scrapeMetrics ---

func TestScrapeMetrics_UnreachableEndpoint(t *testing.T) {
	p := NewProvider(nil)
	metrics, err := p.scrapeMetrics("http://localhost:59997/metrics")
	// Unreachable → zero metrics, no error (graceful degradation)
	if err != nil {
		t.Fatalf("expected nil error for unreachable endpoint, got: %v", err)
	}
	if metrics == nil {
		t.Fatal("expected non-nil metrics even for unreachable endpoint")
	}
}

// --- DiscoverModels ---

func TestProvider_DiscoverModels_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	p := NewProvider(nil)

	models, err := p.DiscoverModels(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverModels failed: %v", err)
	}
	if len(models) != 0 {
		t.Errorf("expected 0 models in empty dir, got %d", len(models))
	}
}

func TestProvider_DiscoverModels_NonExistentDir(t *testing.T) {
	p := NewProvider(nil)
	_, err := p.DiscoverModels("/nonexistent/path/xyz")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestProvider_DiscoverModels_FindsModelDirs(t *testing.T) {
	tmpDir := t.TempDir()
	p := NewProvider(nil)

	// Create a "model" directory with a safetensors file
	modelDir := filepath.Join(tmpDir, "my-llama")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modelDir, "model.safetensors"), []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a directory without model files (should be ignored)
	emptyDir := filepath.Join(tmpDir, "empty-dir")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file in tmpDir root (should not be included — not a dir)
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}

	models, err := p.DiscoverModels(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverModels failed: %v", err)
	}
	if len(models) != 1 {
		t.Errorf("expected 1 model directory, got %d: %v", len(models), models)
	}
	if models[0] != "my-llama" {
		t.Errorf("expected model name 'my-llama', got %s", models[0])
	}
}

func TestProvider_DiscoverModels_MultipleFormats(t *testing.T) {
	tmpDir := t.TempDir()
	p := NewProvider(nil)

	formats := []struct {
		dir  string
		file string
	}{
		{"model-safetensors", "model.safetensors"},
		{"model-bin", "pytorch_model.bin"},
		{"model-pt", "model.pt"},
		{"model-config-only", "config.json"},
	}

	for _, f := range formats {
		dir := filepath.Join(tmpDir, f.dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, f.file), []byte("fake"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	models, err := p.DiscoverModels(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverModels failed: %v", err)
	}
	if len(models) != 4 {
		t.Errorf("expected 4 model dirs, got %d: %v", len(models), models)
	}
}

// --- ServiceProvider delegation ---

func TestServiceProvider_Create_Delegates(t *testing.T) {
	sp := NewServiceProvider(nil)
	svc, err := sp.Create(context.Background(), "model-abc", service.ResourceClassSmall, 1, false)
	if err != nil {
		t.Fatalf("ServiceProvider.Create failed: %v", err)
	}
	if svc.ModelID != "model-abc" {
		t.Errorf("expected ModelID 'model-abc', got %s", svc.ModelID)
	}
}

func TestServiceProvider_Scale_ReturnsNotImplemented(t *testing.T) {
	sp := NewServiceProvider(nil)
	err := sp.Scale(context.Background(), "svc-x", 3)
	if err == nil {
		t.Error("expected error for Scale (not implemented)")
	}
}

func TestServiceProvider_Stop_NonExistent(t *testing.T) {
	sp := NewServiceProvider(nil)
	err := sp.Stop(context.Background(), "nonexistent", true)
	if err == nil {
		t.Error("expected error for stopping non-existent service")
	}
}

func TestServiceProvider_GetRecommendation(t *testing.T) {
	sp := NewServiceProvider(nil)
	rec, err := sp.GetRecommendation(context.Background(), "llama3-70b", "")
	if err != nil {
		t.Fatalf("GetRecommendation failed: %v", err)
	}
	if rec.ResourceClass != service.ResourceClassLarge {
		t.Errorf("expected ResourceClassLarge, got %s", rec.ResourceClass)
	}
}

func TestServiceProvider_IsRunning_NoProcess(t *testing.T) {
	sp := NewServiceProvider(nil)
	if sp.IsRunning(context.Background(), "nonexistent") {
		t.Error("expected IsRunning=false for unknown service")
	}
}

func TestServiceProvider_SetGPUDevices_NotFound(t *testing.T) {
	sp := NewServiceProvider(nil)
	err := sp.SetGPUDevices("nonexistent", []int{0})
	if err == nil {
		t.Error("expected error for SetGPUDevices on unknown service")
	}
}

func TestServiceProvider_GetServiceInfo_NotFound(t *testing.T) {
	sp := NewServiceProvider(nil)
	_, err := sp.GetServiceInfo("nonexistent")
	if err == nil {
		t.Error("expected error for GetServiceInfo on unknown service")
	}
}

// --- Concurrent access ---

func TestProvider_ConcurrentCreate_NoRace(t *testing.T) {
	p := NewProvider(nil)
	const goroutines = 20
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := p.Create(context.Background(), "model-concurrent", service.ResourceClassSmall, 1, false)
			if err != nil {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("unexpected error in concurrent Create: %v", err)
	}
}

func TestProvider_ConcurrentGetServiceInfo_NoRace(t *testing.T) {
	p := NewProvider(nil)
	p.mu.Lock()
	for i := 0; i < 5; i++ {
		id := strings.Repeat("svc-", 1) + strings.Repeat("x", i+1)
		p.services[id] = &ServiceInfo{
			ServiceID: id,
			Port:      8000 + i,
			StartedAt: time.Now(),
		}
	}
	p.mu.Unlock()

	const goroutines = 20
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := "svc-" + strings.Repeat("x", (n%5)+1)
			_, _ = p.GetServiceInfo(id)
		}(i)
	}
	wg.Wait()
}

func TestProvider_ConcurrentSetGPUDevices_NoRace(t *testing.T) {
	p := NewProvider(nil)
	p.mu.Lock()
	p.services["svc-gpu"] = &ServiceInfo{ServiceID: "svc-gpu"}
	p.mu.Unlock()

	const goroutines = 10
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = p.SetGPUDevices("svc-gpu", []int{n % 4})
		}(i)
	}
	wg.Wait()
}
