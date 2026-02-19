package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/docker"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/engine"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
)

// ResourceLimits defines resource constraints for containers
type ResourceLimits struct {
	Memory    string  // e.g., "4g", "512m"
	CPU       float64 // e.g., 2.0 for 2 cores
	GPU       bool
	GPUMemory string // e.g., "10g" for unified memory systems
}

// StartupConfig defines startup behavior
type StartupConfig struct {
	MaxRetries     int
	RetryInterval  time.Duration
	StartupTimeout time.Duration
	HealthCheckURL string
}

// HybridEngineProvider supports both Docker and Native process modes
type HybridEngineProvider struct {
	dockerClient    *docker.SimpleClient
	containers      map[string]string // name -> container ID
	nativeProcesses map[string]*exec.Cmd
	serviceInfo     map[string]*ServiceInfo
	modelStore      model.ModelStore

	// Resource management
	resourceLimits map[string]ResourceLimits
	startupConfigs map[string]StartupConfig

	// Concurrency protection
	mu sync.RWMutex
}

// ServiceInfo holds runtime information
type ServiceInfo struct {
	ServiceID string
	ModelID   string
	Engine    string
	Port      int
	UseGPU    bool
	ProcessID string
	Endpoint  string
}

// NewHybridEngineProvider creates a new hybrid engine provider
func NewHybridEngineProvider(modelStore model.ModelStore) *HybridEngineProvider {
	return &HybridEngineProvider{
		dockerClient:    docker.NewSimpleClient(),
		containers:      make(map[string]string),
		nativeProcesses: make(map[string]*exec.Cmd),
		serviceInfo:     make(map[string]*ServiceInfo),
		modelStore:      modelStore,
		resourceLimits:  getDefaultResourceLimits(),
		startupConfigs:  getDefaultStartupConfigs(),
	}
}

// getDefaultResourceLimits returns default resource limits for each engine type
// Can be overridden via environment variables: AIMA_{ENGINE}_MEMORY, AIMA_{ENGINE}_CPU, AIMA_{ENGINE}_GPU
func getDefaultResourceLimits() map[string]ResourceLimits {
	limits := map[string]ResourceLimits{
		"vllm": {
			Memory:    "0", // 0 means no limit, vLLM manages its own memory
			CPU:       0,   // 0 means no limit
			GPU:       true,
			GPUMemory: "80g", // Limit to 80GB on unified memory systems
		},
		"whisper": {
			Memory:    "4g",
			CPU:       2.0,
			GPU:       false, // Force CPU for ASR to save GPU memory
			GPUMemory: "0",
		},
		"asr": {
			Memory:    "4g",
			CPU:       2.0,
			GPU:       false, // Force CPU for ASR
			GPUMemory: "0",
		},
		"tts": {
			Memory:    "4g",
			CPU:       2.0,
			GPU:       false, // Force CPU for TTS
			GPUMemory: "0",
		},
	}

	// Override with environment variables
	for engine := range limits {
		if mem := os.Getenv(fmt.Sprintf("AIMA_%s_MEMORY", strings.ToUpper(engine))); mem != "" {
			limits[engine] = ResourceLimits{
				Memory:    mem,
				CPU:       limits[engine].CPU,
				GPU:       limits[engine].GPU,
				GPUMemory: limits[engine].GPUMemory,
			}
		}
		if cpuStr := os.Getenv(fmt.Sprintf("AIMA_%s_CPU", strings.ToUpper(engine))); cpuStr != "" {
			if cpu, err := strconv.ParseFloat(cpuStr, 64); err == nil {
				l := limits[engine]
				l.CPU = cpu
				limits[engine] = l
			}
		}
		if gpuStr := os.Getenv(fmt.Sprintf("AIMA_%s_GPU", strings.ToUpper(engine))); gpuStr != "" {
			l := limits[engine]
			l.GPU = gpuStr == "true" || gpuStr == "1"
			limits[engine] = l
		}
	}

	return limits
}

// getDefaultStartupConfigs returns default startup configurations
func getDefaultStartupConfigs() map[string]StartupConfig {
	return map[string]StartupConfig{
		"vllm": {
			MaxRetries:     3,
			RetryInterval:  10 * time.Second,
			StartupTimeout: 300 * time.Second, // 5 minutes for large models
			HealthCheckURL: "/health",
		},
		"whisper": {
			MaxRetries:     3,
			RetryInterval:  5 * time.Second,
			StartupTimeout: 60 * time.Second,
			HealthCheckURL: "/",
		},
		"asr": {
			MaxRetries:     3,
			RetryInterval:  5 * time.Second,
			StartupTimeout: 60 * time.Second,
			HealthCheckURL: "/",
		},
		"tts": {
			MaxRetries:     3,
			RetryInterval:  5 * time.Second,
			StartupTimeout: 60 * time.Second,
			HealthCheckURL: "/health",
		},
	}
}

