package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
)

// FileStore implements a file-based ModelStore using JSON
type FileStore struct {
	dataDir string
	models  map[string]*model.Model
	mu      sync.RWMutex
}

// NewFileStore creates a new file-based model store
func NewFileStore(dataDir string) (*FileStore, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	s := &FileStore{
		dataDir: dataDir,
		models:  make(map[string]*model.Model),
	}

	// Load existing data
	if err := s.load(); err != nil {
		// Ignore errors if file doesn't exist
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return s, nil
}

func (s *FileStore) filePath() string {
	return filepath.Join(s.dataDir, "models.json")
}

func (s *FileStore) load() error {
	data, err := os.ReadFile(s.filePath())
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return json.Unmarshal(data, &s.models)
}

// save persists models to disk. Caller must hold s.mu (read or write lock).
func (s *FileStore) save() error {
	data, err := json.MarshalIndent(s.models, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal models: %w", err)
	}

	return os.WriteFile(s.filePath(), data, 0644)
}

// Create implements ModelStore.Create
func (s *FileStore) Create(ctx context.Context, m *model.Model) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.models[m.ID]; exists {
		return model.ErrModelAlreadyExists
	}

	s.models[m.ID] = m
	return s.save()
}

// Get implements ModelStore.Get
func (s *FileStore) Get(ctx context.Context, id string) (*model.Model, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, exists := s.models[id]
	if !exists {
		return nil, model.ErrModelNotFound
	}

	// Return a copy to prevent callers from mutating the internal map entry.
	copy := *m
	return &copy, nil
}

// List implements ModelStore.List
func (s *FileStore) List(ctx context.Context, filter model.ModelFilter) ([]model.Model, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []model.Model
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

	// Apply pagination
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

// Delete implements ModelStore.Delete
func (s *FileStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.models[id]; !exists {
		return model.ErrModelNotFound
	}

	delete(s.models, id)
	return s.save()
}

// Update implements ModelStore.Update
func (s *FileStore) Update(ctx context.Context, m *model.Model) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.models[m.ID]; !exists {
		return model.ErrModelNotFound
	}

	s.models[m.ID] = m
	return s.save()
}

// Ensure FileStore implements ModelStore interface
var _ model.ModelStore = (*FileStore)(nil)
