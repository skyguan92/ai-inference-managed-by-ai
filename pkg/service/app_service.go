package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/eventbus"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/app"
)

type AppService struct {
	registry *unit.Registry
	store    app.AppStore
	provider app.AppProvider
	bus      *eventbus.InMemoryEventBus
}

func NewAppService(registry *unit.Registry, store app.AppStore, provider app.AppProvider, bus *eventbus.InMemoryEventBus) *AppService {
	return &AppService{
		registry: registry,
		store:    store,
		provider: provider,
		bus:      bus,
	}
}

type InstallWithVerifyResult struct {
	App        *app.App
	Healthy    bool
	HealthInfo map[string]any
}

func (s *AppService) InstallWithVerify(ctx context.Context, template, name string, config map[string]any) (*InstallWithVerifyResult, error) {
	installCmd := s.registry.GetCommand("app.install")
	if installCmd == nil {
		return nil, fmt.Errorf("app.install command not found")
	}

	input := map[string]any{
		"template": template,
	}
	if name != "" {
		input["name"] = name
	}
	if config != nil {
		input["config"] = config
	}

	result, err := installCmd.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("install app: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected install result type")
	}

	appID, _ := resultMap["app_id"].(string)
	if appID == "" {
		return nil, fmt.Errorf("app_id not found in install result")
	}

	installedApp, err := s.store.Get(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("get installed app %s: %w", appID, err)
	}

	startCmd := s.registry.GetCommand("app.start")
	if startCmd != nil {
		_, startErr := startCmd.Execute(ctx, map[string]any{"app_id": appID})
		if startErr != nil {
			_ = s.rollbackUninstall(ctx, appID)
			return nil, fmt.Errorf("start app after install: %w", startErr)
		}

		healthy, healthInfo := s.checkAppHealth(ctx, appID)
		if !healthy {
			_ = s.rollbackUninstall(ctx, appID)
			return nil, fmt.Errorf("app health check failed after install")
		}

		installedApp, err = s.store.Get(ctx, appID)
		if err != nil {
			_ = s.rollbackUninstall(ctx, appID)
			return nil, fmt.Errorf("get installed app %s after health check: %w", appID, err)
		}

		s.publishAppEvent(ctx, "app.installed_and_verified", map[string]any{
			"app_id":   appID,
			"template": template,
			"healthy":  healthy,
		})

		return &InstallWithVerifyResult{
			App:        installedApp,
			Healthy:    healthy,
			HealthInfo: healthInfo,
		}, nil
	}

	s.publishAppEvent(ctx, "app.installed", map[string]any{
		"app_id":   appID,
		"template": template,
	})

	return &InstallWithVerifyResult{
		App:        installedApp,
		Healthy:    false,
		HealthInfo: nil,
	}, nil
}

type UninstallWithCleanupResult struct {
	Success      bool
	RemovedData  bool
	CleanedItems []string
}

func (s *AppService) UninstallWithCleanup(ctx context.Context, appID string, removeData bool) (*UninstallWithCleanupResult, error) {
	a, err := s.store.Get(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("get app %s: %w", appID, err)
	}

	var cleanedItems []string

	if a.Status == app.AppStatusRunning {
		stopCmd := s.registry.GetCommand("app.stop")
		if stopCmd != nil {
			_, stopErr := stopCmd.Execute(ctx, map[string]any{
				"app_id":  appID,
				"timeout": 30,
			})
			if stopErr != nil {
				return nil, fmt.Errorf("stop app before uninstall: %w", stopErr)
			}
			cleanedItems = append(cleanedItems, "stopped_container")
		}
	}

	if len(a.Volumes) > 0 {
		cleanedItems = append(cleanedItems, a.Volumes...)
	}

	uninstallCmd := s.registry.GetCommand("app.uninstall")
	if uninstallCmd == nil {
		return nil, fmt.Errorf("app.uninstall command not found")
	}

	input := map[string]any{
		"app_id":      appID,
		"remove_data": removeData,
	}

	_, err = uninstallCmd.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("uninstall app: %w", err)
	}

	s.publishAppEvent(ctx, "app.uninstalled_with_cleanup", map[string]any{
		"app_id":        appID,
		"removed_data":  removeData,
		"cleaned_items": cleanedItems,
	})

	return &UninstallWithCleanupResult{
		Success:      true,
		RemovedData:  removeData,
		CleanedItems: cleanedItems,
	}, nil
}

type StartWithHealthCheckResult struct {
	Success    bool
	Healthy    bool
	HealthInfo map[string]any
}