// Install tries to prepare the engine (check Docker image or native binary)
func (p *HybridEngineProvider) Install(ctx context.Context, name string, version string) (*engine.InstallResult, error) {
	// Check if Docker is available
	if err := docker.CheckDocker(); err == nil {
		// Get list of candidate images
		candidates := p.getDockerImages(name, version)
		fmt.Printf("Checking Docker images for %s...\n", name)

		// Check all candidates for local availability
		for _, image := range candidates {
			if p.imageExists(image) {
				fmt.Printf("✓ Docker image already exists: %s\n", image)
				return &engine.InstallResult{
					Success: true,
					Path:    image,
				}, nil
			}
		}

		// No local image found, try to pull the first candidate
		image := candidates[0]
		fmt.Printf("⚠ No local image found. Pulling: %s...\n", image)
		pullCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		if err := p.dockerClient.PullImage(pullCtx, image); err != nil {
			fmt.Printf("⚠ Failed to pull image: %v\n", err)
			fmt.Println("Will try to use existing image or native mode.")
		} else {
			fmt.Printf("✓ Docker image pulled successfully\n")
			return &engine.InstallResult{
				Success: true,
				Path:    image,
			}, nil
		}
	} else {
		fmt.Printf("Docker not available: %v\n", err)
	}

	// Fall back to native mode
	if p.checkNativeBinary(name) {
		fmt.Printf("✓ Native binary available for %s\n", name)
		return &engine.InstallResult{
			Success: true,
			Path:    name,
		}, nil
	}

	// Return success anyway - will try to start with available resources
	fmt.Printf("⚠ Neither Docker image nor native binary available for %s\n", name)
	fmt.Println("Will attempt to start with best-effort mode.")
	return &engine.InstallResult{
		Success: true,
		Path:    name,
	}, nil
}

// Start starts the engine service for a model
func (p *HybridEngineProvider) Start(ctx context.Context, name string, config map[string]any) (*engine.StartResult, error) {
	// Get startup config
	startupCfg := p.startupConfigs[name]
	if startupCfg.MaxRetries == 0 {
		startupCfg.MaxRetries = 3
	}

	// Get model info if provided
	var modelInfo *model.Model
	var modelPath string

	if modelID, ok := config["model_id"].(string); ok && modelID != "" {
		if m, err := p.modelStore.Get(ctx, modelID); err == nil {
			modelInfo = m
			modelPath = m.Path
		}
	}

	// Fall back to config path
	if modelPath == "" {
		modelPath, _ = config["model_path"].(string)
	}

	// Determine engine type from name or model
	engineType := name
	if modelInfo != nil {
		engineType = p.getEngineTypeForModel(modelInfo.Type)
	}

	// Get resource limits
	limits := p.resourceLimits[engineType]

	// Override with config if provided
	if device, ok := config["device"].(string); ok {
		limits.GPU = device == "gpu"
	}
	if gpu, ok := config["gpu"].(bool); ok {
		limits.GPU = gpu
	}

	useGPU := limits.GPU
	port := p.getDefaultPort(engineType)

	// Try Docker first if available
	if docker.CheckDocker() == nil {
		var lastErr error
		for attempt := 1; attempt <= startupCfg.MaxRetries; attempt++ {
			if attempt > 1 {
				fmt.Printf("Retrying %s start (attempt %d/%d)...\n", name, attempt, startupCfg.MaxRetries)
				time.Sleep(startupCfg.RetryInterval)
			}

			result, err := p.startDockerWithRetry(ctx, engineType, modelPath, port, useGPU, config, limits)
			if err == nil {
				// Wait for health check
				if err := p.waitForHealth(ctx, result.ProcessID, port, startupCfg.HealthCheckURL, startupCfg.StartupTimeout); err != nil {
					fmt.Printf("⚠ Health check failed: %v\n", err)
					p.Stop(ctx, engineType, true, 10)
					lastErr = err
					continue
				}
				return result, nil
			}
			lastErr = err
			fmt.Printf("⚠ Docker start failed (attempt %d): %v\n", attempt, err)
		}
		fmt.Printf("Docker start failed after %d attempts: %v, trying native mode...\n", startupCfg.MaxRetries, lastErr)
	}

	// Fall back to native mode
	return p.startNative(ctx, engineType, modelPath, port, useGPU, config)
}

