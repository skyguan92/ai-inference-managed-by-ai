package model

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

var (
	ErrModelNotFound      = errors.New("model not found")
	ErrInvalidModelID     = errors.New("invalid model id")
	ErrInvalidInput       = errors.New("invalid input")
	ErrModelAlreadyExists = errors.New("model already exists")
	ErrPullInProgress     = errors.New("pull already in progress")
	ErrProviderNotSet     = errors.New("model provider not set")
)

type ModelStore interface {
	Create(ctx context.Context, model *Model) error
	Get(ctx context.Context, id string) (*Model, error)
	List(ctx context.Context, filter ModelFilter) ([]Model, int, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, model *Model) error
}

type ModelFilter struct {
	Type   ModelType
	Status ModelStatus
	Format ModelFormat
	Limit  int
	Offset int
}

type ModelProvider interface {
	Pull(ctx context.Context, source, repo, tag string, progressCh chan<- PullProgress) (*Model, error)
	Search(ctx context.Context, query string, source string, modelType ModelType, limit int) ([]ModelSearchResult, error)
	ImportLocal(ctx context.Context, path string, autoDetect bool) (*Model, error)
	Verify(ctx context.Context, modelID string, checksum string) (*VerificationResult, error)
	EstimateResources(ctx context.Context, modelID string) (*ModelRequirements, error)
}

// EventPublisher interface for publishing events
type EventPublisher interface {
	Publish(event any) error
}

type CreateCommand struct {
	store  ModelStore
	events EventPublisher
}

func NewCreateCommand(store ModelStore) *CreateCommand {
	return &CreateCommand{store: store}
}

func NewCreateCommandWithEvents(store ModelStore, events EventPublisher) *CreateCommand {
	return &CreateCommand{store: store, events: events}
}

func (c *CreateCommand) Name() string {
	return "model.create"
}

func (c *CreateCommand) Domain() string {
	return "model"
}

func (c *CreateCommand) Description() string {
	return "Create a new model record"
}

func (c *CreateCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"name": {
				Name: "name",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Model name",
					MinLength:   ptrInt(1),
				},
			},
			"type": {
				Name: "type",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Model type",
					Enum:        []any{string(ModelTypeLLM), string(ModelTypeVLM), string(ModelTypeASR), string(ModelTypeTTS), string(ModelTypeEmbedding), string(ModelTypeDiffusion), string(ModelTypeVideoGen), string(ModelTypeDetection), string(ModelTypeRerank)},
				},
			},
			"source": {
				Name: "source",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Model source (ollama, huggingface, modelscope)",
				},
			},
			"format": {
				Name: "format",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Model format",
					Enum:        []any{string(FormatGGUF), string(FormatSafetensors), string(FormatONNX), string(FormatTensorRT), string(FormatPyTorch)},
				},
			},
			"path": {
				Name: "path",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Local path to model files",
				},
			},
		},
		Required: []string{"name"},
	}
}

func (c *CreateCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"model_id": {
				Name:   "model_id",
				Schema: unit.Schema{Type: "string"},
			},
		},
	}
}

func (c *CreateCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"name": "llama3", "type": "llm", "format": "gguf"},
			Output:      map[string]any{"model_id": "model-abc123"},
			Description: "Create a new LLM model record",
		},
	}
}

func (c *CreateCommand) Execute(ctx context.Context, input any) (any, error) {
	if c.store == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	name, _ := inputMap["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("name is required: %w", ErrInvalidInput)
	}

	now := time.Now().Unix()
	model := &Model{
		ID:        generateModelID(),
		Name:      name,
		Type:      ModelTypeLLM,
		Format:    FormatGGUF,
		Status:    StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if t, ok := inputMap["type"].(string); ok && t != "" {
		model.Type = ModelType(t)
	}
	if f, ok := inputMap["format"].(string); ok && f != "" {
		model.Format = ModelFormat(f)
	}
	if s, ok := inputMap["source"].(string); ok {
		model.Source = s
	}
	if p, ok := inputMap["path"].(string); ok {
		model.Path = p
	}

	if err := c.store.Create(ctx, model); err != nil {
		return nil, fmt.Errorf("create model: %w", err)
	}

	// Publish event if event publisher is set
	if c.events != nil {
		if err := c.events.Publish(NewCreatedEvent(model)); err != nil {
			// Log error but don't fail the command
			fmt.Printf("warning: failed to publish model.created event: %v\n", err)
		}
	}

	return map[string]any{"model_id": model.ID}, nil
}

type DeleteCommand struct {
	store  ModelStore
	events EventPublisher
}

func NewDeleteCommand(store ModelStore) *DeleteCommand {
	return &DeleteCommand{store: store}
}

func NewDeleteCommandWithEvents(store ModelStore, events EventPublisher) *DeleteCommand {
	return &DeleteCommand{store: store, events: events}
}

func (c *DeleteCommand) Name() string {
	return "model.delete"
}

func (c *DeleteCommand) Domain() string {
	return "model"
}

func (c *DeleteCommand) Description() string {
	return "Delete a model"
}

func (c *DeleteCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"model_id": {
				Name: "model_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Model identifier",
				},
			},
			"force": {
				Name: "force",
				Schema: unit.Schema{
					Type:        "boolean",
					Description: "Force delete even if model is in use",
				},
			},
		},
		Required: []string{"model_id"},
	}
}