func (s *AppService) StartWithHealthCheck(ctx context.Context, appID string) (*StartWithHealthCheckResult, error) {
	a, err := s.store.Get(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("get app %s: %w", appID, err)
	}

	if a.Status == app.AppStatusRunning {
		healthy, healthInfo := s.checkAppHealth(ctx, appID)
		return &StartWithHealthCheckResult{
			Success:    true,
			Healthy:    healthy,
			HealthInfo: healthInfo,
		}, nil
	}

	startCmd := s.registry.GetCommand("app.start")
	if startCmd == nil {
		return nil, fmt.Errorf("app.start command not found")
	}

	_, err = startCmd.Execute(ctx, map[string]any{"app_id": appID})
	if err != nil {
		return nil, fmt.Errorf("start app: %w", err)
	}

	healthy, healthInfo := s.checkAppHealth(ctx, appID)

	s.publishAppEvent(ctx, "app.started_with_health_check", map[string]any{
		"app_id":  appID,
		"healthy": healthy,
	})

	return &StartWithHealthCheckResult{
		Success:    true,
		Healthy:    healthy,
		HealthInfo: healthInfo,
	}, nil
}

type StopGracefullyResult struct {
	Success  bool
	Timeout  int
	WaitTime int64
}

func (s *AppService) StopGracefully(ctx context.Context, appID string, timeout int) (*StopGracefullyResult, error) {
	a, err := s.store.Get(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("get app %s: %w", appID, err)
	}

	if a.Status != app.AppStatusRunning {
		return &StopGracefullyResult{
			Success:  true,
			Timeout:  timeout,
			WaitTime: 0,
		}, nil
	}

	if timeout <= 0 {
		timeout = 30
	}

	stopCmd := s.registry.GetCommand("app.stop")
	if stopCmd == nil {
		return nil, fmt.Errorf("app.stop command not found")
	}

	startTime := time.Now()

	_, err = stopCmd.Execute(ctx, map[string]any{
		"app_id":  appID,
		"timeout": timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("stop app: %w", err)
	}

	waitTime := time.Since(startTime).Milliseconds()

	s.publishAppEvent(ctx, "app.stopped_gracefully", map[string]any{
		"app_id":    appID,
		"timeout":   timeout,
		"wait_time": waitTime,
	})

	return &StopGracefullyResult{
		Success:  true,
		Timeout:  timeout,
		WaitTime: waitTime,
	}, nil
}

type FullAppInfo struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Template  string          `json:"template"`
	Status    app.AppStatus   `json:"status"`
	Ports     []int           `json:"ports,omitempty"`
	Volumes   []string        `json:"volumes,omitempty"`
	Config    map[string]any  `json:"config,omitempty"`
	Metrics   *app.AppMetrics `json:"metrics,omitempty"`
	CreatedAt int64           `json:"created_at"`
	UpdatedAt int64           `json:"updated_at"`
	Health    *AppHealthInfo  `json:"health,omitempty"`
}

type AppHealthInfo struct {
	Healthy   bool   `json:"healthy"`
	Status    string `json:"status"`
	LastCheck int64  `json:"last_check"`
}

func (s *AppService) GetFullInfo(ctx context.Context, appID string) (*FullAppInfo, error) {
	getQuery := s.registry.GetQuery("app.get")
	if getQuery == nil {
		return nil, fmt.Errorf("app.get query not found")
	}

	result, err := getQuery.Execute(ctx, map[string]any{"app_id": appID})
	if err != nil {
		return nil, fmt.Errorf("get app: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected get result type")
	}

	info := &FullAppInfo{
		ID:        getString(resultMap, "id"),
		Name:      getString(resultMap, "name"),
		Template:  getString(resultMap, "template"),
		Status:    app.AppStatus(getString(resultMap, "status")),
		CreatedAt: getInt64(resultMap, "created_at"),
		UpdatedAt: getInt64(resultMap, "updated_at"),
	}

	if ports, ok := resultMap["ports"].([]int); ok {
		info.Ports = ports
	} else if portsAny, ok := resultMap["ports"].([]any); ok {
		info.Ports = make([]int, len(portsAny))
		for i, p := range portsAny {
			switch v := p.(type) {
			case int:
				info.Ports[i] = v
			case float64:
				info.Ports[i] = int(v)
			}
		}
	}

	if volumes, ok := resultMap["volumes"].([]string); ok {
		info.Volumes = volumes
	} else if volumesAny, ok := resultMap["volumes"].([]any); ok {
		info.Volumes = make([]string, 0, len(volumesAny))
		for _, v := range volumesAny {
			if s, ok := v.(string); ok {
				info.Volumes = append(info.Volumes, s)
			}
		}
	}

	if config, ok := resultMap["config"].(map[string]any); ok {
		info.Config = config
	}

	if metricsMap, ok := resultMap["metrics"].(map[string]any); ok {
		info.Metrics = &app.AppMetrics{
			CPUUsage:    getFloat64(metricsMap, "cpu_usage"),
			MemoryUsage: getFloat64(metricsMap, "memory_usage"),
			Uptime:      getInt64(metricsMap, "uptime"),
		}
	}

	if info.Status == app.AppStatusRunning {
		healthy, healthInfo := s.checkAppHealth(ctx, appID)
		info.Health = &AppHealthInfo{
			Healthy:   healthy,
			LastCheck: time.Now().Unix(),
		}
		if healthInfo != nil {
			if status, ok := healthInfo["status"].(string); ok {
				info.Health.Status = status
			}
		}
	}

	return info, nil
}

