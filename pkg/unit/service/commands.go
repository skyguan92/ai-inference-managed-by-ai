package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type CreateCommand struct {
	store    ServiceStore
	provider ServiceProvider
}

func NewCreateCommand(store ServiceStore, provider ServiceProvider) *CreateCommand {
	return &CreateCommand{store: store, provider: provider}
}

func (c *CreateCommand) Name() string {
	return "service.create"
}

func (c *CreateCommand) Domain() string {
	return "service"
}

func (c *CreateCommand) Description() string {
	return "Create a new model inference service"
}

func (c *CreateCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"model_id": {
				Name: "model_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Model ID to deploy",
					MinLength:   ptrInt(1),
				},
			},
			"resource_class": {
				Name: "resource_class",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Resource class (small, medium, large)",
					Enum:        []any{string(ResourceClassSmall), string(ResourceClassMedium), string(ResourceClassLarge)},
					Default:     string(ResourceClassMedium),
				},
			},
			"replicas": {
				Name: "replicas",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Number of replicas",
					Min:         ptrFloat(1),
					Max:         ptrFloat(100),
					Default:     1,
				},
			},
			"persistent": {
				Name: "persistent",
				Schema: unit.Schema{
					Type:        "boolean",
					Description: "Whether service should persist across restarts",
					Default:     false,
				},
			},
		},
		Required: []string{"model_id"},
	}
}

func (c *CreateCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"service_id": {
				Name:   "service_id",
				Schema: unit.Schema{Type: "string"},
			},
		},
	}
}

func (c *CreateCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"model_id": "llama3-70b"},
			Output:      map[string]any{"service_id": "svc-abc123"},
			Description: "Create service for llama3-70b with defaults",
		},
		{
			Input:       map[string]any{"model_id": "mistral-7b", "resource_class": "large", "replicas": 3},
			Output:      map[string]any{"service_id": "svc-def456"},
			Description: "Create service with custom configuration",
		},
	}
}

func (c *CreateCommand) Execute(ctx context.Context, input any) (any, error) {
	if c.store == nil || c.provider == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	modelID, _ := inputMap["model_id"].(string)
	if modelID == "" {
		return nil, fmt.Errorf("model_id is required: %w", ErrInvalidInput)
	}

	resourceClass := ResourceClassMedium
	if rc, ok := inputMap["resource_class"].(string); ok && rc != "" {
		resourceClass = ResourceClass(rc)
	}

	replicas := 1
	if r, ok := toInt(inputMap["replicas"]); ok && r > 0 {
		replicas = r
	}

	persistent := false
	if p, ok := inputMap["persistent"].(bool); ok {
		persistent = p
	}

	result, err := c.provider.Create(ctx, modelID, resourceClass, replicas, persistent)
	if err != nil {
		return nil, fmt.Errorf("create service: %w", err)
	}

	now := time.Now().Unix()
	service := &ModelService{
		ID:            result.ID,
		Name:          "service-" + result.ID,
		ModelID:       modelID,
		Status:        ServiceStatusCreating,
		Replicas:      replicas,
		ResourceClass: resourceClass,
		Endpoints:     []string{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := c.store.Create(ctx, service); err != nil {
		return nil, fmt.Errorf("save service: %w", err)
	}

	return map[string]any{"service_id": service.ID}, nil
}

type DeleteCommand struct {
	store    ServiceStore
	provider ServiceProvider
}

func NewDeleteCommand(store ServiceStore, provider ServiceProvider) *DeleteCommand {
	return &DeleteCommand{store: store, provider: provider}
}

func (c *DeleteCommand) Name() string {
	return "service.delete"
}

func (c *DeleteCommand) Domain() string {
	return "service"
}

func (c *DeleteCommand) Description() string {
	return "Delete a model inference service"
}

func (c *DeleteCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"service_id": {
				Name: "service_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Service ID to delete",
					MinLength:   ptrInt(1),
				},
			},
		},
		Required: []string{"service_id"},
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
			Input:       map[string]any{"service_id": "svc-abc123"},
			Output:      map[string]any{"success": true},
			Description: "Delete a service",
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

	serviceID, _ := inputMap["service_id"].(string)
	if serviceID == "" {
		return nil, fmt.Errorf("service_id is required: %w", ErrInvalidInput)
	}

	if err := c.store.Delete(ctx, serviceID); err != nil {
		return nil, fmt.Errorf("delete service %s: %w", serviceID, err)
	}

	return map[string]any{"success": true}, nil
}

type ScaleCommand struct {
	store    ServiceStore
	provider ServiceProvider
}

func NewScaleCommand(store ServiceStore, provider ServiceProvider) *ScaleCommand {
	return &ScaleCommand{store: store, provider: provider}
}

func (c *ScaleCommand) Name() string {
	return "service.scale"
}

func (c *ScaleCommand) Domain() string {
	return "service"
}

func (c *ScaleCommand) Description() string {
	return "Scale service replicas up or down"
}

func (c *ScaleCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"service_id": {
				Name: "service_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Service ID to scale",
					MinLength:   ptrInt(1),
				},
			},
			"replicas": {
				Name: "replicas",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Target number of replicas",
					Min:         ptrFloat(0),
					Max:         ptrFloat(100),
				},
			},
		},
		Required: []string{"service_id", "replicas"},
	}
}

