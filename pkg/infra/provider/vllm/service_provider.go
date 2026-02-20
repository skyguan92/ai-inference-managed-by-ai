package vllm

import (
	"context"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
)

// ServiceProvider implements service.ServiceProvider interface for vLLM
type ServiceProvider struct {
	provider *Provider
}

// NewServiceProvider creates a new vLLM service provider
func NewServiceProvider(modelStore interface{}) *ServiceProvider {
	var store model.ModelStore
	if s, ok := modelStore.(model.ModelStore); ok {
		store = s
	}
	return &ServiceProvider{
		provider: NewProvider(store),
	}
}

// Create creates a new vLLM service
func (s *ServiceProvider) Create(ctx context.Context, modelID string, resourceClass service.ResourceClass, replicas int, persistent bool) (*service.ModelService, error) {
	return s.provider.Create(ctx, modelID, resourceClass, replicas, persistent)
}

// Start starts the vLLM service
func (s *ServiceProvider) Start(ctx context.Context, serviceID string) error {
	return s.provider.Start(ctx, serviceID)
}

// Stop stops the vLLM service
func (s *ServiceProvider) Stop(ctx context.Context, serviceID string, force bool) error {
	return s.provider.Stop(ctx, serviceID, force)
}

// Scale scales the service
func (s *ServiceProvider) Scale(ctx context.Context, serviceID string, replicas int) error {
	return s.provider.Scale(ctx, serviceID, replicas)
}

// GetMetrics returns service metrics
func (s *ServiceProvider) GetMetrics(ctx context.Context, serviceID string) (*service.ServiceMetrics, error) {
	return s.provider.GetMetrics(ctx, serviceID)
}

// GetRecommendation provides resource recommendations
func (s *ServiceProvider) GetRecommendation(ctx context.Context, modelID string, hint string) (*service.Recommendation, error) {
	return s.provider.GetRecommendation(ctx, modelID, hint)
}

// SetGPUDevices configures GPU devices for a service
func (s *ServiceProvider) SetGPUDevices(serviceID string, gpus []int) error {
	return s.provider.SetGPUDevices(serviceID, gpus)
}

// GetServiceInfo returns service runtime information
func (s *ServiceProvider) GetServiceInfo(serviceID string) (*ServiceInfo, error) {
	return s.provider.GetServiceInfo(serviceID)
}

// IsRunning checks if the service is actually running
func (s *ServiceProvider) IsRunning(ctx context.Context, serviceID string) bool {
	return s.provider.isRunning(serviceID)
}

// Ensure ServiceProvider implements the interface
var _ service.ServiceProvider = (*ServiceProvider)(nil)
