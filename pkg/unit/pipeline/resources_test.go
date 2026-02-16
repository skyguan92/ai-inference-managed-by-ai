package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestPipelineResource_URI(t *testing.T) {
	r := NewPipelineResource("pipe-123", nil)
	expected := "asms://pipeline/pipe-123"
	if r.URI() != expected {
		t.Errorf("expected URI '%s', got '%s'", expected, r.URI())
	}
}

func TestPipelineResource_Domain(t *testing.T) {
	r := NewPipelineResource("pipe-123", nil)
	if r.Domain() != "pipeline" {
		t.Errorf("expected domain 'pipeline', got '%s'", r.Domain())
	}
}

func TestPipelineResource_Get(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		store   PipelineStore
		wantErr bool
	}{
		{
			name:    "successful get",
			id:      "pipe-123",
			store:   createStoreWithPipeline("pipe-123", PipelineStatusIdle),
			wantErr: false,
		},
		{
			name:    "nil store",
			id:      "pipe-123",
			store:   nil,
			wantErr: true,
		},
		{
			name:    "pipeline not found",
			id:      "nonexistent",
			store:   NewMemoryStore(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewPipelineResource(tt.id, tt.store)
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

			if _, exists := resultMap["id"]; !exists {
				t.Error("expected field 'id' not found")
			}
			if _, exists := resultMap["steps"]; !exists {
				t.Error("expected field 'steps' not found")
			}
		})
	}
}

func TestPipelineResource_Watch(t *testing.T) {
	store := createStoreWithPipeline("pipe-123", PipelineStatusIdle)
	r := NewPipelineResource("pipe-123", store)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Errorf("unexpected error from Watch: %v", err)
		return
	}

	select {
	case update, ok := <-ch:
		if ok && update.URI != r.URI() {
			t.Errorf("expected URI '%s', got '%s'", r.URI(), update.URI)
		}
	case <-ctx.Done():
	}
}

func TestPipelinesResource_URI(t *testing.T) {
	r := NewPipelinesResource(nil)
	expected := "asms://pipelines"
	if r.URI() != expected {
		t.Errorf("expected URI '%s', got '%s'", expected, r.URI())
	}
}

func TestPipelinesResource_Domain(t *testing.T) {
	r := NewPipelinesResource(nil)
	if r.Domain() != "pipeline" {
		t.Errorf("expected domain 'pipeline', got '%s'", r.Domain())
	}
}

func TestPipelinesResource_Get(t *testing.T) {
	tests := []struct {
		name    string
		store   PipelineStore
		wantErr bool
	}{
		{
			name:    "successful get",
			store:   createStoreWithMultiplePipelines(),
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
			r := NewPipelinesResource(tt.store)
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

			if _, exists := resultMap["pipelines"]; !exists {
				t.Error("expected field 'pipelines' not found")
			}
			if _, exists := resultMap["total"]; !exists {
				t.Error("expected field 'total' not found")
			}
		})
	}
}

func TestPipelinesResource_Watch(t *testing.T) {
	store := createStoreWithMultiplePipelines()
	r := NewPipelinesResource(store)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Errorf("unexpected error from Watch: %v", err)
		return
	}

	select {
	case update, ok := <-ch:
		if ok && update.URI != r.URI() {
			t.Errorf("expected URI '%s', got '%s'", r.URI(), update.URI)
		}
	case <-ctx.Done():
	}
}

func TestParsePipelineResourceURI(t *testing.T) {
	tests := []struct {
		uri      string
		expected string
		ok       bool
	}{
		{"asms://pipeline/pipe-123", "pipe-123", true},
		{"asms://pipeline/", "", false},
		{"asms://pipelines", "", false},
		{"asms://service/svc-123", "", false},
		{"invalid", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			id, ok := ParsePipelineResourceURI(tt.uri)
			if ok != tt.ok {
				t.Errorf("expected ok=%v, got %v", tt.ok, ok)
			}
			if id != tt.expected {
				t.Errorf("expected id='%s', got '%s'", tt.expected, id)
			}
		})
	}
}

func TestResourceImplementsInterface(t *testing.T) {
	var _ unit.Resource = NewPipelineResource("test", nil)
	var _ unit.Resource = NewPipelinesResource(nil)
}