func (c *DeleteCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"success": {
				Name:   "success",
				Schema: unit.Schema{Type: "boolean"},
			},
		},
	}
}

func (c *DeleteCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"model_id": "model-abc123"},
			Output:      map[string]any{"success": true},
			Description: "Delete a model",
		},
		{
			Input:       map[string]any{"model_id": "model-abc123", "force": true},
			Output:      map[string]any{"success": true},
			Description: "Force delete a model",
		},
	}
}

func (c *DeleteCommand) Execute(ctx context.Context, input any) (any, error) {
	if c.store == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	modelID, _ := inputMap["model_id"].(string)
	if modelID == "" {
		return nil, ErrInvalidModelID
	}

	model, err := c.store.Get(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("get model %s: %w", modelID, err)
	}

	if err := c.store.Delete(ctx, modelID); err != nil {
		return nil, fmt.Errorf("delete model %s: %w", modelID, err)
	}

	// Publish event if event publisher is set
	if c.events != nil {
		if err := c.events.Publish(NewDeletedEvent(modelID, model.Name)); err != nil {
			fmt.Printf("warning: failed to publish model.deleted event: %v\n", err)
		}
	}

	return map[string]any{"success": true}, nil
}

type PullCommand struct {
	store    ModelStore
	provider ModelProvider
	progress map[string]bool
	mu       sync.Mutex
	events   unit.EventPublisher
}

func NewPullCommand(store ModelStore, provider ModelProvider) *PullCommand {
	return &PullCommand{
		store:    store,
		provider: provider,
		progress: make(map[string]bool),
	}
}

func NewPullCommandWithEvents(store ModelStore, provider ModelProvider, events unit.EventPublisher) *PullCommand {
	return &PullCommand{
		store:    store,
		provider: provider,
		progress: make(map[string]bool),
		events:   events,
	}
}

func (c *PullCommand) Name() string {
	return "model.pull"
}

func (c *PullCommand) Domain() string {
	return "model"
}

func (c *PullCommand) Description() string {
	return "Pull a model from a remote source"
}

func (c *PullCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"source": {
				Name: "source",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Model source (ollama, huggingface, modelscope)",
					Enum:        []any{"ollama", "huggingface", "modelscope"},
				},
			},
			"repo": {
				Name: "repo",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Repository name (e.g., llama3, meta-llama/Llama-3-8B)",
				},
			},
			"tag": {
				Name: "tag",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Model tag or version (e.g., latest, v1.0)",
				},
			},
			"mirror": {
				Name: "mirror",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Mirror URL for faster download",
				},
			},
		},
		Required: []string{"source", "repo"},
	}
}

func (c *PullCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"model_id": {
				Name:   "model_id",
				Schema: unit.Schema{Type: "string"},
			},
			"status": {
				Name:   "status",
				Schema: unit.Schema{Type: "string"},
			},
		},
	}
}

func (c *PullCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"source": "ollama", "repo": "llama3"},
			Output:      map[string]any{"model_id": "model-abc123", "status": "ready"},
			Description: "Pull llama3 from Ollama",
		},
		{
			Input:       map[string]any{"source": "huggingface", "repo": "meta-llama/Llama-3-8B", "tag": "main"},
			Output:      map[string]any{"model_id": "model-def456", "status": "ready"},
			Description: "Pull Llama-3-8B from HuggingFace",
		},
	}
}

func (c *PullCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	if c.store == nil || c.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	source, _ := inputMap["source"].(string)
	repo, _ := inputMap["repo"].(string)
	tag, _ := inputMap["tag"].(string)

	if source == "" || repo == "" {
		err := fmt.Errorf("source and repo are required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	pullKey := fmt.Sprintf("%s/%s/%s", source, repo, tag)
	c.mu.Lock()
	if c.progress[pullKey] {
		c.mu.Unlock()
		err := ErrPullInProgress
		ec.PublishFailed(err)
		return nil, err
	}
	c.progress[pullKey] = true
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.progress, pullKey)
		c.mu.Unlock()
	}()

	model, err := c.provider.Pull(ctx, source, repo, tag, nil)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("pull model from %s: %w", source, err)
	}

	if err := c.store.Create(ctx, model); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("save model: %w", err)
	}

	output := map[string]any{
		"model_id": model.ID,
		"status":   string(model.Status),
	}
	ec.PublishCompleted(output)
	return output, nil
}

type ImportCommand struct {
	store    ModelStore
	provider ModelProvider
	events   unit.EventPublisher
}