func (c *ScaleCommand) OutputSchema() unit.Schema {
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

func (c *ScaleCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"service_id": "svc-abc123", "replicas": 5},
			Output:      map[string]any{"success": true},
			Description: "Scale service to 5 replicas",
		},
	}
}

func (c *ScaleCommand) Execute(ctx context.Context, input any) (any, error) {
	if c.store == nil || c.provider == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	serviceID, _ := inputMap["service_id"].(string)
	if serviceID == "" {
		return nil, fmt.Errorf("service_id is required: %w", ErrInvalidInput)
	}

	replicas, ok := toInt(inputMap["replicas"])
	if !ok || replicas < 0 {
		return nil, fmt.Errorf("replicas must be a non-negative integer: %w", ErrInvalidInput)
	}

	service, err := c.store.Get(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("get service %s: %w", serviceID, err)
	}

	if err := c.provider.Scale(ctx, serviceID, replicas); err != nil {
		return nil, fmt.Errorf("scale service %s: %w", serviceID, err)
	}

	service.Replicas = replicas
	service.UpdatedAt = time.Now().Unix()

	if err := c.store.Update(ctx, service); err != nil {
		return nil, fmt.Errorf("update service %s: %w", serviceID, err)
	}

	return map[string]any{"success": true}, nil
}

type StartCommand struct {
	store    ServiceStore
	provider ServiceProvider
}

func NewStartCommand(store ServiceStore, provider ServiceProvider) *StartCommand {
	return &StartCommand{store: store, provider: provider}
}

func (c *StartCommand) Name() string {
	return "service.start"
}

func (c *StartCommand) Domain() string {
	return "service"
}

func (c *StartCommand) Description() string {
	return "Start a stopped service"
}

func (c *StartCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"service_id": {
				Name: "service_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Service ID to start",
					MinLength:   ptrInt(1),
				},
			},
		},
		Required: []string{"service_id"},
	}
}

func (c *StartCommand) OutputSchema() unit.Schema {
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

func (c *StartCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"service_id": "svc-abc123"},
			Output:      map[string]any{"success": true},
			Description: "Start a service",
		},
	}
}

func (c *StartCommand) Execute(ctx context.Context, input any) (any, error) {
	if c.store == nil || c.provider == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	serviceID, _ := inputMap["service_id"].(string)
	if serviceID == "" {
		return nil, fmt.Errorf("service_id is required: %w", ErrInvalidInput)
	}

	service, err := c.store.Get(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("get service %s: %w", serviceID, err)
	}

	if service.Status == ServiceStatusRunning {
		return nil, ErrServiceAlreadyRunning
	}

	if err := c.provider.Start(ctx, serviceID); err != nil {
		return nil, fmt.Errorf("start service %s: %w", serviceID, err)
	}

	service.Status = ServiceStatusRunning
	service.UpdatedAt = time.Now().Unix()

	if err := c.store.Update(ctx, service); err != nil {
		return nil, fmt.Errorf("update service %s: %w", serviceID, err)
	}

	return map[string]any{"success": true}, nil
}

type StopCommand struct {
	store    ServiceStore
	provider ServiceProvider
}

func NewStopCommand(store ServiceStore, provider ServiceProvider) *StopCommand {
	return &StopCommand{store: store, provider: provider}
}

func (c *StopCommand) Name() string {
	return "service.stop"
}

func (c *StopCommand) Domain() string {
	return "service"
}

func (c *StopCommand) Description() string {
	return "Stop a running service"
}

func (c *StopCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"service_id": {
				Name: "service_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Service ID to stop",
					MinLength:   ptrInt(1),
				},
			},
			"force": {
				Name: "force",
				Schema: unit.Schema{
					Type:        "boolean",
					Description: "Force stop without graceful shutdown",
				},
			},
		},
		Required: []string{"service_id"},
	}
}

func (c *StopCommand) OutputSchema() unit.Schema {
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

func (c *StopCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"service_id": "svc-abc123"},
			Output:      map[string]any{"success": true},
			Description: "Stop a service gracefully",
		},
		{
			Input:       map[string]any{"service_id": "svc-abc123", "force": true},
			Output:      map[string]any{"success": true},
			Description: "Force stop a service",
		},
	}
}

func (c *StopCommand) Execute(ctx context.Context, input any) (any, error) {
	if c.store == nil || c.provider == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	serviceID, _ := inputMap["service_id"].(string)
	if serviceID == "" {
		return nil, fmt.Errorf("service_id is required: %w", ErrInvalidInput)
	}

	service, err := c.store.Get(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("get service %s: %w", serviceID, err)
	}

	if service.Status != ServiceStatusRunning {
		return nil, ErrServiceNotRunning
	}

	force := false
	if f, ok := inputMap["force"].(bool); ok {
		force = f
	}

	if err := c.provider.Stop(ctx, serviceID, force); err != nil {
		return nil, fmt.Errorf("stop service %s: %w", serviceID, err)
	}

	service.Status = ServiceStatusStopped
	service.UpdatedAt = time.Now().Unix()

	if err := c.store.Update(ctx, service); err != nil {
		return nil, fmt.Errorf("update service %s: %w", serviceID, err)
	}

	return map[string]any{"success": true}, nil
}
