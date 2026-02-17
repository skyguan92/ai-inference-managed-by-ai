package service

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/eventbus"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
)

type ModelService struct {
	registry *unit.Registry
	store    model.ModelStore
	provider model.ModelProvider
	bus      *eventbus.InMemoryEventBus
}

func NewModelService(registry *unit.Registry, store model.ModelStore, provider model.ModelProvider, bus *eventbus.InMemoryEventBus) *ModelService {
	return &ModelService{
		registry: registry,
		store:    store,
		provider: provider,
		bus:      bus,
	}
}

type PullAndVerifyResult struct {
	Model        *model.Model
	Valid        bool
	Issues       []string
	Requirements *model.ModelRequirements
}

func (s *ModelService) PullAndVerify(ctx context.Context, source, repo, tag string) (*PullAndVerifyResult, error) {
	pullCmd := s.registry.GetCommand("model.pull")
	if pullCmd == nil {
		return nil, fmt.Errorf("model.pull command not found")
	}

	input := map[string]any{
		"source": source,
		"repo":   repo,
	}
	if tag != "" {
		input["tag"] = tag
	}

	result, err := pullCmd.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("pull model: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected pull result type")
	}

	modelID, _ := resultMap["model_id"].(string)
	if modelID == "" {
		return nil, fmt.Errorf("model_id not found in pull result")
	}

	verifyCmd := s.registry.GetCommand("model.verify")
	if verifyCmd != nil {
		verifyResult, err := verifyCmd.Execute(ctx, map[string]any{
			"model_id": modelID,
		})
		if err != nil {
			rollbackErr := s.rollbackDelete(ctx, modelID)
			if rollbackErr != nil {
				return nil, fmt.Errorf("verify failed: %w, rollback also failed: %v", err, rollbackErr)
			}
			return nil, fmt.Errorf("verify model: %w", err)
		}

		verifyMap, ok := verifyResult.(map[string]any)
		if ok {
			valid, _ := verifyMap["valid"].(bool)
			if !valid {
				rollbackErr := s.rollbackDelete(ctx, modelID)
				if rollbackErr != nil {
					return nil, fmt.Errorf("model verification failed, rollback also failed: %v", rollbackErr)
				}
				issues, _ := verifyMap["issues"].([]string)
				return nil, fmt.Errorf("model verification failed: %v", issues)
			}
		}
	}

	m, err := s.store.Get(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("get model: %w", err)
	}

	s.publishEvent(ctx, "model.pulled_and_verified", map[string]any{
		"model_id": modelID,
		"source":   source,
		"repo":     repo,
	})

	return &PullAndVerifyResult{
		Model:        m,
		Valid:        true,
		Requirements: m.Requirements,
	}, nil
}

type ImportAndVerifyResult struct {
	Model        *model.Model
	Valid        bool
	Issues       []string
	Requirements *model.ModelRequirements
}

func (s *ModelService) ImportAndVerify(ctx context.Context, path string, opts ...ImportOption) (*ImportAndVerifyResult, error) {
	importCmd := s.registry.GetCommand("model.import")
	if importCmd == nil {
		return nil, fmt.Errorf("model.import command not found")
	}

	input := map[string]any{
		"path":        path,
		"auto_detect": true,
	}

	cfg := &importConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.name != "" {
		input["name"] = cfg.name
	}
	if cfg.modelType != "" {
		input["type"] = cfg.modelType
	}

	result, err := importCmd.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("import model: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected import result type")
	}

	modelID, _ := resultMap["model_id"].(string)
	if modelID == "" {
		return nil, fmt.Errorf("model_id not found in import result")
	}

	verifyCmd := s.registry.GetCommand("model.verify")
	if verifyCmd != nil {
		verifyResult, err := verifyCmd.Execute(ctx, map[string]any{
			"model_id": modelID,
		})
		if err != nil {
			rollbackErr := s.rollbackDelete(ctx, modelID)
			if rollbackErr != nil {
				return nil, fmt.Errorf("verify failed: %w, rollback also failed: %v", err, rollbackErr)
			}
			return nil, fmt.Errorf("verify model: %w", err)
		}

		verifyMap, ok := verifyResult.(map[string]any)
		if ok {
			valid, _ := verifyMap["valid"].(bool)
			if !valid {
				rollbackErr := s.rollbackDelete(ctx, modelID)
				if rollbackErr != nil {
					return nil, fmt.Errorf("model verification failed, rollback also failed: %v", rollbackErr)
				}
				issues, _ := verifyMap["issues"].([]string)
				return nil, fmt.Errorf("model verification failed: %v", issues)
			}
		}
	}

	m, err := s.store.Get(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("get model: %w", err)
	}

	s.publishEvent(ctx, "model.imported_and_verified", map[string]any{
		"model_id": modelID,
		"path":     path,
	})

	return &ImportAndVerifyResult{
		Model:        m,
		Valid:        true,
		Requirements: m.Requirements,
	}, nil
}

type importConfig struct {
	name      string
	modelType string
}

type ImportOption func(*importConfig)

func WithImportName(name string) ImportOption {
	return func(c *importConfig) {
		c.name = name
	}
}

func WithImportType(modelType string) ImportOption {
	return func(c *importConfig) {
		c.modelType = modelType
	}
}

type ModelWithRequirements struct {
	Model        *model.Model
	Requirements *model.ModelRequirements
}

