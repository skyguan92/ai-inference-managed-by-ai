package provider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	catalogdata "github.com/jguan/ai-inference-managed-by-ai/catalog"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/docker"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/eventbus"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/catalog"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/engine"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
)

// fatalStartError wraps an error that should not be retried (e.g. non-AIMA port conflict).
type fatalStartError struct{ cause error }

func (e *fatalStartError) Error() string { return e.cause.Error() }
func (e *fatalStartError) Unwrap() error  { return e.cause }

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
	dockerClient    docker.Client
	containers      map[string]string // name -> container ID
	nativeProcesses map[string]*exec.Cmd
	serviceInfo     map[string]*ServiceInfo
	modelStore      model.ModelStore

	// Resource management
	resourceLimits map[string]ResourceLimits
	startupConfigs map[string]StartupConfig

	// Engine assets loaded from YAML files (keyed by engine type)
	engineAssets map[string]catalog.EngineAsset

	// Event publishing (optional)
	eventBus eventbus.EventBus

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

// NewHybridEngineProvider creates a new hybrid engine provider.
// Defaults to SDKClient (Docker Go SDK); falls back to SimpleClient (CLI-based) on error.
func NewHybridEngineProvider(modelStore model.ModelStore) *HybridEngineProvider {
	dc, err := docker.NewSDKClient()
	if err != nil {
		slog.Warn("failed to create Docker SDK client, falling back to CLI client", "error", err)
		return newHybridEngineProviderWithClient(modelStore, docker.NewSimpleClient())
	}
	return newHybridEngineProviderWithClient(modelStore, dc)
}

// newHybridEngineProviderWithClient creates a HybridEngineProvider with a specific docker.Client.
// Used in tests to inject a mock client.
func newHybridEngineProviderWithClient(modelStore model.ModelStore, dc docker.Client) *HybridEngineProvider {
	assets, err := catalog.LoadEngineAssetsFromFS(catalogdata.EngineFS, "engines")
	if err != nil {
		slog.Warn("failed to load embedded engine assets, using hardcoded defaults", "error", err)
		assets = make(map[string]catalog.EngineAsset)
	} else {
		slog.Info("loaded engine assets from embedded YAML", "count", len(assets))
	}

	return &HybridEngineProvider{
		dockerClient:    dc,
		containers:      make(map[string]string),
		nativeProcesses: make(map[string]*exec.Cmd),
		serviceInfo:     make(map[string]*ServiceInfo),
		modelStore:      modelStore,
		resourceLimits:  getDefaultResourceLimits(),
		startupConfigs:  getDefaultStartupConfigs(),
		engineAssets:    assets,
	}
}

// AssetTypes returns the engine type keys from the loaded YAML assets.
// This is used by the CLI to seed the EngineStore at startup.
func (p *HybridEngineProvider) AssetTypes() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	types := make([]string, 0, len(p.engineAssets))
	for k := range p.engineAssets {
		types = append(types, k)
	}
	return types
}

// SetEventBus injects an event bus so the provider can publish progress events.
func (p *HybridEngineProvider) SetEventBus(bus eventbus.EventBus) {
	p.mu.Lock()
	p.eventBus = bus
	p.mu.Unlock()
}

