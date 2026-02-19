package provider

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/provider/vllm"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
)

// MultiEngineProvider implements a service provider that supports multiple inference engines
type MultiEngineProvider struct {
	modelStore    model.ModelStore
	vllmProvider  *vllm.ServiceProvider
	processes     map[string]*exec.Cmd
	serviceInfo   map[string]*ServiceRuntimeInfo
	mu            sync.RWMutex
}

// ServiceRuntimeInfo holds runtime information for a service
type ServiceRuntimeInfo struct {
	ServiceID string
	ModelID   string
	ModelType model.ModelType
	Port      int
	UseGPU    bool
	ProcessID string
	Endpoint  string
}

var (
	// Script paths
	scriptDir string
)

func init() {
	// Get the directory of the current file
	_, filename, _, _ := runtime.Caller(0)
	pkgDir := filepath.Dir(filename)
	projectRoot := filepath.Dir(filepath.Dir(pkgDir))
	scriptDir = filepath.Join(projectRoot, "scripts")
}

// NewMultiEngineProvider creates a new multi-engine service provider
func NewMultiEngineProvider(modelStore model.ModelStore) *MultiEngineProvider {
	return &MultiEngineProvider{
		modelStore:   modelStore,
		vllmProvider: vllm.NewServiceProvider(modelStore),
		processes:    make(map[string]*exec.Cmd),
		serviceInfo:  make(map[string]*ServiceRuntimeInfo),
	}
}

// Create creates a new service using the appropriate engine
func (p *MultiEngineProvider) Create(ctx context.Context, modelID string, resourceClass service.ResourceClass, replicas int, persistent bool) (*service.ModelService, error) {
	// Get model info to determine which engine to use
	m, err := p.modelStore.Get(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("model not found: %s", modelID)
	}

	// Select engine based on model type
	switch m.Type {
	case model.ModelTypeLLM, model.ModelTypeVLM:
		// Use vLLM for LLM and VLM models
		return p.vllmProvider.Create(ctx, modelID, resourceClass, replicas, persistent)
	
	case model.ModelTypeASR:
		// TODO: Use ASR provider
		return nil, fmt.Errorf("ASR provider not yet implemented")
	
	case model.ModelTypeTTS:
		// TODO: Use TTS provider
		return nil, fmt.Errorf("TTS provider not yet implemented")
	
	default:
		// Default to vLLM
		return p.vllmProvider.Create(ctx, modelID, resourceClass, replicas, persistent)
	}
}

// Start starts the service using the appropriate engine.
// It tries each registered provider in order and returns the first success.
// All provider errors are collected and joined so callers get a full picture
// of what was attempted.
func (p *MultiEngineProvider) Start(ctx context.Context, serviceID string) error {
	var errs []string

	// vLLM — primary provider for LLM/VLM workloads.
	if err := p.vllmProvider.Start(ctx, serviceID); err == nil {
		return nil
	} else {
		errs = append(errs, fmt.Sprintf("vllm: %v", err))
	}

	// Additional providers can be inserted here as they are implemented.
	// Example:
	//   if err := p.asrProvider.Start(ctx, serviceID); err == nil { return nil }
	//   errs = append(errs, fmt.Sprintf("asr: %v", err))

	return fmt.Errorf("no provider could start service %s: [%s]", serviceID, strings.Join(errs, "; "))
}

// Stop stops the service using the appropriate engine.
// It tries each registered provider in order and returns the first success.
func (p *MultiEngineProvider) Stop(ctx context.Context, serviceID string, force bool) error {
	var errs []string

	if err := p.vllmProvider.Stop(ctx, serviceID, force); err == nil {
		return nil
	} else {
		errs = append(errs, fmt.Sprintf("vllm: %v", err))
	}

	// Additional providers can be inserted here as they are implemented.

	return fmt.Errorf("no provider could stop service %s: [%s]", serviceID, strings.Join(errs, "; "))
}

