package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/engine"
)

var (
	ErrHealthCheckFailed  = errors.New("engine health check failed")
	ErrHealthCheckTimeout = errors.New("engine health check timeout")
	ErrInstallFailed      = errors.New("engine install failed")
)

type HealthStatus struct {
	Healthy   bool   `json:"healthy"`
	Message   string `json:"message,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

type EngineFullStatus struct {
	Engine  *engine.Engine         `json:"engine"`
	Health  *HealthStatus          `json:"health"`
	Feature *engine.EngineFeatures `json:"features,omitempty"`
}

type AvailableEngine struct {
	Name    string              `json:"name"`
	Type    engine.EngineType   `json:"type"`
	Status  engine.EngineStatus `json:"status"`
	Version string              `json:"version,omitempty"`
}

type InstallEngineResult struct {
	Success bool   `json:"success"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message,omitempty"`
}

type EngineService struct {
	registry            *unit.Registry
	store               engine.EngineStore
	provider            engine.EngineProvider
	healthCheckInterval time.Duration
	healthCheckTimeout  time.Duration
}

func NewEngineService(registry *unit.Registry, store engine.EngineStore, provider engine.EngineProvider) *EngineService {
	return &EngineService{
		registry:            registry,
		store:               store,
		provider:            provider,
		healthCheckInterval: 2 * time.Second,
		healthCheckTimeout:  30 * time.Second,
	}
}

func (s *EngineService) SetHealthCheckConfig(interval, timeout time.Duration) {
	s.healthCheckInterval = interval
	s.healthCheckTimeout = timeout
}

func (s *EngineService) StartWithHealthCheck(ctx context.Context, name string) (*EngineFullStatus, error) {
	startCmd := s.registry.GetCommand("engine.start")
	if startCmd == nil {
		return nil, unit.ErrCommandNotFound
	}

	result, err := startCmd.Execute(ctx, map[string]any{"name": name})
	if err != nil {
		return nil, fmt.Errorf("start engine %s: %w", name, err)
	}

	startResult, ok := result.(map[string]any)
	if !ok {
		return nil, errors.New("invalid start command result")
	}

	if status, _ := startResult["status"].(string); status != string(engine.EngineStatusRunning) {
		return nil, fmt.Errorf("engine %s did not reach running status: %s", name, status)
	}

	healthCtx, cancel := context.WithTimeout(ctx, s.healthCheckTimeout)
	defer cancel()

	healthStatus := s.waitForHealthy(healthCtx, name)
	if !healthStatus.Healthy {
		return nil, fmt.Errorf("%w: %s", ErrHealthCheckFailed, healthStatus.Message)
	}

	return s.GetStatus(ctx, name)
}

func (s *EngineService) StopGracefully(ctx context.Context, name string, timeout time.Duration) error {
	stopCmd := s.registry.GetCommand("engine.stop")
	if stopCmd == nil {
		return unit.ErrCommandNotFound
	}

	timeoutSeconds := int(timeout.Seconds())
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}

	input := map[string]any{
		"name":    name,
		"force":   false,
		"timeout": timeoutSeconds,
	}

	_, err := stopCmd.Execute(ctx, input)
	if err != nil {
		return fmt.Errorf("stop engine %s gracefully: %w", name, err)
	}

	return nil
}

func (s *EngineService) Restart(ctx context.Context, name string) (*EngineFullStatus, error) {
	restartCmd := s.registry.GetCommand("engine.restart")
	if restartCmd == nil {
		return nil, unit.ErrCommandNotFound
	}

	result, err := restartCmd.Execute(ctx, map[string]any{"name": name})
	if err != nil {
		return nil, fmt.Errorf("restart engine %s: %w", name, err)
	}

	restartResult, ok := result.(map[string]any)
	if !ok {
		return nil, errors.New("invalid restart command result")
	}

	if status, _ := restartResult["status"].(string); status != string(engine.EngineStatusRunning) {
		return nil, fmt.Errorf("engine %s did not reach running status after restart: %s", name, status)
	}

	return s.GetStatus(ctx, name)
}

func (s *EngineService) GetStatus(ctx context.Context, name string) (*EngineFullStatus, error) {
	eng, err := s.store.Get(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get engine %s: %w", name, err)
	}

	status := &EngineFullStatus{
		Engine: eng,
		Health: &HealthStatus{
			Healthy:   eng.Status == engine.EngineStatusRunning,
			Message:   string(eng.Status),
			Timestamp: time.Now().Unix(),
		},
	}

	if eng.Status == engine.EngineStatusRunning {
		features, err := s.provider.GetFeatures(ctx, name)
		if err == nil {
			status.Feature = features
		}
	}

	return status, nil
}