// publishProgress fires a StartProgressEvent if an event bus is configured.
// It is fire-and-forget; errors are silently ignored.
func (p *HybridEngineProvider) publishProgress(serviceID, phase, message string, progress int) {
	p.mu.RLock()
	bus := p.eventBus
	p.mu.RUnlock()
	if bus == nil {
		return
	}
	_ = bus.Publish(engine.NewStartProgressEvent(serviceID, phase, message, progress))
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
			MaxRetries:     5,
			RetryInterval:  10 * time.Second,
			StartupTimeout: 1200 * time.Second, // 20 minutes for large models like Qwen3-Omni-30B
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
		slog.Info("checking Docker images", "engine", name)

		// Check all candidates for local availability
		for _, image := range candidates {
			if p.imageExists(image) {
				slog.Info("Docker image already exists", "image", image)
				return &engine.InstallResult{
					Success: true,
					Path:    image,
				}, nil
			}
		}

		// No local image found, try to pull the first candidate
		image := candidates[0]
		slog.Warn("no local image found, pulling", "image", image)
		pullCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		if err := p.dockerClient.PullImage(pullCtx, image); err != nil {
			slog.Warn("failed to pull image, will try existing image or native mode", "image", image, "error", err)
		} else {
			slog.Info("Docker image pulled successfully", "image", image)
			return &engine.InstallResult{
				Success: true,
				Path:    image,
			}, nil
		}
	} else {
		slog.Debug("Docker not available", "error", err)
	}

	// Fall back to native mode
	if p.checkNativeBinary(name) {
		slog.Info("native binary available", "engine", name)
		return &engine.InstallResult{
			Success: true,
			Path:    name,
		}, nil
	}

	// Return success anyway - will try to start with available resources
	slog.Warn("neither Docker image nor native binary available, will attempt best-effort mode", "engine", name)
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
	// Allow caller to override port via config (e.g. when a service has a stored port assignment).
	if configPort, ok := config["port"]; ok {
		switch v := configPort.(type) {
		case int:
			port = v
		case int64:
			port = int(v)
		case float64:
			port = int(v)
		}
	}

	// Check for async mode
	asyncMode := false
	if async, ok := config["async"].(bool); ok {
		asyncMode = async
	}

	// Try Docker first if available
	if docker.CheckDocker() == nil {
		var lastErr error
		for attempt := 1; attempt <= startupCfg.MaxRetries; attempt++ {
			if attempt > 1 {
				slog.Info("retrying engine start", "engine", name, "attempt", attempt, "max_retries", startupCfg.MaxRetries)
				time.Sleep(startupCfg.RetryInterval)
			}

			result, err := p.startDockerWithRetry(ctx, engineType, modelPath, port, useGPU, config, limits)
			if err == nil {
				// In async mode, don't wait for health check
				if asyncMode {
					slog.Info("async mode: container started, model loading in background", "container_id", result.ProcessID[:12])
					return result, nil
				}

				// Wait for health check
				if err := p.waitForHealth(ctx, engineType, result.ProcessID, port, startupCfg.HealthCheckURL, startupCfg.StartupTimeout); err != nil {
					slog.Warn("health check failed", "error", err)

					// If the request context expired, waitForHealth already cleaned up the
					// container. No point retrying with an expired context.
					if ctx.Err() != nil {
						slog.Warn("request context expired, aborting retries", "engine", name, "error", ctx.Err())
						lastErr = ctx.Err()
						break
					}

					// Bug #3 fix: Check if container is still running before stopping
					status, statusErr := p.dockerClient.GetContainerStatus(ctx, result.ProcessID)
					if statusErr == nil && status == "running" {
						// Container is running but health check timed out
						// This usually means the model is still loading
						slog.Warn("health check timeout but container still running, model may still be loading",
							"container_id", result.ProcessID[:12],
							"model_path", modelPath,
							"status", status)
						// Return success - the container is healthy and model is loading
						return result, nil
					}

					// Bug #14 fix: Capture container logs before cleanup to aid debugging.
					if logs, logErr := p.dockerClient.GetContainerLogs(ctx, result.ProcessID, 10); logErr == nil && logs != "" {
						slog.Warn("container failed, last logs", "container_id", result.ProcessID[:12], "logs", logs)
					}

					_, _ = p.Stop(ctx, engineType, true, 10)
					// Wait briefly for port to be released before retrying.
					time.Sleep(2 * time.Second)
					lastErr = err
					continue
				}
				return result, nil
			}
			lastErr = err
			var fatal *fatalStartError
			if errors.As(err, &fatal) {
				slog.Warn("Docker start failed (fatal, aborting retries)", "attempt", attempt, "error", err)
				break
			}
			slog.Warn("Docker start failed", "attempt", attempt, "error", err)
		}
		slog.Warn("Docker start failed after retries, trying native mode", "attempts", startupCfg.MaxRetries, "error", lastErr)
	}

	// Fall back to native mode
	return p.startNative(ctx, engineType, modelPath, port, useGPU, config)
}