// startDockerWithRetry starts engine in Docker container with resource limits
func (p *HybridEngineProvider) startDockerWithRetry(ctx context.Context, engineType, modelPath string, port int, useGPU bool, config map[string]any, limits ResourceLimits) (*engine.StartResult, error) {
	image := p.getDockerImage(engineType, "")

	// Check if image exists
	if !p.imageExists(image) {
		return nil, fmt.Errorf("Docker image not found: %s", image)
	}

	opts := docker.ContainerOptions{
		Ports: map[string]string{
			strconv.Itoa(port): strconv.Itoa(port),
		},
		Labels: map[string]string{
			"aima.engine":  engineType,
			"aima.managed": "true",
		},
		GPU: useGPU,
	}

	// Add resource limits
	if limits.Memory != "" && limits.Memory != "0" {
		opts.Memory = limits.Memory
	}
	if limits.CPU > 0 {
		opts.CPU = fmt.Sprintf("%.1f", limits.CPU)
	}

	// Add model volume if provided
	if modelPath != "" {
		// Validate model path to prevent command injection
		if err := validateModelPath(modelPath); err != nil {
			return nil, fmt.Errorf("invalid model path: %w", err)
		}
		// Determine mount path based on image type
		mountPath := "/models"
		if strings.Contains(image, "qujing-glm-asr-nano") {
			mountPath = "/model" // ASR image expects /model
		}
		opts.Volumes = map[string]string{
			modelPath: mountPath,
		}
	}

	// Build command based on engine type
	opts.Cmd = p.buildDockerCommand(engineType, image, config, port)

	containerName := fmt.Sprintf("aima-%s-%d", engineType, time.Now().Unix())

	fmt.Printf("\n🐳 Starting %s in Docker container...\n", engineType)
	fmt.Printf("   Image: %s\n", image)
	fmt.Printf("   Model: %s\n", modelPath)
	fmt.Printf("   Port: %d\n", port)
	fmt.Printf("   GPU: %v\n", useGPU)
	if limits.Memory != "" {
		fmt.Printf("   Memory Limit: %s\n", limits.Memory)
	}
	if limits.CPU > 0 {
		fmt.Printf("   CPU Limit: %.1f cores\n", limits.CPU)
	}

	containerID, err := p.dockerClient.CreateAndStartContainer(ctx, containerName, image, opts)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	p.containers[engineType] = containerID
	p.mu.Unlock()

	fmt.Printf("✓ Container started: %s\n", containerID[:12])
	fmt.Printf("   Endpoint: http://localhost:%d\n", port)

	return &engine.StartResult{
		ProcessID: containerID,
		Status:    engine.EngineStatusRunning,
	}, nil
}

// waitForHealth waits for service to become healthy
func (p *HybridEngineProvider) waitForHealth(ctx context.Context, containerID string, port int, healthPath string, timeout time.Duration) error {
	if healthPath == "" {
		healthPath = "/health"
	}

	endpoint := fmt.Sprintf("http://localhost:%d%s", port, healthPath)
	fmt.Printf("⏳ Waiting for health check: %s (timeout: %v)\n", endpoint, timeout)

	deadline := time.Now().Add(timeout)
	checkInterval := 2 * time.Second

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Try HTTP health check
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(endpoint)
		if err == nil {
			func() {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					fmt.Printf("✓ Health check passed!\n")
				}
			}()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		// Check if container is still running
		status, err := p.dockerClient.GetContainerStatus(ctx, containerID)
		if err != nil || status != "running" {
			return fmt.Errorf("container not running (status: %s)", status)
		}

		time.Sleep(checkInterval)
	}

	return fmt.Errorf("health check timeout after %v", timeout)
}

