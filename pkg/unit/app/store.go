package app

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrAppNotFound       = errors.New("app not found")
	ErrInvalidAppID      = errors.New("invalid app id")
	ErrInvalidInput      = errors.New("invalid input")
	ErrAppAlreadyExists  = errors.New("app already exists")
	ErrProviderNotSet    = errors.New("app provider not set")
	ErrAppNotRunning     = errors.New("app not running")
	ErrAppAlreadyRunning = errors.New("app already running")
	ErrTemplateNotFound  = errors.New("template not found")
)

type AppStore interface {
	Create(ctx context.Context, app *App) error
	Get(ctx context.Context, id string) (*App, error)
	List(ctx context.Context, filter AppFilter) ([]App, int, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, app *App) error
}

type MemoryStore struct {
	apps map[string]*App
	mu   sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		apps: make(map[string]*App),
	}
}

func (s *MemoryStore) Create(ctx context.Context, app *App) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.apps[app.ID]; exists {
		return ErrAppAlreadyExists
	}

	s.apps[app.ID] = app
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, id string) (*App, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	app, exists := s.apps[id]
	if !exists {
		return nil, ErrAppNotFound
	}
	return app, nil
}

func (s *MemoryStore) List(ctx context.Context, filter AppFilter) ([]App, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []App
	for _, a := range s.apps {
		if filter.Status != "" && a.Status != filter.Status {
			continue
		}
		if filter.Template != "" && a.Template != filter.Template {
			continue
		}
		result = append(result, *a)
	}

	total := len(result)

	offset := filter.Offset
	if offset > len(result) {
		offset = len(result)
	}

	end := len(result)
	if filter.Limit > 0 {
		end = offset + filter.Limit
		if end > len(result) {
			end = len(result)
		}
	}

	return result[offset:end], total, nil
}

func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.apps[id]; !exists {
		return ErrAppNotFound
	}

	delete(s.apps, id)
	return nil
}

func (s *MemoryStore) Update(ctx context.Context, app *App) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.apps[app.ID]; !exists {
		return ErrAppNotFound
	}

	s.apps[app.ID] = app
	return nil
}

type AppProvider interface {
	Install(ctx context.Context, template string, name string, config map[string]any) (*InstallResult, error)
	Uninstall(ctx context.Context, appID string, removeData bool) (*UninstallResult, error)
	Start(ctx context.Context, appID string) (*StartResult, error)
	Stop(ctx context.Context, appID string, timeout int) (*StopResult, error)
	GetLogs(ctx context.Context, appID string, tail int, since int64) ([]LogEntry, error)
	GetTemplates(ctx context.Context, category AppCategory) ([]Template, error)
	GetMetrics(ctx context.Context, appID string) (*AppMetrics, error)
}

type MockProvider struct {
	installErr      error
	uninstallErr    error
	startErr        error
	stopErr         error
	logsErr         error
	templatesErr    error
	metricsErr      error
	installResult   *InstallResult
	uninstallResult *UninstallResult
	startResult     *StartResult
	stopResult      *StopResult
	logs            []LogEntry
	templates       []Template
	metrics         *AppMetrics
}

func (m *MockProvider) Install(ctx context.Context, template string, name string, config map[string]any) (*InstallResult, error) {
	if m.installErr != nil {
		return nil, m.installErr
	}
	if m.installResult != nil {
		return m.installResult, nil
	}
	return &InstallResult{
		AppID: "app-" + uuid.New().String()[:8],
	}, nil
}

func (m *MockProvider) Uninstall(ctx context.Context, appID string, removeData bool) (*UninstallResult, error) {
	if m.uninstallErr != nil {
		return nil, m.uninstallErr
	}
	if m.uninstallResult != nil {
		return m.uninstallResult, nil
	}
	return &UninstallResult{Success: true}, nil
}

func (m *MockProvider) Start(ctx context.Context, appID string) (*StartResult, error) {
	if m.startErr != nil {
		return nil, m.startErr
	}
	if m.startResult != nil {
		return m.startResult, nil
	}
	return &StartResult{Success: true}, nil
}

func (m *MockProvider) Stop(ctx context.Context, appID string, timeout int) (*StopResult, error) {
	if m.stopErr != nil {
		return nil, m.stopErr
	}
	if m.stopResult != nil {
		return m.stopResult, nil
	}
	return &StopResult{Success: true}, nil
}

func (m *MockProvider) GetLogs(ctx context.Context, appID string, tail int, since int64) ([]LogEntry, error) {
	if m.logsErr != nil {
		return nil, m.logsErr
	}
	if m.logs != nil {
		return m.logs, nil
	}
	return []LogEntry{
		{Timestamp: time.Now().Unix(), Message: "App started", Level: "info"},
		{Timestamp: time.Now().Unix(), Message: "Listening on port 8080", Level: "info"},
	}, nil
}

func (m *MockProvider) GetTemplates(ctx context.Context, category AppCategory) ([]Template, error) {
	if m.templatesErr != nil {
		return nil, m.templatesErr
	}
	if m.templates != nil {
		return m.templates, nil
	}
	return []Template{
		{ID: "open-webui", Name: "Open WebUI", Category: AppCategoryAIChat, Description: "AI Chat Interface", Image: "ghcr.io/open-webui/open-webui:main"},
		{ID: "grafana", Name: "Grafana", Category: AppCategoryMonitoring, Description: "Monitoring Dashboard", Image: "grafana/grafana:latest"},
	}, nil
}

func (m *MockProvider) GetMetrics(ctx context.Context, appID string) (*AppMetrics, error) {
	if m.metricsErr != nil {
		return nil, m.metricsErr
	}
	if m.metrics != nil {
		return m.metrics, nil
	}
	return &AppMetrics{
		CPUUsage:    10.5,
		MemoryUsage: 256.0,
		Uptime:      3600,
	}, nil
}

func createTestApp(id string, template string, status AppStatus) *App {
	now := time.Now().Unix()
	return &App{
		ID:        id,
		Name:      id,
		Template:  template,
		Status:    status,
		Ports:     []int{8080},
		Volumes:   []string{"/data"},
		CreatedAt: now,
		UpdatedAt: now,
	}
}