// startDockerWithRetry starts engine in Docker container with resource limits
func (p *HybridEngineProvider) startDockerWithRetry(ctx context.Context, engineType, modelPath string, port int, useGPU bool, config map[string]any, limits ResourceLimits) (*engine.StartResult, error) {
	p.publishProgress(engineType, "pulling", "Checking image for "+engineType, 0)

	image := p.getDockerImage(engineType, "")

	// Check if image exists
	if !p.imageExists(image) {
		err := fmt.Errorf("Docker image not found: %s", image)
		p.publishProgress(engineType, "failed", err.Error(), -1)
		return nil, err
	}

	p.publishProgress(engineType, "pulling", "Image found: "+image, 50)

	// Phase 1 — Port-based: detect any container occupying the port we need.
	// This finds externally-created containers that label-based listing would miss.
	portScanCtx, portScanCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer portScanCancel()
	portConflicts, err := p.dockerClient.FindContainersByPort(portScanCtx, port)
	if err != nil {
		slog.Warn("port conflict check failed, proceeding without port scan", "port", port, "error", err)
	}
	for _, conflict := range portConflicts {
		if !conflict.IsAIMA {
			// Non-AIMA container: we cannot safely remove it; surface an actionable error.
			return nil, &fatalStartError{cause: fmt.Errorf(
				"port %d is occupied by non-AIMA container %q (image: %s). Remove it with: docker rm -f %s",
				port, conflict.Name, conflict.Image, conflict.ContainerID,
			)}
		}
		slog.Info("removing AIMA container blocking port", "container_id", conflict.ContainerID, "port", port, "engine", engineType)
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := p.dockerClient.StopContainer(cleanupCtx, conflict.ContainerID, 10); err != nil {
			slog.Warn("failed to remove conflicting AIMA container", "container_id", conflict.ContainerID, "error", err)
		}
		cancel()
	}

	// Phase 2 — Label-based: catch "created" state containers that haven't bound
	// their port yet (so FindContainersByPort wouldn't find them).
	// Note: containers stopped in Phase 1 may also appear here; StopContainer is idempotent.
	listCtx, listCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer listCancel()
	staleIDs, err := p.dockerClient.ListContainers(listCtx, map[string]string{"aima.engine": engineType})
	if err != nil {
		slog.Warn("stale container scan failed, proceeding without label cleanup", "engine", engineType, "error", err)
	}
	for _, cid := range staleIDs {
		slog.Info("removing stale AIMA container before start", "container_id", cid, "engine", engineType)
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := p.dockerClient.StopContainer(cleanupCtx, cid, 10); err != nil {
			slog.Warn("failed to remove stale container", "container_id", cid, "error", err)
		}
		cancel()
	}

	if len(portConflicts) > 0 || len(staleIDs) > 0 {
		// Brief wait for Docker to fully release ports after container removal.
		time.Sleep(2 * time.Second)
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
		if strings.Contains(image, "qujing-glm-asr-nano") || strings.Contains(image, "qujing-qwen3-tts") {
			mountPath = "/model" // ASR and TTS images expect /model
		}
		opts.Volumes = map[string]string{
			modelPath: mountPath,
		}
	}

	// Build command based on engine type
	opts.Cmd = p.buildDockerCommand(engineType, image, config, port)

	containerName := fmt.Sprintf("aima-%s-%d", engineType, time.Now().Unix())

	slog.Info("starting engine in Docker container",
		"engine", engineType, "image", image, "model_path", modelPath,
		"port", port, "gpu", useGPU, "memory_limit", limits.Memory, "cpu_limit", limits.CPU)

	containerID, err := p.dockerClient.CreateAndStartContainer(ctx, containerName, image, opts)
	if err != nil {
		p.publishProgress(engineType, "failed", err.Error(), -1)
		return nil, err
	}

	p.mu.Lock()
	p.containers[engineType] = containerID
	p.mu.Unlock()

	shortID := containerID
	if len(containerID) > 12 {
		shortID = containerID[:12]
	}
	slog.Info("container started", "container_id", shortID, "endpoint", fmt.Sprintf("http://localhost:%d", port))
	p.publishProgress(engineType, "starting", "Container started: "+shortID, 70)

	return &engine.StartResult{
		ProcessID: containerID,
		Status:    engine.EngineStatusRunning,
	}, nil
}