func (s *EngineService) ListAvailable(ctx context.Context) ([]AvailableEngine, error) {
	listQuery := s.registry.GetQuery("engine.list")
	if listQuery == nil {
		return nil, unit.ErrQueryNotFound
	}

	result, err := listQuery.Execute(ctx, map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("list engines: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, errors.New("invalid list query result")
	}

	items, ok := resultMap["items"].([]map[string]any)
	if !ok {
		if itemsRaw, ok := resultMap["items"].([]any); ok {
			items = make([]map[string]any, len(itemsRaw))
			for i, item := range itemsRaw {
				if m, ok := item.(map[string]any); ok {
					items[i] = m
				}
			}
		} else {
			return []AvailableEngine{}, nil
		}
	}

	engines := make([]AvailableEngine, 0, len(items))
	for _, item := range items {
		eng := AvailableEngine{
			Name:    getString(item, "name"),
			Type:    engine.EngineType(getString(item, "type")),
			Status:  engine.EngineStatus(getString(item, "status")),
			Version: getString(item, "version"),
		}
		engines = append(engines, eng)
	}

	return engines, nil
}

func (s *EngineService) InstallEngine(ctx context.Context, name string, version string) (*InstallEngineResult, error) {
	installCmd := s.registry.GetCommand("engine.install")
	if installCmd == nil {
		return nil, unit.ErrCommandNotFound
	}

	input := map[string]any{
		"name": name,
	}
	if version != "" {
		input["version"] = version
	}

	result, err := installCmd.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("install engine %s: %w", name, err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, errors.New("invalid install command result")
	}

	success, _ := resultMap["success"].(bool)
	path, _ := resultMap["path"].(string)

	if !success {
		return &InstallEngineResult{
			Success: false,
			Message: "installation reported failure",
		}, ErrInstallFailed
	}

	return &InstallEngineResult{
		Success: true,
		Path:    path,
		Message: fmt.Sprintf("engine %s installed successfully", name),
	}, nil
}

func (s *EngineService) ForceStop(ctx context.Context, name string) error {
	stopCmd := s.registry.GetCommand("engine.stop")
	if stopCmd == nil {
		return unit.ErrCommandNotFound
	}

	_, err := stopCmd.Execute(ctx, map[string]any{
		"name":  name,
		"force": true,
	})
	if err != nil {
		return fmt.Errorf("force stop engine %s: %w", name, err)
	}

	return nil
}

func (s *EngineService) StartWithConfig(ctx context.Context, name string, config map[string]any) (map[string]any, error) {
	startCmd := s.registry.GetCommand("engine.start")
	if startCmd == nil {
		return nil, unit.ErrCommandNotFound
	}

	input := map[string]any{
		"name": name,
	}
	if config != nil {
		input["config"] = config
	}

	result, err := startCmd.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("start engine %s with config: %w", name, err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, errors.New("invalid start command result")
	}

	return resultMap, nil
}

func (s *EngineService) GetFeatures(ctx context.Context, name string) (*engine.EngineFeatures, error) {
	featuresQuery := s.registry.GetQuery("engine.features")
	if featuresQuery == nil {
		return nil, unit.ErrQueryNotFound
	}

	result, err := featuresQuery.Execute(ctx, map[string]any{"name": name})
	if err != nil {
		return nil, fmt.Errorf("get features for engine %s: %w", name, err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, errors.New("invalid features query result")
	}

	features := &engine.EngineFeatures{
		SupportsStreaming:    getBool(resultMap, "supports_streaming"),
		SupportsBatch:        getBool(resultMap, "supports_batch"),
		SupportsMultimodal:   getBool(resultMap, "supports_multimodal"),
		SupportsTools:        getBool(resultMap, "supports_tools"),
		SupportsEmbedding:    getBool(resultMap, "supports_embedding"),
		MaxConcurrent:        getInt(resultMap, "max_concurrent"),
		MaxContextLength:     getInt(resultMap, "max_context_length"),
		MaxBatchSize:         getInt(resultMap, "max_batch_size"),
		SupportsGPULayers:    getBool(resultMap, "supports_gpu_layers"),
		SupportsQuantization: getBool(resultMap, "supports_quantization"),
	}

	return features, nil
}

func (s *EngineService) IsHealthy(ctx context.Context, name string) (*HealthStatus, error) {
	eng, err := s.store.Get(ctx, name)
	if err != nil {
		return &HealthStatus{
			Healthy:   false,
			Message:   fmt.Sprintf("engine not found: %s", name),
			Timestamp: time.Now().Unix(),
		}, nil
	}

	healthy := eng.Status == engine.EngineStatusRunning
	message := string(eng.Status)
	if !healthy {
		message = fmt.Sprintf("engine is %s", eng.Status)
	}

	return &HealthStatus{
		Healthy:   healthy,
		Message:   message,
		Timestamp: time.Now().Unix(),
	}, nil
}

func (s *EngineService) waitForHealthy(ctx context.Context, name string) *HealthStatus {
	ticker := time.NewTicker(s.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return &HealthStatus{
				Healthy:   false,
				Message:   "health check timed out",
				Timestamp: time.Now().Unix(),
			}
		case <-ticker.C:
			health, _ := s.IsHealthy(ctx, name)
			if health.Healthy {
				return health
			}
		}
	}
}