func (s *AppService) ListByStatus(ctx context.Context, status app.AppStatus) ([]app.App, int, error) {
	filter := app.AppFilter{
		Status: status,
		Limit:  1000,
	}

	return s.store.List(ctx, filter)
}

type LogEntry struct {
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
	Level     string `json:"level"`
}

func (s *AppService) GetLogsWithTail(ctx context.Context, appID string, tail int, since int64) ([]LogEntry, error) {
	logsQuery := s.registry.GetQuery("app.logs")
	if logsQuery == nil {
		return nil, fmt.Errorf("app.logs query not found")
	}

	if tail <= 0 {
		tail = 100
	}

	input := map[string]any{
		"app_id": appID,
		"tail":   tail,
	}
	if since > 0 {
		input["since"] = since
	}

	result, err := logsQuery.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("get logs: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected logs result type")
	}

	logsAny, ok := resultMap["logs"].([]any)
	if !ok {
		return nil, nil
	}

	logs := make([]LogEntry, 0, len(logsAny))
	for _, l := range logsAny {
		if logMap, ok := l.(map[string]any); ok {
			logs = append(logs, LogEntry{
				Timestamp: getInt64(logMap, "timestamp"),
				Message:   getString(logMap, "message"),
				Level:     getString(logMap, "level"),
			})
		}
	}

	return logs, nil
}

type TemplateInfo struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Category    app.AppCategory `json:"category"`
	Description string          `json:"description"`
	Image       string          `json:"image"`
}

func (s *AppService) ListTemplatesByCategory(ctx context.Context, category app.AppCategory) ([]TemplateInfo, error) {
	templatesQuery := s.registry.GetQuery("app.templates")
	if templatesQuery == nil {
		return nil, fmt.Errorf("app.templates query not found")
	}

	input := map[string]any{}
	if category != "" {
		input["category"] = string(category)
	}

	result, err := templatesQuery.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("get templates: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected templates result type")
	}

	templatesAny, ok := resultMap["templates"].([]any)
	if !ok {
		return nil, nil
	}

	templates := make([]TemplateInfo, 0, len(templatesAny))
	for _, t := range templatesAny {
		if tMap, ok := t.(map[string]any); ok {
			templates = append(templates, TemplateInfo{
				ID:          getString(tMap, "id"),
				Name:        getString(tMap, "name"),
				Category:    app.AppCategory(getString(tMap, "category")),
				Description: getString(tMap, "description"),
				Image:       getString(tMap, "image"),
			})
		}
	}

	return templates, nil
}

func (s *AppService) checkAppHealth(ctx context.Context, appID string) (bool, map[string]any) {
	if s.provider == nil {
		return false, nil
	}

	metrics, err := s.provider.GetMetrics(ctx, appID)
	if err != nil {
		return false, map[string]any{
			"healthy": false,
			"status":  "error",
			"error":   err.Error(),
		}
	}

	healthy := metrics != nil && metrics.Uptime > 0

	var status string
	if healthy {
		status = "healthy"
	} else {
		status = "unhealthy"
	}

	return healthy, map[string]any{
		"healthy":      healthy,
		"status":       status,
		"cpu_usage":    metrics.CPUUsage,
		"memory_usage": metrics.MemoryUsage,
		"uptime":       metrics.Uptime,
	}
}

func (s *AppService) rollbackUninstall(ctx context.Context, appID string) error {
	uninstallCmd := s.registry.GetCommand("app.uninstall")
	if uninstallCmd == nil {
		return fmt.Errorf("app.uninstall command not found")
	}

	_, err := uninstallCmd.Execute(ctx, map[string]any{
		"app_id":      appID,
		"remove_data": true,
	})
	return err
}

func (s *AppService) publishAppEvent(ctx context.Context, eventType string, payload any) {
	if s.bus == nil {
		return
	}

	evt := &appSimpleEvent{
		eventType: eventType,
		domain:    "app",
		payload:   payload,
	}

	_ = s.bus.Publish(evt)
}

type appSimpleEvent struct {
	eventType string
	domain    string
	payload   any
}

func (e *appSimpleEvent) Type() string          { return e.eventType }
func (e *appSimpleEvent) Domain() string        { return e.domain }
func (e *appSimpleEvent) Payload() any          { return e.payload }
func (e *appSimpleEvent) Timestamp() time.Time  { return time.Now() }
func (e *appSimpleEvent) CorrelationID() string { return "" }
