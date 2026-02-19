package pipeline

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Domain errors are defined in errors.go

type PipelineStore interface {
	CreatePipeline(ctx context.Context, pipeline *Pipeline) error
	GetPipeline(ctx context.Context, id string) (*Pipeline, error)
	ListPipelines(ctx context.Context, filter PipelineFilter) ([]Pipeline, int, error)
	DeletePipeline(ctx context.Context, id string) error
	UpdatePipeline(ctx context.Context, pipeline *Pipeline) error

	CreateRun(ctx context.Context, run *PipelineRun) error
	GetRun(ctx context.Context, id string) (*PipelineRun, error)
	ListRuns(ctx context.Context, pipelineID string) ([]PipelineRun, error)
	UpdateRun(ctx context.Context, run *PipelineRun) error
}

type MemoryStore struct {
	pipelines map[string]*Pipeline
	runs      map[string]*PipelineRun
	mu        sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		pipelines: make(map[string]*Pipeline),
		runs:      make(map[string]*PipelineRun),
	}
}

func (s *MemoryStore) CreatePipeline(ctx context.Context, pipeline *Pipeline) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.pipelines[pipeline.ID]; exists {
		return ErrPipelineAlreadyExists
	}

	s.pipelines[pipeline.ID] = pipeline
	return nil
}

func (s *MemoryStore) GetPipeline(ctx context.Context, id string) (*Pipeline, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pipeline, exists := s.pipelines[id]
	if !exists {
		return nil, ErrPipelineNotFound
	}
	return pipeline, nil
}

func (s *MemoryStore) ListPipelines(ctx context.Context, filter PipelineFilter) ([]Pipeline, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Pipeline
	for _, p := range s.pipelines {
		if filter.Status != "" && p.Status != filter.Status {
			continue
		}
		result = append(result, *p)
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

func (s *MemoryStore) DeletePipeline(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.pipelines[id]; !exists {
		return ErrPipelineNotFound
	}

	delete(s.pipelines, id)
	return nil
}

func (s *MemoryStore) UpdatePipeline(ctx context.Context, pipeline *Pipeline) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.pipelines[pipeline.ID]; !exists {
		return ErrPipelineNotFound
	}

	s.pipelines[pipeline.ID] = pipeline
	return nil
}

func (s *MemoryStore) CreateRun(ctx context.Context, run *PipelineRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.runs[run.ID]; exists {
		return ErrRunNotFound
	}

	s.runs[run.ID] = run
	return nil
}

func (s *MemoryStore) GetRun(ctx context.Context, id string) (*PipelineRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	run, exists := s.runs[id]
	if !exists {
		return nil, ErrRunNotFound
	}
	return run, nil
}

func (s *MemoryStore) ListRuns(ctx context.Context, pipelineID string) ([]PipelineRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []PipelineRun
	for _, r := range s.runs {
		if pipelineID != "" && r.PipelineID != pipelineID {
			continue
		}
		result = append(result, *r)
	}
	return result, nil
}

func (s *MemoryStore) UpdateRun(ctx context.Context, run *PipelineRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.runs[run.ID]; !exists {
		return ErrRunNotFound
	}

	s.runs[run.ID] = run
	return nil
}

func generateID(prefix string) string {
	return prefix + "-" + uuid.New().String()[:8]
}

func createTestPipeline(id string, name string, status PipelineStatus) *Pipeline {
	now := time.Now().Unix()
	return &Pipeline{
		ID:     id,
		Name:   name,
		Status: status,
		Steps: []PipelineStep{
			{ID: "step1", Name: "Step 1", Type: "inference.chat", Input: map[string]any{"model": "test"}},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func createTestRun(id string, pipelineID string, status RunStatus) *PipelineRun {
	return &PipelineRun{
		ID:          id,
		PipelineID:  pipelineID,
		Status:      status,
		StepResults: make(map[string]any),
		StartedAt:   time.Now(),
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

func ptrInt(v int) *int {
	return &v
}

func ptrFloat(v float64) *float64 {
	return &v
}