// waitForHealth waits for service to become healthy
func (p *HybridEngineProvider) waitForHealth(ctx context.Context, engineType, containerID string, port int, healthPath string, timeout time.Duration) error {
	if healthPath == "" {
		healthPath = "/health"
	}

	endpoint := fmt.Sprintf("http://localhost:%d%s", port, healthPath)
	slog.Info("waiting for health check", "endpoint", endpoint, "timeout", timeout)
	p.publishProgress(engineType, "loading", "Waiting for health check...", 75)

	deadline := time.Now().Add(timeout)
	checkInterval := 2 * time.Second

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			// Context cancelled (e.g. gateway timeout) — clean up the container
			// so the port is freed for subsequent start attempts.
			shortID := containerID
			if len(shortID) > 12 {
				shortID = shortID[:12]
			}
			slog.Warn("health check cancelled, cleaning up container",
				"container_id", shortID, "reason", ctx.Err())
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cleanupCancel()
			if stopErr := p.dockerClient.StopContainer(cleanupCtx, containerID, 10); stopErr != nil {
				slog.Warn("failed to stop container during cleanup", "container_id", shortID, "error", stopErr)
			}
			p.mu.Lock()
			delete(p.containers, engineType)
			p.mu.Unlock()
			return fmt.Errorf("health check cancelled, container cleaned up: %w", ctx.Err())
		default:
		}

		// Try HTTP health check
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(endpoint)
		if err == nil {
			func() {
				defer func() { _ = resp.Body.Close() }()
				if resp.StatusCode == http.StatusOK {
					slog.Info("health check passed", "endpoint", endpoint)
				}
			}()
			if resp.StatusCode == http.StatusOK {
				p.publishProgress(engineType, "ready", "Engine ready at port "+strconv.Itoa(port), 100)
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
	slog.Info("starting engine as native process", "engine", engineType)

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

	slog.Debug("native process command", "command", "vllm "+strings.Join(args, " "))

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start vllm: %w", err)
	}

	p.mu.Lock()
	p.nativeProcesses[engineType] = cmd
	p.mu.Unlock()

	// Start goroutine to wait for process and clean up
	go func() {
		if err := cmd.Wait(); err != nil {
			slog.Error("native process exited with error", "engine", engineType, "error", err)
		}
		p.mu.Lock()
		delete(p.nativeProcesses, engineType)
		p.mu.Unlock()
	}()

	slog.Info("native process started", "pid", cmd.Process.Pid, "endpoint", fmt.Sprintf("http://localhost:%d", port))

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
		slog.Info("stopping Docker container", "container_id", containerID[:12])
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
		slog.Info("stopping native process", "pid", cmd.Process.Pid)
		if force {
			_ = cmd.Process.Kill()
		} else {
			_ = cmd.Process.Signal(os.Interrupt)
			// Wait for graceful shutdown
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			select {
			case <-done:
			case <-time.After(time.Duration(timeout) * time.Second):
				_ = cmd.Process.Kill()
			}
		}
		p.mu.Lock()
		delete(p.nativeProcesses, name)
		p.mu.Unlock()
		return &engine.StopResult{Success: true}, nil
	}

	// Fallback: query Docker by label to find containers from previous sessions.
	// Containers are labeled aima.managed=true + aima.engine=<engineType> at creation time.
	if docker.CheckDocker() == nil {
		containerIDs, err := p.dockerClient.ListContainers(ctx, map[string]string{"aima.engine": name})
		if err == nil && len(containerIDs) > 0 {
			for _, cid := range containerIDs {
				slog.Info("stopping orphaned container found by label", "container_id", cid, "engine", name)
				if stopErr := p.dockerClient.StopContainer(ctx, cid, timeout); stopErr != nil {
					slog.Warn("failed to stop orphaned container", "container_id", cid, "error", stopErr)
				}
			}
			return &engine.StopResult{Success: true}, nil
		}

		// Port-based fallback: find AIMA containers by default port when label lookup
		// misses them (e.g., started without aima.engine label from a previous AIMA version).
		port := p.getDefaultPort(name)
		if portConflicts, portErr := p.dockerClient.FindContainersByPort(ctx, port); portErr == nil {
			found := 0
			for _, conflict := range portConflicts {
				if conflict.IsAIMA {
					slog.Info("stopping AIMA container found by port", "container_id", conflict.ContainerID, "port", port, "engine", name)
					if stopErr := p.dockerClient.StopContainer(ctx, conflict.ContainerID, timeout); stopErr != nil {
						slog.Warn("failed to stop container found by port", "container_id", conflict.ContainerID, "error", stopErr)
					}
					found++
				}
			}
			if found > 0 {
				return &engine.StopResult{Success: true}, nil
			}
		}
	}

	slog.Debug("service not found, nothing to stop", "name", name)
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

	// Use YAML-loaded asset when available.
	if asset, ok := p.engineAssets[name]; ok && asset.ImageFullName != "" {
		images := []string{asset.ImageFullName}
		images = append(images, asset.AlternativeNames...)
		return images
	}

	// Hardcoded fallback.
	switch name {
	case "vllm":
		// Priority: GB10 compatible (general) > Qwen3-Omni specific > official image
		return []string{
			"zhiwen-vllm:0128",              // GB10 compatible - supports most models (priority 1)
			"aima-vllm-qwen3-omni:latest",   // Qwen3-Omni vLLM specific (priority 2)
			"aima-qwen3-omni-server:latest", // Qwen3-Omni FastAPI server (priority 3)
			"vllm/vllm-openai:v0.15.0",      // Official image (fallback)
		}
	case "whisper", "asr":
		// Priority: local custom image > official image
		return []string{
			"qujing-glm-asr-nano:latest", // Local custom image (priority 1)
			"registry.cn-hangzhou.aliyuncs.com/funasr_repo/funasr:funasr-runtime-sdk-cpu-0.4.6", // Official (priority 2)
		}
	case "tts":
		// Priority: real TTS image > placeholder image > official image
		return []string{
			"qujing-qwen3-tts-real:latest", // Real TTS with model loading (priority 1)
			"qujing-qwen3-tts:latest",      // Placeholder TTS image (priority 2)
			"ghcr.io/coqui-ai/tts:latest",  // Official (priority 3)
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
			slog.Debug("using local image", "image", image)
			return image
		}
	}

	// Return first candidate (will trigger pull attempt)
	slog.Warn("no local image found, will attempt to pull", "image", candidates[0])
	return candidates[0]
}

