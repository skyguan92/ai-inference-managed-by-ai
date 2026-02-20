package provider

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/docker"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/engine"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
)

// DockerEngineProvider implements EngineProvider using Docker containers
type DockerEngineProvider struct {
	client     *docker.SimpleClient
	containers map[string]string // engine name -> container ID
}

// NewDockerEngineProvider creates a new Docker-based engine provider
func NewDockerEngineProvider() (*DockerEngineProvider, error) {
	if err := docker.CheckDocker(); err != nil {
		return nil, fmt.Errorf("docker check failed: %w", err)
	}

	return &DockerEngineProvider{
		client:     docker.NewSimpleClient(),
		containers: make(map[string]string),
	}, nil
}

// Install pulls the required Docker image for an engine
func (p *DockerEngineProvider) Install(ctx context.Context, name string, version string) (*engine.InstallResult, error) {
	image := p.getImageForEngine(name, version)

	slog.Info("pulling Docker image", "engine", name, "image", image)
	if err := p.client.PullImage(ctx, image); err != nil {
		// Image might not exist or network issue, try to continue
		slog.Warn("failed to pull image, will try existing image or build locally", "image", image, "error", err)
	}

	return &engine.InstallResult{
		Success: true,
		Path:    image,
	}, nil
}

// Start starts the engine in a Docker container
func (p *DockerEngineProvider) Start(ctx context.Context, name string, config map[string]any) (*engine.StartResult, error) {
	// Check if already running
	if containerID, exists := p.containers[name]; exists {
		status, err := p.client.GetContainerStatus(ctx, containerID)
		if err == nil && status == "running" {
			return &engine.StartResult{
				ProcessID: containerID,
				Status:    engine.EngineStatusRunning,
			}, nil
		}
	}

	// Get configuration
	image := p.getImageForEngine(name, "")
	port := p.getDefaultPort(name)
	useGPU := p.shouldUseGPU(config, name)

	// Extract model path from config
	var modelPath string
	if mp, ok := config["model_path"].(string); ok && mp != "" {
		modelPath = mp
	}

	// Build container options
	opts := docker.ContainerOptions{
		Ports: map[string]string{
			strconv.Itoa(port): strconv.Itoa(port),
		},
		Labels: map[string]string{
			"aima.engine":  name,
			"aima.managed": "true",
		},
		GPU: useGPU,
	}

	// Add model volume if provided
	if modelPath != "" {
		opts.Volumes = map[string]string{
			modelPath: "/models",
		}
	}

	// Build command based on engine type
	opts.Cmd = p.buildCommand(name, config, port)

	// Create unique container name
	containerName := fmt.Sprintf("aima-%s-%d", name, time.Now().Unix())

	slog.Info("starting engine in Docker container", "engine", name, "image", image, "port", port, "gpu", useGPU)

	containerID, err := p.client.CreateAndStartContainer(ctx, containerName, image, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Store container ID
	p.containers[name] = containerID

	slog.Info("container started, waiting for service to be ready", "container_id", containerID[:12])

	// Wait for service to be ready
	if err := p.waitForReady(ctx, containerID, name, port); err != nil {
		slog.Warn("service may not be fully ready", "error", err)
	}

	return &engine.StartResult{
		ProcessID: containerID,
		Status:    engine.EngineStatusRunning,
	}, nil
}

// Stop stops the engine container
func (p *DockerEngineProvider) Stop(ctx context.Context, name string, force bool, timeout int) (*engine.StopResult, error) {
	containerID, exists := p.containers[name]
	if !exists {
		// Try to find container by label
		containers, err := p.client.ListContainers(ctx, map[string]string{"aima.engine": name})
		if err != nil || len(containers) == 0 {
			return &engine.StopResult{Success: true}, nil
		}
		containerID = containers[0]
	}

	slog.Info("stopping engine", "engine", name, "container_id", containerID[:12])

	if force {
		// For force stop, we kill and remove immediately
		timeout = 0
	}

	if err := p.client.StopContainer(ctx, containerID, timeout); err != nil {
		return nil, fmt.Errorf("failed to stop container: %w", err)
	}

	delete(p.containers, name)
	slog.Info("engine stopped", "engine", name)

	return &engine.StopResult{Success: true}, nil
}

// GetFeatures returns engine capabilities
func (p *DockerEngineProvider) GetFeatures(ctx context.Context, name string) (*engine.EngineFeatures, error) {
	switch name {
	case "vllm":
		return &engine.EngineFeatures{
			SupportsStreaming:    true,
			SupportsBatch:        true,
			SupportsMultimodal:   false,
			SupportsTools:        true,
			SupportsEmbedding:    false,
			MaxConcurrent:        100,
			MaxContextLength:     128000,
			MaxBatchSize:         256,
			SupportsGPULayers:    true,
			SupportsQuantization: true,
		}, nil
	case "whisper", "asr":
		return &engine.EngineFeatures{
			SupportsStreaming:    false,
			SupportsBatch:        false,
			SupportsMultimodal:   false,
			SupportsTools:        false,
			SupportsEmbedding:    false,
			MaxConcurrent:        10,
			MaxContextLength:     0,
			MaxBatchSize:         1,
			SupportsGPULayers:    false,
			SupportsQuantization: false,
		}, nil
	case "tts":
		return &engine.EngineFeatures{
			SupportsStreaming:    false,
			SupportsBatch:        false,
			SupportsMultimodal:   false,
			SupportsTools:        false,
			SupportsEmbedding:    false,
			MaxConcurrent:        10,
			MaxContextLength:     0,
			MaxBatchSize:         1,
			SupportsGPULayers:    false,
			SupportsQuantization: false,
		}, nil
	default:
		return &engine.EngineFeatures{
			SupportsStreaming:    true,
			SupportsBatch:        false,
			SupportsMultimodal:   false,
			SupportsTools:        false,
			SupportsEmbedding:    false,
			MaxConcurrent:        10,
			MaxContextLength:     8192,
			MaxBatchSize:         1,
			SupportsGPULayers:    false,
			SupportsQuantization: false,
		}, nil
	}
}

// Helper methods

func (p *DockerEngineProvider) getImageForEngine(name, version string) string {
	if version == "" {
		version = "latest"
	}

	switch name {
	case "vllm":
		return fmt.Sprintf("vllm/vllm-openai:%s", version)
	case "whisper", "asr":
		// Use a generic ASR image or funasr
		return fmt.Sprintf("registry.cn-hangzhou.aliyuncs.com/funasr/funasr:funasr-runtime-sdk-cpu-0.4.5")
	case "tts":
		// Use Coqui TTS or similar
		return fmt.Sprintf("ghcr.io/coqui-ai/tts:%s", version)
	default:
		return fmt.Sprintf("aima/%s:%s", name, version)
	}
}

func (p *DockerEngineProvider) getDefaultPort(name string) int {
	switch name {
	case "vllm":
		return 8000
	case "whisper", "asr":
		return 8001
	case "tts":
		return 8002
	default:
		return 8080
	}
}

func (p *DockerEngineProvider) shouldUseGPU(config map[string]any, engineName string) bool {
	// Check explicit config
	if gpu, ok := config["gpu"].(bool); ok {
		return gpu
	}
	if device, ok := config["device"].(string); ok {
		return device == "gpu"
	}

	// Default: LLM uses GPU, others use CPU
	switch engineName {
	case "vllm":
		return true
	case "whisper", "asr", "tts":
		return false
	default:
		return false
	}
}

func (p *DockerEngineProvider) buildCommand(name string, config map[string]any, port int) []string {
	switch name {
	case "vllm":
		cmd := []string{
			"--model", "/models",
			"--port", strconv.Itoa(port),
		}

		// GPU memory utilization
		gpuUtil := 0.9
		if v, ok := config["gpu_memory_utilization"].(float64); ok {
			gpuUtil = v
		}
		cmd = append(cmd, "--gpu-memory-utilization", fmt.Sprintf("%.2f", gpuUtil))

		// Tensor parallel size
		if tp, ok := config["tensor_parallel_size"].(int); ok && tp > 1 {
			cmd = append(cmd, "--tensor-parallel-size", strconv.Itoa(tp))
		}

		// Quantization
		if q, ok := config["quantization"].(string); ok && q != "" {
			cmd = append(cmd, "--quantization", q)
		}

		return cmd

	case "whisper", "asr":
		// FunASR or similar ASR server
		return []string{
			"--model-dir", "/models",
			"--port", strconv.Itoa(port),
		}

	case "tts":
		// TTS server
		return []string{
			"--model", "/models",
			"--port", strconv.Itoa(port),
		}

	default:
		return nil
	}
}

func (p *DockerEngineProvider) waitForReady(ctx context.Context, containerID, name string, port int) error {
	// Poll for container status
	for i := 0; i < 30; i++ {
		status, err := p.client.GetContainerStatus(ctx, containerID)
		if err != nil {
			return fmt.Errorf("check container status: %w", err)
		}

		if status == "running" {
			// For simple check, we just verify container is running
			// In production, should check health endpoint
			if i > 5 {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	return fmt.Errorf("timeout waiting for container to be ready")
}

// Ensure DockerEngineProvider implements EngineProvider interface
var _ engine.EngineProvider = (*DockerEngineProvider)(nil)

// DockerServiceProvider implements ServiceProvider using Docker containers
type DockerServiceProvider struct {
	dockerProvider *DockerEngineProvider
	modelStore     model.ModelStore
	portCounter    int
}

// NewDockerServiceProvider creates a new Docker-based service provider
func NewDockerServiceProvider(modelStore model.ModelStore) (*DockerServiceProvider, error) {
	engineProvider, err := NewDockerEngineProvider()
	if err != nil {
		return nil, err
	}

	return NewDockerServiceProviderWithEngine(modelStore, engineProvider), nil
}

// NewDockerServiceProviderWithEngine creates a new Docker-based service provider with an existing engine provider
func NewDockerServiceProviderWithEngine(modelStore model.ModelStore, engineProvider *DockerEngineProvider) *DockerServiceProvider {
	return &DockerServiceProvider{
		dockerProvider: engineProvider,
		modelStore:     modelStore,
		portCounter:    8000,
	}
}

// Create creates a service configuration
func (p *DockerServiceProvider) Create(ctx context.Context, modelID string, resourceClass service.ResourceClass, replicas int, persistent bool) (*service.ModelService, error) {
	m, err := p.modelStore.Get(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("model not found: %s", modelID)
	}

	// Determine engine type from model type
	engineType := p.getEngineTypeForModel(m.Type)

	// Allocate port
	port := p.portCounter
	p.portCounter++

	return &service.ModelService{
		ID:            fmt.Sprintf("svc-%s-%d", engineType, time.Now().Unix()),
		Name:          fmt.Sprintf("%s-%s", engineType, m.Name),
		ModelID:       modelID,
		Status:        service.ServiceStatusCreating,
		Replicas:      replicas,
		ResourceClass: resourceClass,
		Endpoints:     []string{fmt.Sprintf("http://localhost:%d", port)},
		Config: map[string]any{
			"engine_type": engineType,
			"port":        port,
			"model_path":  m.Path,
		},
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}, nil
}

// Start starts the service in a Docker container
func (p *DockerServiceProvider) Start(ctx context.Context, serviceID string) error {
	// This is a simplified implementation
	// In production, should look up service from store and start with proper config
	return fmt.Errorf("use engine.start with model_path config instead")
}

// Stop stops the service
func (p *DockerServiceProvider) Stop(ctx context.Context, serviceID string, force bool) error {
	_, err := p.dockerProvider.Stop(ctx, serviceID, force, 30)
	return err
}

// Scale scales the service
func (p *DockerServiceProvider) Scale(ctx context.Context, serviceID string, replicas int) error {
	return fmt.Errorf("scaling not yet implemented for Docker services")
}

// GetMetrics returns service metrics
func (p *DockerServiceProvider) GetMetrics(ctx context.Context, serviceID string) (*service.ServiceMetrics, error) {
	return &service.ServiceMetrics{}, nil
}

// GetRecommendation provides resource recommendations
func (p *DockerServiceProvider) GetRecommendation(ctx context.Context, modelID string, hint string) (*service.Recommendation, error) {
	m, err := p.modelStore.Get(ctx, modelID)
	if err != nil {
		return nil, err
	}

	engineType := p.getEngineTypeForModel(m.Type)
	deviceType := "cpu"
	resourceClass := service.ResourceClassSmall

	if engineType == "vllm" {
		deviceType = "gpu"
		resourceClass = service.ResourceClassLarge
	}

	return &service.Recommendation{
		ResourceClass:      resourceClass,
		Replicas:           1,
		ExpectedThroughput: 50.0,
		EngineType:         engineType,
		DeviceType:         deviceType,
		Reason:             fmt.Sprintf("Model type '%s' recommended to run on %s with %s engine", m.Type, deviceType, engineType),
	}, nil
}

func (p *DockerServiceProvider) getEngineTypeForModel(modelType model.ModelType) string {
	switch modelType {
	case model.ModelTypeLLM, model.ModelTypeVLM:
		return "vllm"
	case model.ModelTypeASR:
		return "whisper"
	case model.ModelTypeTTS:
		return "tts"
	default:
		return "vllm"
	}
}

// IsRunning checks if the service container is actually running
func (p *DockerServiceProvider) IsRunning(ctx context.Context, serviceID string) bool {
	// For DockerServiceProvider, check if the service is running via the provider
	// This is a simplified implementation
	containerID, exists := p.dockerProvider.containers[serviceID]
	if !exists || containerID == "" {
		return false
	}

	status, err := p.dockerProvider.client.GetContainerStatus(ctx, containerID)
	if err != nil {
		return false
	}
	return status == "running"
}

// Ensure DockerServiceProvider implements ServiceProvider interface
var _ service.ServiceProvider = (*DockerServiceProvider)(nil)
