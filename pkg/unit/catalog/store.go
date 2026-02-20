package catalog

import (
	"context"
	"sync"
)

// RecipeStore is the storage interface for recipes.
type RecipeStore interface {
	Create(ctx context.Context, recipe *Recipe) error
	Get(ctx context.Context, id string) (*Recipe, error)
	List(ctx context.Context, filter RecipeFilter) ([]Recipe, int, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, recipe *Recipe) error
}

// MemoryStore is an in-memory implementation of RecipeStore.
type MemoryStore struct {
	recipes map[string]*Recipe
	mu      sync.RWMutex
}

// NewMemoryStore creates a new MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		recipes: make(map[string]*Recipe),
	}
}

func (s *MemoryStore) Create(ctx context.Context, recipe *Recipe) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.recipes[recipe.ID]; exists {
		return ErrRecipeAlreadyExists
	}

	s.recipes[recipe.ID] = recipe
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, id string) (*Recipe, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	recipe, exists := s.recipes[id]
	if !exists {
		return nil, ErrRecipeNotFound
	}
	return recipe, nil
}

func (s *MemoryStore) List(ctx context.Context, filter RecipeFilter) ([]Recipe, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Recipe
	for _, r := range s.recipes {
		if filter.GPUVendor != "" && r.Profile.GPUVendor != filter.GPUVendor {
			continue
		}
		if filter.VerifiedOnly && !r.Verified {
			continue
		}
		if len(filter.Tags) > 0 && !hasAnyTag(r.Tags, filter.Tags) {
			continue
		}
		result = append(result, *r)
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

	if _, exists := s.recipes[id]; !exists {
		return ErrRecipeNotFound
	}

	delete(s.recipes, id)
	return nil
}

func (s *MemoryStore) Update(ctx context.Context, recipe *Recipe) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.recipes[recipe.ID]; !exists {
		return ErrRecipeNotFound
	}

	s.recipes[recipe.ID] = recipe
	return nil
}

// hasAnyTag returns true if the recipe has at least one of the given tags.
func hasAnyTag(recipeTags, filterTags []string) bool {
	tagSet := make(map[string]struct{}, len(recipeTags))
	for _, t := range recipeTags {
		tagSet[t] = struct{}{}
	}
	for _, t := range filterTags {
		if _, ok := tagSet[t]; ok {
			return true
		}
	}
	return false
}