func (s *ModelService) GetWithRequirements(ctx context.Context, modelID string) (*ModelWithRequirements, error) {
	getQuery := s.registry.GetQuery("model.get")
	if getQuery == nil {
		return nil, fmt.Errorf("model.get query not found")
	}

	result, err := getQuery.Execute(ctx, map[string]any{
		"model_id": modelID,
	})
	if err != nil {
		return nil, fmt.Errorf("get model: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected get result type")
	}

	m := &model.Model{
		ID:     getString(resultMap, "id"),
		Name:   getString(resultMap, "name"),
		Type:   model.ModelType(getString(resultMap, "type")),
		Format: model.ModelFormat(getString(resultMap, "format")),
		Status: model.ModelStatus(getString(resultMap, "status")),
		Size:   getInt64(resultMap, "size"),
	}

	var requirements *model.ModelRequirements
	if reqMap, ok := resultMap["requirements"].(map[string]any); ok {
		requirements = &model.ModelRequirements{
			MemoryMin:         getInt64(reqMap, "memory_min"),
			MemoryRecommended: getInt64(reqMap, "memory_recommended"),
			GPUType:           getString(reqMap, "gpu_type"),
			GPUMemory:         getInt64(reqMap, "gpu_memory"),
		}
	}

	if requirements == nil {
		estimateQuery := s.registry.GetQuery("model.estimate_resources")
		if estimateQuery != nil {
			estimateResult, err := estimateQuery.Execute(ctx, map[string]any{
				"model_id": modelID,
			})
			if err == nil {
				if estimateMap, ok := estimateResult.(map[string]any); ok {
					requirements = &model.ModelRequirements{
						MemoryMin:         getInt64(estimateMap, "memory_min"),
						MemoryRecommended: getInt64(estimateMap, "memory_recommended"),
						GPUType:           getString(estimateMap, "gpu_type"),
					}
				}
			}
		}
	}

	m.Requirements = requirements

	return &ModelWithRequirements{
		Model:        m,
		Requirements: requirements,
	}, nil
}

type DeleteWithCleanupResult struct {
	Success      bool
	DeletedFiles []string
	CleanedSpace int64
}

func (s *ModelService) DeleteWithCleanup(ctx context.Context, modelID string, force bool) (*DeleteWithCleanupResult, error) {
	m, err := s.store.Get(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("get model %s: %w", modelID, err)
	}

	var cleanedSpace int64
	var deletedFiles []string

	if m != nil {
		cleanedSpace = m.Size
		if m.Path != "" {
			deletedFiles = append(deletedFiles, m.Path)
		}
	}

	deleteCmd := s.registry.GetCommand("model.delete")
	if deleteCmd == nil {
		return nil, fmt.Errorf("model.delete command not found")
	}

	input := map[string]any{
		"model_id": modelID,
	}
	if force {
		input["force"] = true
	}

	_, err = deleteCmd.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("delete model: %w", err)
	}

	s.publishEvent(ctx, "model.deleted_with_cleanup", map[string]any{
		"model_id":      modelID,
		"cleaned_space": cleanedSpace,
		"deleted_files": deletedFiles,
	})

	return &DeleteWithCleanupResult{
		Success:      true,
		DeletedFiles: deletedFiles,
		CleanedSpace: cleanedSpace,
	}, nil
}

type SearchResultWithEstimate struct {
	ID           string
	Name         string
	Type         model.ModelType
	Source       string
	Description  string
	Downloads    int
	Requirements *model.ModelRequirements
}

func (s *ModelService) SearchAndEstimate(ctx context.Context, query, source string, modelType model.ModelType, limit int) ([]SearchResultWithEstimate, error) {
	searchQuery := s.registry.GetQuery("model.search")
	if searchQuery == nil {
		return nil, fmt.Errorf("model.search query not found")
	}

	input := map[string]any{
		"query": query,
	}
	if source != "" {
		input["source"] = source
	}
	if modelType != "" {
		input["type"] = string(modelType)
	}
	if limit > 0 {
		input["limit"] = limit
	}

	result, err := searchQuery.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("search models: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected search result type")
	}

	results, ok := resultMap["results"].([]map[string]any)
	if !ok {
		if items, ok := resultMap["results"].([]any); ok {
			results = make([]map[string]any, len(items))
			for i, item := range items {
				if m, ok := item.(map[string]any); ok {
					results[i] = m
				}
			}
		}
	}

	searchResults := make([]SearchResultWithEstimate, len(results))
	for i, r := range results {
		searchResults[i] = SearchResultWithEstimate{
			ID:          getString(r, "id"),
			Name:        getString(r, "name"),
			Type:        model.ModelType(getString(r, "type")),
			Source:      getString(r, "source"),
			Description: getString(r, "description"),
			Downloads:   getInt(r, "downloads"),
		}
	}

	return searchResults, nil
}

func (s *ModelService) List(ctx context.Context, filter model.ModelFilter) ([]model.Model, int, error) {
	return s.store.List(ctx, filter)
}

func (s *ModelService) rollbackDelete(ctx context.Context, modelID string) error {
	deleteCmd := s.registry.GetCommand("model.delete")
	if deleteCmd == nil {
		return fmt.Errorf("model.delete command not found")
	}

	_, err := deleteCmd.Execute(ctx, map[string]any{
		"model_id": modelID,
		"force":    true,
	})
	return err
}

func (s *ModelService) publishEvent(ctx context.Context, eventType string, payload any) {
	if s.bus == nil {
		return
	}

	evt := &BaseEvent{
		eventType: eventType,
		domain:    "model",
		payload:   payload,
	}

	_ = s.bus.Publish(evt)
}