// Scale scales the service using the appropriate engine.
// It tries each registered provider in order and returns the first success.
func (p *MultiEngineProvider) Scale(ctx context.Context, serviceID string, replicas int) error {
	var errs []string

	if err := p.vllmProvider.Scale(ctx, serviceID, replicas); err == nil {
		return nil
	} else {
		errs = append(errs, fmt.Sprintf("vllm: %v", err))
	}

	// Additional providers can be inserted here as they are implemented.

	return fmt.Errorf("no provider could scale service %s: [%s]", serviceID, strings.Join(errs, "; "))
}

// GetMetrics returns service metrics from the appropriate engine.
// It tries each registered provider in order and returns the first success.
func (p *MultiEngineProvider) GetMetrics(ctx context.Context, serviceID string) (*service.ServiceMetrics, error) {
	var errs []string

	if metrics, err := p.vllmProvider.GetMetrics(ctx, serviceID); err == nil {
		return metrics, nil
	} else {
		errs = append(errs, fmt.Sprintf("vllm: %v", err))
	}

	// Additional providers can be inserted here as they are implemented.

	return nil, fmt.Errorf("no provider could get metrics for service %s: [%s]", serviceID, strings.Join(errs, "; "))
}

// GetRecommendation provides resource recommendations including engine and device type
func (p *MultiEngineProvider) GetRecommendation(ctx context.Context, modelID string, hint string) (*service.Recommendation, error) {
	// Get model info
	m, err := p.modelStore.Get(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("model not found: %s", modelID)
	}

	// Select engine and device based on model type
	switch m.Type {
	case model.ModelTypeLLM, model.ModelTypeVLM:
		rec, err := p.vllmProvider.GetRecommendation(ctx, modelID, hint)
		if err != nil {
			// Fallback recommendation for LLM
			rec = &service.Recommendation{
				ResourceClass:      service.ResourceClassLarge,
				Replicas:           1,
				ExpectedThroughput: 50.0,
			}
		}
		rec.EngineType = "vllm"
		rec.DeviceType = "gpu"
		rec.Reason = fmt.Sprintf("%s model '%s' recommended for GPU acceleration with vLLM for high-performance inference", m.Type, m.Name)
		return rec, nil
	
	case model.ModelTypeASR:
		return &service.Recommendation{
			ResourceClass:      service.ResourceClassSmall,
			Replicas:           1,
			ExpectedThroughput: 10.0,
			EngineType:         "whisper",
			DeviceType:         "cpu",
			Reason:             fmt.Sprintf("ASR model '%s' runs efficiently on CPU with Whisper engine", m.Name),
		}, nil
	
	case model.ModelTypeTTS:
		return &service.Recommendation{
			ResourceClass:      service.ResourceClassSmall,
			Replicas:           1,
			ExpectedThroughput: 5.0,
			EngineType:         "tts",
			DeviceType:         "cpu",
			Reason:             fmt.Sprintf("TTS model '%s' runs efficiently on CPU with dedicated TTS engine", m.Name),
		}, nil
	
	case model.ModelTypeEmbedding:
		return &service.Recommendation{
			ResourceClass:      service.ResourceClassMedium,
			Replicas:           1,
			ExpectedThroughput: 100.0,
			EngineType:         "transformers",
			DeviceType:         "cpu",
			Reason:             fmt.Sprintf("Embedding model '%s' recommended for CPU inference with Transformers", m.Name),
		}, nil
	
	default:
		// Default to vLLM on GPU for unknown types
		return &service.Recommendation{
			ResourceClass:      service.ResourceClassMedium,
			Replicas:           1,
			ExpectedThroughput: 30.0,
			EngineType:         "vllm",
			DeviceType:         "gpu",
			Reason:             fmt.Sprintf("Model '%s' of type '%s' defaults to vLLM on GPU", m.Name, m.Type),
		}, nil
	}
}

// Ensure MultiEngineProvider implements the interface
var _ service.ServiceProvider = (*MultiEngineProvider)(nil)
