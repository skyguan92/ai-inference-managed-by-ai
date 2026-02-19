package vllm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/engine"
)

// Provider implements vLLM inference engine support
type Provider struct {
	mu          sync.RWMutex
	processes   map[string]*exec.Cmd
	services    map[string]*ServiceInfo
	modelStore  model.ModelStore
}

// ServiceInfo holds runtime information about a service
type ServiceInfo struct {
	ServiceID   string
	ModelID     string
	ModelPath   string
	ProcessID   string
	Port        int
	GPUs        []int
	Status      string
	Endpoint    string
	StartedAt   time.Time
}

// NewProvider creates a new vLLM provider
func NewProvider(modelStore model.ModelStore) *Provider {
	return &Provider{
		processes:  make(map[string]*exec.Cmd),
	services:   make(map[string]*ServiceInfo),
		modelStore: modelStore,
	}
}

// Install checks and installs vLLM if needed
func (p *Provider) Install(ctx context.Context, version string) (*engine.InstallResult, error) {
	// Check if vllm is already installed
	if p.isInstalled() {
		path, _ := exec.LookPath("vllm")
		return &engine.InstallResult{
			Success: true,
			Path:    path,
		}, nil
	}

	// Install vLLM via pip
	fmt.Println("Installing vLLM...")
	cmd := exec.CommandContext(ctx, "pip", "install", "vllm")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to install vLLM: %w", err)
	}

	// Verify installation
	if !p.isInstalled() {
		return nil, fmt.Errorf("vLLM installation verification failed")
	}

	path, _ := exec.LookPath("vllm")
	return &engine.InstallResult{
		Success: true,
		Path:    path,
	}, nil
}

func (p *Provider) isInstalled() bool {
	_, err := exec.LookPath("vllm")
	return err == nil
}

// Create creates a new vLLM service configuration
func (p *Provider) Create(ctx context.Context, modelID string, resourceClass service.ResourceClass, replicas int, persistent bool) (*service.ModelService, error) {
	// Get model info from store
	var modelInfo *model.Model
	if p.modelStore != nil {
		m, err := p.modelStore.Get(ctx, modelID)
		if err != nil {
			return nil, fmt.Errorf("model not found: %s", modelID)
		}
		modelInfo = m
	} else {
		// Fallback: create minimal model info
		modelInfo = &model.Model{
			ID:   modelID,
			Name: modelID,
			Path: "/mnt/data/models/" + modelID,
		}
	}

	// Allocate port
	port := p.allocatePort()

	serviceID := "svc-vllm-" + uuid.New().String()[:8]
	
	now := time.Now().Unix()
	svc := &service.ModelService{
		ID:            serviceID,
		Name:          "vllm-" + modelInfo.Name,
		ModelID:       modelID,
		Status:        service.ServiceStatusCreating,
		Replicas:      replicas,
		ResourceClass: resourceClass,
		Endpoints:     []string{fmt.Sprintf("http://localhost:%d", port)},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Store service info
	p.mu.Lock()
	p.services[serviceID] = &ServiceInfo{
		ServiceID: serviceID,
		ModelID:   modelID,
		ModelPath: modelInfo.Path,
		Port:      port,
		Status:    "creating",
	}
	p.mu.Unlock()

	return svc, nil
}

// Start starts the vLLM service
func (p *Provider) Start(ctx context.Context, serviceID string) error {
	p.mu.Lock()
	svcInfo, exists := p.services[serviceID]
	if !exists {
		p.mu.Unlock()
		return fmt.Errorf("service not found: %s", serviceID)
	}
	p.mu.Unlock()

	// Check if already running
	if p.isRunning(serviceID) {
		return fmt.Errorf("service already running: %s", serviceID)
	}

	// Build vLLM serve command
	args := p.buildServeArgs(svcInfo)
	
	fmt.Printf("Starting vLLM service: %s\n", serviceID)
	fmt.Printf("Command: vllm serve %s %v\n", svcInfo.ModelPath, args)

	cmd := exec.CommandContext(ctx, "vllm", append([]string{"serve", svcInfo.ModelPath}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	// Set GPU devices if specified
	if len(svcInfo.GPUs) > 0 {
		gpuStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(svcInfo.GPUs)), ","), "[]")
		cmd.Env = append(cmd.Env, fmt.Sprintf("CUDA_VISIBLE_DEVICES=%s", gpuStr))
	}

	// Start process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start vLLM: %w", err)
	}

	// Store process
	p.mu.Lock()
	p.processes[serviceID] = cmd
	svcInfo.ProcessID = strconv.Itoa(cmd.Process.Pid)
	svcInfo.Status = "starting"
	svcInfo.StartedAt = time.Now()
	p.mu.Unlock()

	// Wait for service to be ready
	if err := p.waitForReady(ctx, svcInfo); err != nil {
		p.Stop(ctx, serviceID, true)
		return fmt.Errorf("service failed to start: %w", err)
	}

	p.mu.Lock()
	svcInfo.Status = "running"
	svcInfo.Endpoint = fmt.Sprintf("http://localhost:%d", svcInfo.Port)
	p.mu.Unlock()

	fmt.Printf("vLLM service started successfully: %s at %s\n", serviceID, svcInfo.Endpoint)
	return nil
}