// startNative starts engine as native process
func (p *HybridEngineProvider) startNative(ctx context.Context, engineType, modelPath string, port int, useGPU bool, config map[string]any) (*engine.StartResult, error) {
	fmt.Printf("\n⚙️  Starting %s as native process...\n", engineType)

	// Check if vllm is available in PATH
	if _, err := exec.LookPath("vllm"); err != nil {
		return nil, fmt.Errorf("vllm not found in PATH and Docker not available")
	}

	// Build vllm command
	args := []string{"serve"}
	if modelPath != "" {
		args = append(args, modelPath)
	} else {
		args = append(args, "--model", config["model"].(string))
	}

	args = append(args, "--port", strconv.Itoa(port))

	if gpuUtil, ok := config["gpu_memory_utilization"].(float64); ok {
		args = append(args, "--gpu-memory-utilization", fmt.Sprintf("%.2f", gpuUtil))
	} else {
		args = append(args, "--gpu-memory-utilization", "0.9")
	}

	cmd := exec.CommandContext(ctx, "vllm", args...)

	// Set environment
	if !useGPU {
		cmd.Env = append(cmd.Env, "CUDA_VISIBLE_DEVICES=")
	}

	fmt.Printf("   Command: vllm %s\n", strings.Join(args, " "))

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start vllm: %w", err)
	}

	p.mu.Lock()
	p.nativeProcesses[engineType] = cmd
	p.mu.Unlock()

	// Start goroutine to wait for process and clean up
	go func() {
		if err := cmd.Wait(); err != nil {
			fmt.Printf("Process %s exited with error: %v\n", engineType, err)
		}
		p.mu.Lock()
		delete(p.nativeProcesses, engineType)
		p.mu.Unlock()
	}()

	fmt.Printf("✓ Process started (PID: %d)\n", cmd.Process.Pid)
	fmt.Printf("   Endpoint: http://localhost:%d\n", port)

	return &engine.StartResult{
		ProcessID: strconv.Itoa(cmd.Process.Pid),
		Status:    engine.EngineStatusRunning,
	}, nil
}

// Stop stops the engine
func (p *HybridEngineProvider) Stop(ctx context.Context, name string, force bool, timeout int) (*engine.StopResult, error) {
	// Try Docker first
	p.mu.RLock()
	containerID, exists := p.containers[name]
	p.mu.RUnlock()
	if exists {
		fmt.Printf("Stopping Docker container: %s...\n", containerID[:12])
		if err := p.dockerClient.StopContainer(ctx, containerID, timeout); err != nil {
			return nil, err
		}
		p.mu.Lock()
		delete(p.containers, name)
		p.mu.Unlock()
		return &engine.StopResult{Success: true}, nil
	}

	// Try native process
	p.mu.RLock()
	cmd, exists := p.nativeProcesses[name]
	p.mu.RUnlock()
	if exists {
		fmt.Printf("Stopping native process (PID: %d)...\n", cmd.Process.Pid)
		if force {
			cmd.Process.Kill()
		} else {
			cmd.Process.Signal(os.Interrupt)
			// Wait for graceful shutdown
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			select {
			case <-done:
			case <-time.After(time.Duration(timeout) * time.Second):
				cmd.Process.Kill()
			}
		}
		p.mu.Lock()
		delete(p.nativeProcesses, name)
		p.mu.Unlock()
		return &engine.StopResult{Success: true}, nil
	}

	return &engine.StopResult{Success: true}, nil
}

// GetFeatures returns engine capabilities
func (p *HybridEngineProvider) GetFeatures(ctx context.Context, name string) (*engine.EngineFeatures, error) {
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
			SupportsStreaming: false,
			MaxConcurrent:     10,
		}, nil
	case "tts":
		return &engine.EngineFeatures{
			SupportsStreaming: false,
			MaxConcurrent:     10,
		}, nil
	default:
		return &engine.EngineFeatures{
			SupportsStreaming: true,
			MaxConcurrent:     10,
		}, nil
	}
}

// Helper methods

func (p *HybridEngineProvider) imageExists(image string) bool {
	cmd := exec.Command("docker", "images", "-q", image)
	output, err := cmd.Output()
	return err == nil && len(output) > 0
}

