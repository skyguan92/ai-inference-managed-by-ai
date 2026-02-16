package engine

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEngineNotFound       = errors.New("engine not found")
	ErrInvalidEngineName    = errors.New("invalid engine name")
	ErrInvalidInput         = errors.New("invalid input")
	ErrEngineAlreadyExists  = errors.New("engine already exists")
	ErrProviderNotSet       = errors.New("engine provider not set")
	ErrEngineNotRunning     = errors.New("engine not running")
	ErrEngineAlreadyRunning = errors.New("engine already running")
)

type EngineStore interface {
	Create(ctx context.Context, engine *Engine) error
	Get(ctx context.Context, name string) (*Engine, error)
	GetByID(ctx context.Context, id string) (*Engine, error)
	List(ctx context.Context, filter EngineFilter) ([]Engine, int, error)
	Delete(ctx context.Context, name string) error
	Update(ctx context.Context, engine *Engine) error
}

type MemoryStore struct {
	engines map[string]*Engine
	mu      sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		engines: make(map[string]*Engine),
	}
}

func (s *MemoryStore) Create(ctx context.Context, engine *Engine) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.engines[engine.Name]; exists {
		return ErrEngineAlreadyExists
	}

	s.engines[engine.Name] = engine
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, name string) (*Engine, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	engine, exists := s.engines[name]
	if !exists {
		return nil, ErrEngineNotFound
	}
	return engine, nil
}

func (s *MemoryStore) GetByID(ctx context.Context, id string) (*Engine, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, engine := range s.engines {
		if engine.ID == id {
			return engine, nil
		}
	}
	return nil, ErrEngineNotFound
}

func (s *MemoryStore) List(ctx context.Context, filter EngineFilter) ([]Engine, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Engine
	for _, e := range s.engines {
		if filter.Type != "" && e.Type != filter.Type {
			continue
		}
		if filter.Status != "" && e.Status != filter.Status {
			continue
		}
		result = append(result, *e)
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

func (s *MemoryStore) Delete(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.engines[name]; !exists {
		return ErrEngineNotFound
	}

	delete(s.engines, name)
	return nil
}

func (s *MemoryStore) Update(ctx context.Context, engine *Engine) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.engines[engine.Name]; !exists {
		return ErrEngineNotFound
	}

	s.engines[engine.Name] = engine
	return nil
}

type EngineProvider interface {
	Start(ctx context.Context, name string, config map[string]any) (*StartResult, error)
	Stop(ctx context.Context, name string, force bool, timeout int) (*StopResult, error)
	Install(ctx context.Context, name string, version string) (*InstallResult, error)
	GetFeatures(ctx context.Context, name string) (*EngineFeatures, error)
}

type MockProvider struct {
	startErr      error
	stopErr       error
	installErr    error
	features      *EngineFeatures
	featuresErr   error
	startResult   *StartResult
	stopResult    *StopResult
	installResult *InstallResult
}

func (m *MockProvider) Start(ctx context.Context, name string, config map[string]any) (*StartResult, error) {
	if m.startErr != nil {
		return nil, m.startErr
	}
	if m.startResult != nil {
		return m.startResult, nil
	}
	return &StartResult{
		ProcessID: "proc-" + uuid.New().String()[:8],
		Status:    EngineStatusRunning,
	}, nil
}

func (m *MockProvider) Stop(ctx context.Context, name string, force bool, timeout int) (*StopResult, error) {
	if m.stopErr != nil {
		return nil, m.stopErr
	}
	if m.stopResult != nil {
		return m.stopResult, nil
	}
	return &StopResult{Success: true}, nil
}

func (m *MockProvider) Install(ctx context.Context, name string, version string) (*InstallResult, error) {
	if m.installErr != nil {
		return nil, m.installErr
	}
	if m.installResult != nil {
		return m.installResult, nil
	}
	return &InstallResult{
		Success: true,
		Path:    "/usr/local/bin/" + name,
	}, nil
}

func (m *MockProvider) GetFeatures(ctx context.Context, name string) (*EngineFeatures, error) {
	if m.featuresErr != nil {
		return nil, m.featuresErr
	}
	if m.features != nil {
		return m.features, nil
	}
	return &EngineFeatures{
		SupportsStreaming:    true,
		SupportsBatch:        true,
		SupportsMultimodal:   false,
		SupportsTools:        true,
		SupportsEmbedding:    true,
		MaxConcurrent:        10,
		MaxContextLength:     8192,
		MaxBatchSize:         32,
		SupportsGPULayers:    true,
		SupportsQuantization: true,
	}, nil
}

func createTestEngine(name string, engineType EngineType) *Engine {
	now := time.Now().Unix()
	return &Engine{
		ID:           "engine-" + uuid.New().String()[:8],
		Name:         name,
		Type:         engineType,
		Status:       EngineStatusStopped,
		Version:      "1.0.0",
		CreatedAt:    now,
		UpdatedAt:    now,
		Models:       []string{},
		Capabilities: []string{"chat", "completion"},
	}
}