func NewImportCommand(store ModelStore, provider ModelProvider) *ImportCommand {
	return &ImportCommand{store: store, provider: provider}
}

func NewImportCommandWithEvents(store ModelStore, provider ModelProvider, events unit.EventPublisher) *ImportCommand {
	return &ImportCommand{store: store, provider: provider, events: events}
}

func (c *ImportCommand) Name() string {
	return "model.import"
}

func (c *ImportCommand) Domain() string {
	return "model"
}

func (c *ImportCommand) Description() string {
	return "Import a model from local path"
}

func (c *ImportCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"path": {
				Name: "path",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Local path to model files",
				},
			},
			"name": {
				Name: "name",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Model name (auto-detected if not specified)",
				},
			},
			"type": {
				Name: "type",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Model type (auto-detected if not specified)",
				},
			},
			"auto_detect": {
				Name: "auto_detect",
				Schema: unit.Schema{
					Type:        "boolean",
					Description: "Auto-detect model type and format",
				},
			},
		},
		Required: []string{"path"},
	}
}

func (c *ImportCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"model_id": {
				Name:   "model_id",
				Schema: unit.Schema{Type: "string"},
			},
		},
	}
}

func (c *ImportCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"path": "/models/llama3", "auto_detect": true},
			Output:      map[string]any{"model_id": "model-abc123"},
			Description: "Import model with auto-detection",
		},
		{
			Input:       map[string]any{"path": "/models/custom", "name": "my-model", "type": "llm"},
			Output:      map[string]any{"model_id": "model-def456"},
			Description: "Import model with explicit settings",
		},
	}
}

func (c *ImportCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	if c.store == nil || c.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	path, _ := inputMap["path"].(string)
	if path == "" {
		err := fmt.Errorf("path is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	autoDetect := true
	if v, ok := inputMap["auto_detect"].(bool); ok {
		autoDetect = v
	}

	model, err := c.provider.ImportLocal(ctx, path, autoDetect)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("import model from %s: %w", path, err)
	}

	if name, ok := inputMap["name"].(string); ok && name != "" {
		model.Name = name
	}
	if t, ok := inputMap["type"].(string); ok && t != "" {
		model.Type = ModelType(t)
	}

	if err := c.store.Create(ctx, model); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("save imported model: %w", err)
	}

	output := map[string]any{"model_id": model.ID}
	ec.PublishCompleted(output)
	return output, nil
}

type VerifyCommand struct {
	store    ModelStore
	provider ModelProvider
	events   unit.EventPublisher
}

func NewVerifyCommand(store ModelStore, provider ModelProvider) *VerifyCommand {
	return &VerifyCommand{store: store, provider: provider}
}

func NewVerifyCommandWithEvents(store ModelStore, provider ModelProvider, events unit.EventPublisher) *VerifyCommand {
	return &VerifyCommand{store: store, provider: provider, events: events}
}

func (c *VerifyCommand) Name() string {
	return "model.verify"
}

func (c *VerifyCommand) Domain() string {
	return "model"
}

func (c *VerifyCommand) Description() string {
	return "Verify model integrity"
}

func (c *VerifyCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"model_id": {
				Name: "model_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Model identifier",
				},
			},
			"checksum": {
				Name: "checksum",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Expected checksum (optional)",
				},
			},
		},
		Required: []string{"model_id"},
	}
}

func (c *VerifyCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"valid": {
				Name:   "valid",
				Schema: unit.Schema{Type: "boolean"},
			},
			"issues": {
				Name: "issues",
				Schema: unit.Schema{
					Type:  "array",
					Items: &unit.Schema{Type: "string"},
				},
			},
		},
	}
}

func (c *VerifyCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"model_id": "model-abc123"},
			Output:      map[string]any{"valid": true, "issues": []string{}},
			Description: "Verify model integrity",
		},
		{
			Input:       map[string]any{"model_id": "model-abc123", "checksum": "sha256:abc123..."},
			Output:      map[string]any{"valid": false, "issues": []string{"checksum mismatch"}},
			Description: "Verify model with expected checksum",
		},
	}
}

func (c *VerifyCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	if c.store == nil || c.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	modelID, _ := inputMap["model_id"].(string)
	if modelID == "" {
		err := ErrInvalidModelID
		ec.PublishFailed(err)
		return nil, err
	}

	if _, err := c.store.Get(ctx, modelID); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get model %s: %w", modelID, err)
	}

	checksum, _ := inputMap["checksum"].(string)

	result, err := c.provider.Verify(ctx, modelID, checksum)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("verify model %s: %w", modelID, err)
	}

	output := map[string]any{
		"valid":  result.Valid,
		"issues": result.Issues,
	}
	ec.PublishCompleted(output)
	return output, nil
}

func generateModelID() string {
	return "model-" + uuid.New().String()[:8]
}

func ptrInt(v int) *int {
	return &v
}
