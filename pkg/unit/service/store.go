package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Domain errors are defined in errors.go

type ServiceStore interface {
	Create(ctx context.Context, service *ModelService) error
	Get(ctx context.Context, id string) (*ModelService, error)
	GetByName(ctx context.Context, name string) (*ModelService, error)
	List(ctx context.Context, filter ServiceFilter) ([]ModelService, int, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, service *ModelService) error
}

type MemoryStore struct {
	services map[string]*ModelService
	mu       sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		services: make(map[string]*ModelService),
	}
}

func (s *MemoryStore) Create(ctx context.Context, service *ModelService) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.services[service.ID]; exists {
		return ErrServiceAlreadyExists
	}

	s.services[service.ID] = service
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, id string) (*ModelService, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	service, exists := s.services[id]
	if !exists {
		return nil, ErrServiceNotFound
	}
	return service, nil
}

func (s *MemoryStore) GetByName(ctx context.Context, name string) (*ModelService, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, svc := range s.services {
		if svc.Name == name {
			return svc, nil
		}
	}
	return nil, ErrServiceNotFound
}

func (s *MemoryStore) List(ctx context.Context, filter ServiceFilter) ([]ModelService, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []ModelService
	for _, svc := range s.services {
		if filter.Status != "" && svc.Status != filter.Status {
			continue
		}
		if filter.ModelID != "" && svc.ModelID != filter.ModelID {
			continue
		}
		result = append(result, *svc)
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

	if _, exists := s.services[id]; !exists {
		return ErrServiceNotFound
	}

	delete(s.services, id)
	return nil
}

func (s *MemoryStore) Update(ctx context.Context, service *ModelService) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.services[service.ID]; !exists {
		return ErrServiceNotFound
	}

	s.services[service.ID] = service
	return nil
}

type ServiceProvider interface {
	Create(ctx context.Context, modelID string, resourceClass ResourceClass, replicas int, persistent bool) (*ModelService, error)
	Start(ctx context.Context, serviceID string) error
	Stop(ctx context.Context, serviceID string, force bool) error
	Scale(ctx context.Context, serviceID string, replicas int) error
	GetMetrics(ctx context.Context, serviceID string) (*ServiceMetrics, error)
	GetRecommendation(ctx context.Context, modelID string, hint string) (*Recommendation, error)
	// IsRunning checks if the service container/process is actually running
	IsRunning(ctx context.Context, serviceID string) bool
	// GetLogs returns the last tail lines of container/process logs for the service
	GetLogs(ctx context.Context, serviceID string, tail int) (string, error)
}

type MockProvider struct {
	createErr      error
	startErr       error
	stopErr        error
	scaleErr       error
	metricsErr     error
	recommendErr   error
	metrics        *ServiceMetrics
	recommendation *Recommendation
}

func (m *MockProvider) Create(ctx context.Context, modelID string, resourceClass ResourceClass, replicas int, persistent bool) (*ModelService, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	now := time.Now().Unix()
	return &ModelService{
		ID:            "svc-" + uuid.New().String()[:8],
		ModelID:       modelID,
		Status:        ServiceStatusRunning,
		Replicas:      replicas,
		ResourceClass: resourceClass,
		Endpoints:     []string{"http://localhost:8080"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func (m *MockProvider) Start(ctx context.Context, serviceID string) error {
	return m.startErr
}

func (m *MockProvider) Stop(ctx context.Context, serviceID string, force bool) error {
	return m.stopErr
}

func (m *MockProvider) Scale(ctx context.Context, serviceID string, replicas int) error {
	return m.scaleErr
}

func (m *MockProvider) GetMetrics(ctx context.Context, serviceID string) (*ServiceMetrics, error) {
	if m.metricsErr != nil {
		return nil, m.metricsErr
	}
	if m.metrics != nil {
		return m.metrics, nil
	}
	return &ServiceMetrics{
		RequestsPerSecond: 100.0,
		LatencyP50:        50.0,
		LatencyP99:        200.0,
		TotalRequests:     10000,
		ErrorRate:         0.01,
	}, nil
}

func (m *MockProvider) GetRecommendation(ctx context.Context, modelID string, hint string) (*Recommendation, error) {
	if m.recommendErr != nil {
		return nil, m.recommendErr
	}
	if m.recommendation != nil {
		return m.recommendation, nil
	}
	return &Recommendation{
		ResourceClass:      ResourceClassMedium,
		Replicas:           2,
		ExpectedThroughput: 100.0,
	}, nil
}

func (m *MockProvider) IsRunning(ctx context.Context, serviceID string) bool {
	return true
}

func (m *MockProvider) GetLogs(ctx context.Context, serviceID string, tail int) (string, error) {
	return fmt.Sprintf("mock logs for service %s (last %d lines)", serviceID, tail), nil
}

func createTestService(id string, modelID string, status ServiceStatus) *ModelService {
	now := time.Now().Unix()
	return &ModelService{
		ID:            id,
		Name:          "test-service-" + id,
		ModelID:       modelID,
		Status:        status,
		Replicas:      1,
		ResourceClass: ResourceClassMedium,
		Endpoints:     []string{"http://localhost:8080"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func toInt(v any) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int32:
		return int(val), true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	case float32:
		return int(val), true
	default:
		return 0, false
	}
}
