package service

import (
	"context"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestServiceResource_URI(t *testing.T) {
	r := NewServiceResource("svc-123", nil, nil)
	expected := "asms://service/svc-123"
	if r.URI() != expected {
		t.Errorf("expected URI '%s', got '%s'", expected, r.URI())
	}
}

func TestServiceResource_Domain(t *testing.T) {
	r := NewServiceResource("svc-123", nil, nil)
	if r.Domain() != "service" {
		t.Errorf("expected domain 'service', got '%s'", r.Domain())
	}
}

func TestServiceResource_Schema(t *testing.T) {
	r := NewServiceResource("svc-123", nil, nil)
	schema := r.Schema()

	if schema.Type != "object" {
		t.Errorf("expected schema type 'object', got '%s'", schema.Type)
	}
	if len(schema.Properties) == 0 {
		t.Error("expected schema to have properties")
	}
}

func TestServiceResource_Get(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		store    ServiceStore
		provider ServiceProvider
		wantErr  bool
	}{
		{
			name:     "successful get",
			id:       "svc-123",
			store:    createStoreWithService("svc-123", "model-1", ServiceStatusRunning),
			provider: &MockProvider{},
			wantErr:  false,
		},
		{
			name:     "get without provider",
			id:       "svc-123",
			store:    createStoreWithService("svc-123", "model-1", ServiceStatusRunning),
			provider: nil,
			wantErr:  false,
		},
		{
			name:     "nil store",
			id:       "svc-123",
			store:    nil,
			provider: &MockProvider{},
			wantErr:  true,
		},
		{
			name:     "service not found",
			id:       "nonexistent",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewServiceResource(tt.id, tt.store, tt.provider)
			result, err := r.Get(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Error("expected result to be map[string]any")
				return
			}

			if resultMap["id"] != tt.id {
				t.Errorf("expected id '%s', got '%v'", tt.id, resultMap["id"])
			}
		})
	}
}

func TestServiceResource_Watch(t *testing.T) {
	store := createStoreWithService("svc-123", "model-1", ServiceStatusRunning)
	r := NewServiceResource("svc-123", store, &MockProvider{})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	select {
	case update := <-ch:
		if update.URI != r.URI() {
			t.Errorf("expected URI '%s', got '%s'", r.URI(), update.URI)
		}
	case <-ctx.Done():
	}
}

func TestServicesResource_URI(t *testing.T) {
	r := NewServicesResource(nil)
	expected := "asms://services"
	if r.URI() != expected {
		t.Errorf("expected URI '%s', got '%s'", expected, r.URI())
	}
}

func TestServicesResource_Domain(t *testing.T) {
	r := NewServicesResource(nil)
	if r.Domain() != "service" {
		t.Errorf("expected domain 'service', got '%s'", r.Domain())
	}
}

func TestServicesResource_Schema(t *testing.T) {
	r := NewServicesResource(nil)
	schema := r.Schema()

	if schema.Type != "object" {
		t.Errorf("expected schema type 'object', got '%s'", schema.Type)
	}
	if len(schema.Properties) == 0 {
		t.Error("expected schema to have properties")
	}
}

func TestServicesResource_Get(t *testing.T) {
	tests := []struct {
		name    string
		store   ServiceStore
		wantErr bool
	}{
		{
			name:    "successful get",
			store:   createStoreWithMultipleServices(),
			wantErr: false,
		},
		{
			name:    "empty store",
			store:   NewMemoryStore(),
			wantErr: false,
		},
		{
			name:    "nil store",
			store:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewServicesResource(tt.store)
			result, err := r.Get(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Error("expected result to be map[string]any")
				return
			}

			if _, exists := resultMap["services"]; !exists {
				t.Error("expected field 'services' not found")
			}
			if _, exists := resultMap["total"]; !exists {
				t.Error("expected field 'total' not found")
			}
		})
	}
}

func TestServicesResource_Watch(t *testing.T) {
	store := createStoreWithMultipleServices()
	r := NewServicesResource(store)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	select {
	case update := <-ch:
		if update.URI != r.URI() {
			t.Errorf("expected URI '%s', got '%s'", r.URI(), update.URI)
		}
	case <-ctx.Done():
	}
}

func TestParseServiceResourceURI(t *testing.T) {
	tests := []struct {
		uri    string
		wantID string
		wantOK bool
	}{
		{"asms://service/svc-123", "svc-123", true},
		{"asms://service/", "", false},
		{"asms://services", "", false},
		{"asms://engine/ollama", "", false},
		{"invalid", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			id, ok := ParseServiceResourceURI(tt.uri)
			if ok != tt.wantOK {
				t.Errorf("expected ok=%v, got %v", tt.wantOK, ok)
			}
			if id != tt.wantID {
				t.Errorf("expected id='%s', got '%s'", tt.wantID, id)
			}
		})
	}
}

func TestResourceImplementsInterface(t *testing.T) {
	var _ unit.Resource = NewServiceResource("test", nil, nil)
	var _ unit.Resource = NewServicesResource(nil)
}

func TestServiceResource_GetWithMetrics(t *testing.T) {
	store := createStoreWithService("svc-123", "model-1", ServiceStatusRunning)
	provider := &MockProvider{
		metrics: &ServiceMetrics{
			RequestsPerSecond: 150.0,
			LatencyP50:        45.0,
			LatencyP99:        180.0,
			TotalRequests:     50000,
			ErrorRate:         0.005,
		},
	}

	r := NewServiceResource("svc-123", store, provider)
	result, err := r.Get(context.Background())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	resultMap := result.(map[string]any)
	metrics, ok := resultMap["metrics"].(map[string]any)
	if !ok {
		t.Error("expected metrics in result for running service")
		return
	}

	if metrics["requests_per_second"].(float64) != 150.0 {
		t.Errorf("expected requests_per_second 150.0, got %v", metrics["requests_per_second"])
	}
}

func TestServicesResource_GetWithPagination(t *testing.T) {
	store := NewMemoryStore()
	for i := 0; i < 5; i++ {
		svc := createTestService(string(rune('a'+i)), "model-1", ServiceStatusRunning)
		_ = store.Create(context.Background(), svc)
	}

	r := NewServicesResource(store)
	result, err := r.Get(context.Background())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	resultMap := result.(map[string]any)
	services := resultMap["services"].([]map[string]any)
	total := resultMap["total"].(int)

	if len(services) != 5 {
		t.Errorf("expected 5 services, got %d", len(services))
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
}

func TestServiceResource_WatchContextCancellation(t *testing.T) {
	store := createStoreWithService("svc-123", "model-1", ServiceStatusRunning)
	r := NewServiceResource("svc-123", store, &MockProvider{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed immediately on cancelled context")
	}
}

func TestServiceResource_GetStoppedServiceNoMetrics(t *testing.T) {
	store := createStoreWithService("svc-123", "model-1", ServiceStatusStopped)
	provider := &MockProvider{
		metrics: &ServiceMetrics{
			RequestsPerSecond: 150.0,
		},
	}

	r := NewServiceResource("svc-123", store, provider)
	result, err := r.Get(context.Background())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	resultMap := result.(map[string]any)
	if _, exists := resultMap["metrics"]; exists {
		t.Error("expected no metrics for stopped service")
	}
}
