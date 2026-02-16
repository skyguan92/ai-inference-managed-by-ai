package model

import (
	"context"
	"sync"
	"time"
)

type MemoryStore struct {
	models map[string]*Model
	mu     sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		models: make(map[string]*Model),
	}
}

func (s *MemoryStore) Create(ctx context.Context, model *Model) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.models[model.ID]; exists {
		return ErrModelAlreadyExists
	}

	s.models[model.ID] = model
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, id string) (*Model, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	model, exists := s.models[id]
	if !exists {
		return nil, ErrModelNotFound
	}
	return model, nil
}

func (s *MemoryStore) List(ctx context.Context, filter ModelFilter) ([]Model, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Model
	for _, m := range s.models {
		if filter.Type != "" && m.Type != filter.Type {
			continue
		}
		if filter.Status != "" && m.Status != filter.Status {
			continue
		}
		if filter.Format != "" && m.Format != filter.Format {
			continue
		}
		result = append(result, *m)
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

	if _, exists := s.models[id]; !exists {
		return ErrModelNotFound
	}

	delete(s.models, id)
	return nil
}

func (s *MemoryStore) Update(ctx context.Context, model *Model) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.models[model.ID]; !exists {
		return ErrModelNotFound
	}

	s.models[model.ID] = model
	return nil
}

type MockProvider struct {
	pullErr     error
	searchRes   []ModelSearchResult
	importErr   error
	verifyRes   *VerificationResult
	verifyErr   error
	estimate    *ModelRequirements
	estimateErr error
}

func (m *MockProvider) Pull(ctx context.Context, source, repo, tag string, progressCh chan<- PullProgress) (*Model, error) {
	if m.pullErr != nil {
		return nil, m.pullErr
	}

	now := time.Now().Unix()
	return &Model{
		ID:        generateModelID(),
		Name:      repo,
		Type:      ModelTypeLLM,
		Format:    FormatGGUF,
		Status:    StatusReady,
		Source:    source,
		CreatedAt: now,
		UpdatedAt: now,
		Size:      4500000000,
		Requirements: &ModelRequirements{
			MemoryMin:         8000000000,
			MemoryRecommended: 16000000000,
			GPUType:           "NVIDIA RTX 4090",
		},
	}, nil
}

func (m *MockProvider) Search(ctx context.Context, query string, source string, modelType ModelType, limit int) ([]ModelSearchResult, error) {
	return m.searchRes, nil
}

func (m *MockProvider) ImportLocal(ctx context.Context, path string, autoDetect bool) (*Model, error) {
	if m.importErr != nil {
		return nil, m.importErr
	}

	now := time.Now().Unix()
	return &Model{
		ID:        generateModelID(),
		Name:      "imported-model",
		Type:      ModelTypeLLM,
		Format:    FormatGGUF,
		Status:    StatusReady,
		Path:      path,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (m *MockProvider) Verify(ctx context.Context, modelID string, checksum string) (*VerificationResult, error) {
	if m.verifyErr != nil {
		return nil, m.verifyErr
	}
	if m.verifyRes != nil {
		return m.verifyRes, nil
	}
	return &VerificationResult{Valid: true, Issues: []string{}}, nil
}

func (m *MockProvider) EstimateResources(ctx context.Context, modelID string) (*ModelRequirements, error) {
	if m.estimateErr != nil {
		return nil, m.estimateErr
	}
	if m.estimate != nil {
		return m.estimate, nil
	}
	return &ModelRequirements{
		MemoryMin:         8000000000,
		MemoryRecommended: 16000000000,
		GPUType:           "NVIDIA RTX 4090",
	}, nil
}

func createTestModel(id, name string) *Model {
	now := time.Now().Unix()
	return &Model{
		ID:        id,
		Name:      name,
		Type:      ModelTypeLLM,
		Format:    FormatGGUF,
		Status:    StatusReady,
		CreatedAt: now,
		UpdatedAt: now,
		Size:      4500000000,
		Requirements: &ModelRequirements{
			MemoryMin:         8000000000,
			MemoryRecommended: 16000000000,
			GPUType:           "NVIDIA RTX 4090",
		},
	}
}