func (p *HybridEngineProvider) checkNativeBinary(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func (p *HybridEngineProvider) getDockerImages(name, version string) []string {
	if version == "" {
		version = "latest"
	}

	switch name {
	case "vllm":
		// Priority: GB10 compatible custom image > official image
		return []string{
			"zhiwen-vllm:0128",         // GB10 compatible custom image (priority 1)
			"vllm/vllm-openai:v0.15.0", // Official image (fallback)
		}
	case "whisper", "asr":
		// Priority: local custom image > official image
		return []string{
			"qujing-glm-asr-nano:latest", // Local custom image (priority 1)
			"registry.cn-hangzhou.aliyuncs.com/funasr_repo/funasr:funasr-runtime-sdk-cpu-0.4.6", // Official (priority 2)
		}
	case "tts":
		// Priority: local custom image > official image
		return []string{
			"qujing-qwen3-tts:latest",     // Local custom image (priority 1)
			"ghcr.io/coqui-ai/tts:latest", // Official (priority 2)
		}
	default:
		return []string{fmt.Sprintf("%s:%s", name, version)}
	}
}

func (p *HybridEngineProvider) getDockerImage(name, version string) string {
	candidates := p.getDockerImages(name, version)

	// Try each candidate in order
	for _, image := range candidates {
		if p.imageExists(image) {
			fmt.Printf("✓ Using local image: %s\n", image)
			return image
		}
	}

	// Return first candidate (will trigger pull attempt)
	fmt.Printf("⚠ No local image found, will attempt to pull: %s\n", candidates[0])
	return candidates[0]
}

func (p *HybridEngineProvider) getDefaultPort(engineType string) int {
	switch engineType {
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

func (p *HybridEngineProvider) shouldUseGPU(config map[string]any, engineType string) bool {
	if device, ok := config["device"].(string); ok {
		return device == "gpu"
	}
	if gpu, ok := config["gpu"].(bool); ok {
		return gpu
	}
	// Default: vllm uses GPU, others use CPU
	return engineType == "vllm"
}

// validateModelPath checks if the model path is safe (no shell injection)
func validateModelPath(path string) error {
	// Reject paths with shell metacharacters
	dangerous := []string{";", "&", "|", "`", "$", "(", ")", "<", ">", "\\", "'", "\"", "\n", "\r"}
	for _, char := range dangerous {
		if strings.Contains(path, char) {
			return fmt.Errorf("path contains invalid character: %q", char)
		}
	}
	// Reject relative path traversals
	if strings.Contains(path, "..") {
		return fmt.Errorf("path contains directory traversal")
	}
	// Must be absolute path
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("path must be absolute")
	}
	return nil
}

func (p *HybridEngineProvider) getEngineTypeForModel(modelType model.ModelType) string {
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

func (p *HybridEngineProvider) buildDockerCommand(engineType string, image string, config map[string]any, port int) []string {
	switch engineType {
	case "vllm":
		// Check which image is being used
		if strings.Contains(image, "zhiwen-vllm") {
			// GB10 compatible custom image uses nvidia_entrypoint.sh
			// The entrypoint will handle vllm serve automatically
			// We just need to pass the model path and port
			return []string{
				"vllm", "serve", "/models",
				"--port", strconv.Itoa(port),
				"--gpu-memory-utilization", "0.75",
				"--max-model-len", "8192", // Limit context length for GB10
			}
		}
		cmd := []string{"--model", "/models", "--port", strconv.Itoa(port)}
		if gpuUtil, ok := config["gpu_memory_utilization"].(float64); ok {
			cmd = append(cmd, "--gpu-memory-utilization", fmt.Sprintf("%.2f", gpuUtil))
		} else {
			cmd = append(cmd, "--gpu-memory-utilization", "0.75") // Reduced from 0.9
		}
		return cmd
	case "whisper", "asr":
		// Check which image is being used
		if strings.Contains(image, "qujing-glm-asr-nano") {
			// Custom local ASR image uses uvicorn
			return []string{"uvicorn", "main:app", "--host", "0.0.0.0", "--port", strconv.Itoa(port)}
		}
		// FunASR official image - full command
		return []string{
			"/bin/bash",
			"-c",
			fmt.Sprintf("cd /workspace/FunASR/runtime && python -m funasr.run_server --model-dir /models --port %d --vad-dir damo/speech_fsmn_vad_zh-cn-16k-common-pytorch --punc-dir damo/punc_ct-transformer_cn-en-common-vocab471067-large", port),
		}
	case "tts":
		// Check which image is being used
		if strings.Contains(image, "qujing-qwen3-tts") {
			// Custom local TTS image uses uvicorn
			return []string{"uvicorn", "main:app", "--host", "0.0.0.0", "--port", strconv.Itoa(port)}
		}
		// Coqui TTS official image
		return []string{
			"python",
			"-m",
			"TTS.server.server",
			"--model_path", "/models",
			"--port", strconv.Itoa(port),
		}
	default:
		return []string{"--port", strconv.Itoa(port)}
	}
}

// Ensure HybridEngineProvider implements EngineProvider interface
var _ engine.EngineProvider = (*HybridEngineProvider)(nil)

// HybridServiceProvider wraps the hybrid engine provider for service management
type HybridServiceProvider struct {
	hybridProvider *HybridEngineProvider
	modelStore     model.ModelStore
	portCounter    int
	startupOrder   []string // Track startup order
}

// NewHybridServiceProvider creates a new hybrid service provider
func NewHybridServiceProvider(modelStore model.ModelStore) *HybridServiceProvider {
	return &HybridServiceProvider{
		hybridProvider: NewHybridEngineProvider(modelStore),
		modelStore:     modelStore,
		portCounter:    8000,
		startupOrder:   []string{},
	}
}

// Create creates a service configuration
func (p *HybridServiceProvider) Create(ctx context.Context, modelID string, resourceClass service.ResourceClass, replicas int, persistent bool) (*service.ModelService, error) {
	m, err := p.modelStore.Get(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("model not found: %s", modelID)
	}

	engineType := p.hybridProvider.getEngineTypeForModel(m.Type)
	port := p.portCounter
	p.portCounter++

	// Determine device based on engine type and resource limits
	device := "cpu"
	limits := p.hybridProvider.resourceLimits[engineType]
	if limits.GPU {
		device = "gpu"
	}

	return &service.ModelService{
		ID:            fmt.Sprintf("svc-%s-%s", engineType, m.ID),
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
			"device":      device,
			"gpu":         limits.GPU,
		},
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}, nil
}

// Start starts the service with proper resource management
func (p *HybridServiceProvider) Start(ctx context.Context, serviceID string) error {
	// Parse service ID to extract engine type and model ID
	// Format: svc-{engine_type}-{model_id}
	parts := strings.Split(serviceID, "-")
	if len(parts) >= 3 {
		engineType := parts[1]
		modelID := strings.Join(parts[2:], "-")

		// Get model info
		m, err := p.modelStore.Get(ctx, modelID)
		if err == nil {
			// Build config for engine start with resource limits
			limits := p.hybridProvider.resourceLimits[engineType]
			config := map[string]any{
				"model_id":   modelID,
				"model_path": m.Path,
				"device":     "cpu", // Default to CPU for safety
				"gpu":        limits.GPU,
			}

			// Only vLLM gets GPU by default
			if engineType == "vllm" {
				config["device"] = "gpu"
				config["gpu"] = true
				config["gpu_memory_utilization"] = 0.75 // Limit GPU memory
			}

			// Start the engine with retry and health check
			result, err := p.hybridProvider.Start(ctx, engineType, config)
			if err != nil {
				return fmt.Errorf("start engine %s: %w", engineType, err)
			}

			fmt.Printf("✓ Engine started: %s (PID: %s)\n", engineType, result.ProcessID)
			return nil
		}
	}

	return fmt.Errorf("cannot parse service ID or find model: %s", serviceID)
}

// Stop stops the service
func (p *HybridServiceProvider) Stop(ctx context.Context, serviceID string, force bool) error {
	_, err := p.hybridProvider.Stop(ctx, serviceID, force, 30)
	return err
}

// Scale scales the service
func (p *HybridServiceProvider) Scale(ctx context.Context, serviceID string, replicas int) error {
	return fmt.Errorf("scaling not yet implemented")
}

// GetMetrics returns service metrics
func (p *HybridServiceProvider) GetMetrics(ctx context.Context, serviceID string) (*service.ServiceMetrics, error) {
	return &service.ServiceMetrics{}, nil
}

// GetRecommendation provides resource recommendations
func (p *HybridServiceProvider) GetRecommendation(ctx context.Context, modelID string, hint string) (*service.Recommendation, error) {
	m, err := p.modelStore.Get(ctx, modelID)
	if err != nil {
		return nil, err
	}

	engineType := p.hybridProvider.getEngineTypeForModel(m.Type)
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
		Reason:             fmt.Sprintf("Model type '%s' recommended to run on %s with %s engine (CPU forced for non-LLM models on unified memory systems)", m.Type, deviceType, engineType),
	}, nil
}

// GetEngineProvider returns the underlying engine provider
func (p *HybridServiceProvider) GetEngineProvider() engine.EngineProvider {
	return p.hybridProvider
}

// Ensure HybridServiceProvider implements ServiceProvider interface
var _ service.ServiceProvider = (*HybridServiceProvider)(nil)