func (p *HybridEngineProvider) getDefaultPort(engineType string) int {
	// Prefer port from YAML asset when available.
	if asset, ok := p.engineAssets[engineType]; ok && asset.DefaultPort > 0 {
		return asset.DefaultPort
	}

	// Hardcoded fallback.
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

// applyPortToArgs replaces the value after "--port" in args (or appends it) and
// returns the modified slice. The input slice is not modified.
func applyPortToArgs(args []string, port int) []string {
	result := make([]string, len(args))
	copy(result, args)
	portStr := strconv.Itoa(port)
	for i, arg := range result {
		if arg == "--port" && i+1 < len(result) {
			result[i+1] = portStr
			return result
		}
	}
	return append(result, "--port", portStr)
}

func (p *HybridEngineProvider) buildDockerCommand(engineType string, image string, config map[string]any, port int) []string {
	// Image-specific overrides: custom images with their own CMD/ENTRYPOINT.
	if strings.Contains(image, "aima-qwen3-omni-server") {
		return nil // Custom FastAPI server, Dockerfile already has CMD
	}

	// Use YAML-asset command + DefaultArgs when available (with port substitution).
	if asset, ok := p.engineAssets[engineType]; ok && len(asset.DefaultArgs) > 0 {
		cmd := make([]string, 0, len(asset.BaseCommand)+len(asset.DefaultArgs)+2)
		cmd = append(cmd, asset.BaseCommand...)
		cmd = append(cmd, asset.DefaultArgs...)
		return applyPortToArgs(cmd, port)
	}

	// Hardcoded fallbacks for when YAML assets are not available.
	switch engineType {
	case "vllm":
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
	mu             sync.Mutex
	hybridProvider *HybridEngineProvider
	modelStore     model.ModelStore
	serviceStore   service.ServiceStore
	portCounter    int
	startupOrder   []string // Track startup order
}

// NewHybridServiceProvider creates a new hybrid service provider.
// It scans existing services in serviceStore to resume port assignment after
// the previous port used, avoiding port collisions on restart.
func NewHybridServiceProvider(modelStore model.ModelStore, serviceStore service.ServiceStore) *HybridServiceProvider {
	portCounter := 8000

	// Scan existing services to find the highest port in use.
	services, _, err := serviceStore.List(context.Background(), service.ServiceFilter{})
	if err == nil {
		for _, svc := range services {
			if svc.Config == nil {
				continue
			}
			portVal, ok := svc.Config["port"]
			if !ok {
				continue
			}
			var port int
			switch v := portVal.(type) {
			case int:
				port = v
			case int64:
				port = int(v)
			case float64:
				port = int(v)
			}
			if port >= portCounter {
				portCounter = port + 1
			}
		}
	}

	return &HybridServiceProvider{
		hybridProvider: NewHybridEngineProvider(modelStore),
		modelStore:     modelStore,
		serviceStore:   serviceStore,
		portCounter:    portCounter,
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
	p.mu.Lock()
	port := p.portCounter
	p.portCounter++
	p.mu.Unlock()

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
	return p.StartAsync(ctx, serviceID, false)
}

// StartAsync starts the service with async mode support
// For large models like Qwen3-Omni, async mode allows starting without waiting for health check
func (p *HybridServiceProvider) StartAsync(ctx context.Context, serviceID string, async bool) error {
	// Parse service ID to extract engine type and model ID
	sid, parseErr := service.ParseServiceID(serviceID)
	if parseErr != nil {
		return fmt.Errorf("cannot parse service ID: %w", parseErr)
	}

	engineType := sid.EngineType
	modelID := sid.ModelID

	// Get model info
	m, err := p.modelStore.Get(ctx, modelID)
	if err != nil {
		return fmt.Errorf("cannot find model %s: %w", modelID, err)
	}

	// Build config for engine start with resource limits
	limits := p.hybridProvider.resourceLimits[engineType]
	config := map[string]any{
		"model_id":   modelID,
		"model_path": m.Path,
		"device":     "cpu", // Default to CPU for safety
		"gpu":        limits.GPU,
		"async":      async, // Pass async flag to engine
	}

	// Only vLLM gets GPU by default
	if engineType == "vllm" {
		config["device"] = "gpu"
		config["gpu"] = true
		config["gpu_memory_utilization"] = 0.75 // Limit GPU memory
	}

	// Read the persisted port assignment for this service from the store.
	// This ensures two services with different ports don't both default to 8000.
	if svc, svcErr := p.serviceStore.Get(ctx, serviceID); svcErr == nil && svc.Config != nil {
		if portVal, ok := svc.Config["port"]; ok {
			config["port"] = portVal
		}
	}

	// Start the engine with retry and health check
	result, err := p.hybridProvider.Start(ctx, engineType, config)
	if err != nil {
		return fmt.Errorf("start engine %s: %w", engineType, err)
	}

	if async {
		slog.Info("engine started in async mode", "engine", engineType, "container_id", result.ProcessID[:12])
	} else {
		slog.Info("engine started", "engine", engineType, "process_id", result.ProcessID)
	}
	return nil
}

// Stop stops the service
func (p *HybridServiceProvider) Stop(ctx context.Context, serviceID string, force bool) error {
	// Parse engine type from service ID: svc-{engine_type}-{model_id}
	// hybridProvider.Stop is keyed by engineType, not serviceID.
	engineType := serviceID
	if sid, parseErr := service.ParseServiceID(serviceID); parseErr == nil {
		engineType = sid.EngineType
	}
	_, err := p.hybridProvider.Stop(ctx, engineType, force, 30)
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

// IsRunning checks if the service container/process is actually running
func (p *HybridServiceProvider) IsRunning(ctx context.Context, serviceID string) bool {
	p.hybridProvider.mu.RLock()
	info, exists := p.hybridProvider.serviceInfo[serviceID]
	p.hybridProvider.mu.RUnlock()

	if !exists || info == nil || info.ProcessID == "" {
		return false
	}

	// Check if Docker container is running
	if len(info.ProcessID) == 64 {
		status, err := p.hybridProvider.dockerClient.GetContainerStatus(ctx, info.ProcessID)
		if err != nil {
			return false
		}
		return status == "running"
	}

	// For native processes, check if process exists
	// ProcessID is the PID for native processes
	return true // Simplified check for now
}

// GetLogs returns the last tail lines of logs for the service container.
func (p *HybridServiceProvider) GetLogs(ctx context.Context, serviceID string, tail int) (string, error) {
	// First: check in-memory service info (populated when service was started in this session)
	p.hybridProvider.mu.RLock()
	info, exists := p.hybridProvider.serviceInfo[serviceID]
	p.hybridProvider.mu.RUnlock()

	if exists && info != nil && info.ProcessID != "" {
		logs, err := p.hybridProvider.dockerClient.GetContainerLogs(ctx, info.ProcessID, tail)
		if err != nil {
			return "", fmt.Errorf("get container logs for %s: %w", info.ProcessID, err)
		}
		return logs, nil
	}

	// Fallback: parse engine type from service ID and search by label
	// Service ID format: svc-{engineType}-{modelId}
	sid, parseErr := service.ParseServiceID(serviceID)
	if parseErr == nil {
		containers, listErr := p.hybridProvider.dockerClient.ListContainers(ctx, map[string]string{"aima.engine": sid.EngineType})
		if listErr == nil && len(containers) > 0 {
			logs, logErr := p.hybridProvider.dockerClient.GetContainerLogs(ctx, containers[0], tail)
			if logErr == nil {
				return logs, nil
			}
		}
	}

	return "", fmt.Errorf("no running container found for service %s", serviceID)
}

// GetEngineProvider returns the underlying engine provider
func (p *HybridServiceProvider) GetEngineProvider() engine.EngineProvider {
	return p.hybridProvider
}

// Ensure HybridServiceProvider implements ServiceProvider interface
var _ service.ServiceProvider = (*HybridServiceProvider)(nil)