// Stop stops the vLLM service
func (p *Provider) Stop(ctx context.Context, serviceID string, force bool) error {
	p.mu.Lock()
	cmd, exists := p.processes[serviceID]
	p.mu.Unlock()

	if !exists {
		return fmt.Errorf("service not running: %s", serviceID)
	}

	fmt.Printf("Stopping vLLM service: %s\n", serviceID)

	if force {
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	} else {
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			return fmt.Errorf("failed to send interrupt: %w", err)
		}

		// Wait for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-time.After(30 * time.Second):
			cmd.Process.Kill()
		case <-done:
		}
	}

	p.mu.Lock()
	delete(p.processes, serviceID)
	if svc, ok := p.services[serviceID]; ok {
		svc.Status = "stopped"
	}
	p.mu.Unlock()

	return nil
}

// Scale scales the service (vLLM supports multiple replicas via separate instances)
func (p *Provider) Scale(ctx context.Context, serviceID string, replicas int) error {
	// For vLLM, scaling means starting additional instances
	// This is a simplified implementation
	return fmt.Errorf("scaling not yet implemented for vLLM")
}

// GetMetrics returns service metrics
func (p *Provider) GetMetrics(ctx context.Context, serviceID string) (*service.ServiceMetrics, error) {
	p.mu.RLock()
	_, exists := p.services[serviceID]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("service not found: %s", serviceID)
	}

	// TODO: Query vLLM metrics endpoint
	return &service.ServiceMetrics{
		RequestsPerSecond: 0,
		LatencyP50:        0,
		LatencyP99:        0,
		TotalRequests:     0,
		ErrorRate:         0,
	}, nil
}

// GetRecommendation provides resource recommendations
func (p *Provider) GetRecommendation(ctx context.Context, modelID string, hint string) (*service.Recommendation, error) {
	return &service.Recommendation{
		ResourceClass:      service.ResourceClassLarge,
		Replicas:           1,
		ExpectedThroughput: 50.0,
	}, nil
}

// SetGPUDevices sets which GPUs to use for a service
func (p *Provider) SetGPUDevices(serviceID string, gpus []int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	svcInfo, exists := p.services[serviceID]
	if !exists {
		return fmt.Errorf("service not found: %s", serviceID)
	}

	svcInfo.GPUs = gpus
	return nil
}

// buildServeArgs builds vllm serve command arguments
func (p *Provider) buildServeArgs(svcInfo *ServiceInfo) []string {
	var args []string

	// Port
	args = append(args, "--port", strconv.Itoa(svcInfo.Port))

	// GPU memory utilization
	args = append(args, "--gpu-memory-utilization", "0.9")

	// Tensor parallel size (number of GPUs)
	if len(svcInfo.GPUs) > 1 {
		args = append(args, "--tensor-parallel-size", strconv.Itoa(len(svcInfo.GPUs)))
	}

	// Enable OpenAI-compatible API
	args = append(args, "--api-key", "")

	return args
}

// waitForReady waits for the service to be ready
func (p *Provider) waitForReady(ctx context.Context, svcInfo *ServiceInfo) error {
	_ = fmt.Sprintf("http://localhost:%d/health", svcInfo.Port)
	
	for i := 0; i < 60; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}

		// Check if process is still running
		p.mu.RLock()
		cmd, exists := p.processes[svcInfo.ServiceID]
		p.mu.RUnlock()

		if !exists {
			return fmt.Errorf("process not found")
		}

		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			return fmt.Errorf("process exited with code %d", cmd.ProcessState.ExitCode())
		}

		// Try to connect to health endpoint
		// TODO: Implement actual HTTP check
		if i > 10 { // Give it some time to start
			return nil // Assume ready for now
		}
	}

	return fmt.Errorf("timeout waiting for service to be ready")
}

// isRunning checks if a service is running
func (p *Provider) isRunning(serviceID string) bool {
	p.mu.RLock()
	cmd, exists := p.processes[serviceID]
	p.mu.RUnlock()

	if !exists {
		return false
	}

	return cmd.ProcessState == nil || !cmd.ProcessState.Exited()
}

// allocatePort allocates a random available port
func (p *Provider) allocatePort() int {
	// Simple port allocation starting from 8000
	p.mu.Lock()
	defer p.mu.Unlock()

	basePort := 8000
	for {
		port := basePort
		available := true
		for _, svc := range p.services {
			if svc.Port == port {
				available = false
				break
			}
		}
		if available {
			return port
		}
		basePort++
		if basePort > 9000 {
			break
		}
	}
	return 8000
}

// GetServiceInfo returns service runtime info
func (p *Provider) GetServiceInfo(serviceID string) (*ServiceInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	svcInfo, exists := p.services[serviceID]
	if !exists {
		return nil, fmt.Errorf("service not found: %s", serviceID)
	}

	return svcInfo, nil
}

// DiscoverModels discovers models in a directory
func (p *Provider) DiscoverModels(basePath string) ([]string, error) {
	var models []string

	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check for model files
		modelPath := filepath.Join(basePath, entry.Name())
		if p.isModelDirectory(modelPath) {
			models = append(models, entry.Name())
		}
	}

	return models, nil
}

func (p *Provider) isModelDirectory(path string) bool {
	// Check for common model file patterns
	patterns := []string{"*.safetensors", "*.bin", "*.pt", "*.pth", "config.json"}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(path, pattern))
		if err == nil && len(matches) > 0 {
			return true
		}
	}

	return false
}
