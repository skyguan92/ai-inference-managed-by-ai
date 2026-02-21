package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/ptrs"
)

type CreateCommand struct {
	store    ServiceStore
	provider ServiceProvider
	events   unit.EventPublisher
}

func NewCreateCommand(store ServiceStore, provider ServiceProvider) *CreateCommand {
	return &CreateCommand{store: store, provider: provider}
}

func NewCreateCommandWithEvents(store ServiceStore, provider ServiceProvider, events unit.EventPublisher) *CreateCommand {
	return &CreateCommand{store: store, provider: provider, events: events}
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
					MinLength:   ptrs.Int(1),
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
					Min:         ptrs.Float64(1),
					Max:         ptrs.Float64(100),
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
		err := fmt.Errorf("model_id is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
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
		ec.PublishFailed(err)
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
		ec.PublishFailed(err)
		return nil, fmt.Errorf("save service: %w", err)
	}

	output := map[string]any{"service_id": service.ID}
	ec.PublishCompleted(output)
	return output, nil
}

type DeleteCommand struct {
	store    ServiceStore
	provider ServiceProvider
	events   unit.EventPublisher
}

func NewDeleteCommand(store ServiceStore, provider ServiceProvider) *DeleteCommand {
	return &DeleteCommand{store: store, provider: provider}
}

func NewDeleteCommandWithEvents(store ServiceStore, provider ServiceProvider, events unit.EventPublisher) *DeleteCommand {
	return &DeleteCommand{store: store, provider: provider, events: events}
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
					MinLength:   ptrs.Int(1),
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
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	if c.store == nil {
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

	serviceID, _ := inputMap["service_id"].(string)
	if serviceID == "" {
		err := fmt.Errorf("service_id is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	if err := c.store.Delete(ctx, serviceID); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("delete service %s: %w", serviceID, err)
	}

	output := map[string]any{"success": true}
	ec.PublishCompleted(output)
	return output, nil
}

type ScaleCommand struct {
	store    ServiceStore
	provider ServiceProvider
	events   unit.EventPublisher
}

func NewScaleCommand(store ServiceStore, provider ServiceProvider) *ScaleCommand {
	return &ScaleCommand{store: store, provider: provider}
}

func NewScaleCommandWithEvents(store ServiceStore, provider ServiceProvider, events unit.EventPublisher) *ScaleCommand {
	return &ScaleCommand{store: store, provider: provider, events: events}
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
					MinLength:   ptrs.Int(1),
				},
			},
			"replicas": {
				Name: "replicas",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Target number of replicas",
					Min:         ptrs.Float64(0),
					Max:         ptrs.Float64(100),
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

	serviceID, _ := inputMap["service_id"].(string)
	if serviceID == "" {
		err := fmt.Errorf("service_id is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	replicas, ok := toInt(inputMap["replicas"])
	if !ok || replicas < 0 {
		err := fmt.Errorf("replicas must be a non-negative integer: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	service, err := c.store.Get(ctx, serviceID)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get service %s: %w", serviceID, err)
	}

	if err := c.provider.Scale(ctx, serviceID, replicas); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("scale service %s: %w", serviceID, err)
	}

	service.Replicas = replicas
	service.UpdatedAt = time.Now().Unix()

	if err := c.store.Update(ctx, service); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("update service %s: %w", serviceID, err)
	}

	output := map[string]any{"success": true}
	ec.PublishCompleted(output)
	return output, nil
}

type StartCommand struct {
	store    ServiceStore
	provider ServiceProvider
	events   unit.EventPublisher
}

func NewStartCommand(store ServiceStore, provider ServiceProvider) *StartCommand {
	return &StartCommand{store: store, provider: provider}
}

func NewStartCommandWithEvents(store ServiceStore, provider ServiceProvider, events unit.EventPublisher) *StartCommand {
	return &StartCommand{store: store, provider: provider, events: events}
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
					MinLength:   ptrs.Int(1),
				},
			},
			"timeout": {
				Name: "timeout",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Timeout in seconds for the start operation (including health check). Defaults to the gateway timeout if not set. Use 600 for large models.",
					Min:         ptrs.Float64(1),
					Max:         ptrs.Float64(3600),
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

	serviceID, _ := inputMap["service_id"].(string)
	if serviceID == "" {
		err := fmt.Errorf("service_id is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	// Apply caller-specified timeout so the agent can override the gateway default.
	if timeoutSec, ok := inputMap["timeout"].(float64); ok && timeoutSec > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
		defer cancel()
	}

	service, err := c.store.Get(ctx, serviceID)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get service %s: %w", serviceID, err)
	}

	// Check if service is already running and container actually exists
	if service.Status == ServiceStatusRunning {
		// Verify the container/process is actually running (Bug #2 fix)
		if c.provider.IsRunning(ctx, serviceID) {
			ec.PublishFailed(ErrServiceAlreadyRunning)
			return nil, ErrServiceAlreadyRunning
		}
		// Container not running but status says running - sync status
		slog.Warn("service status out of sync, container not running", "service_id", serviceID)
		service.Status = ServiceStatusStopped
	}

	// Extract async flag from input (for backward compatibility)
	async := false
	if asyncVal, ok := inputMap["async"].(bool); ok {
		async = asyncVal
	}

	// Create a provider wrapper that passes async flag
	var startErr error
	if asyncProvider, ok := c.provider.(interface {
		StartAsync(ctx context.Context, serviceID string, async bool) error
	}); ok {
		startErr = asyncProvider.StartAsync(ctx, serviceID, async)
	} else {
		startErr = c.provider.Start(ctx, serviceID)
	}

	if startErr != nil {
		// Transition to failed status so the service is not stuck at "creating".
		// Use a fresh context because the original ctx may already be cancelled
		// (e.g. gateway timeout), which is exactly what caused the start failure.
		service.Status = ServiceStatusFailed
		service.UpdatedAt = time.Now().Unix()
		updateCtx, updateCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer updateCancel()
		if updateErr := c.store.Update(updateCtx, service); updateErr != nil {
			slog.Warn("failed to update service status to failed", "service_id", serviceID, "error", updateErr)
		}
		ec.PublishFailed(startErr)
		return nil, fmt.Errorf("start service %s: %w", serviceID, startErr)
	}

	service.Status = ServiceStatusRunning
	service.UpdatedAt = time.Now().Unix()

	if err := c.store.Update(ctx, service); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("update service %s: %w", serviceID, err)
	}

	output := map[string]any{"success": true}
	ec.PublishCompleted(output)
	return output, nil
}

type StopCommand struct {
	store    ServiceStore
	provider ServiceProvider
	events   unit.EventPublisher
}

func NewStopCommand(store ServiceStore, provider ServiceProvider) *StopCommand {
	return &StopCommand{store: store, provider: provider}
}

func NewStopCommandWithEvents(store ServiceStore, provider ServiceProvider, events unit.EventPublisher) *StopCommand {
	return &StopCommand{store: store, provider: provider, events: events}
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
					MinLength:   ptrs.Int(1),
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

	serviceID, _ := inputMap["service_id"].(string)
	if serviceID == "" {
		err := fmt.Errorf("service_id is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	service, err := c.store.Get(ctx, serviceID)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get service %s: %w", serviceID, err)
	}

	force := false
	if f, ok := inputMap["force"].(bool); ok {
		force = f
	}

	if service.Status == ServiceStatusStopped {
		// Already stopped — nothing to do
		output := map[string]any{"success": true}
		ec.PublishCompleted(output)
		return output, nil
	}

	// Attempt to stop even for non-running states (creating, failed) so that
	// any orphaned containers are cleaned up. Ignore stop errors for services
	// that were never fully running — the provider may return an error if there is
	// nothing to stop, which is fine.
	if service.Status == ServiceStatusRunning || service.Status == ServiceStatusCreating || service.Status == ServiceStatusFailed {
		if err := c.provider.Stop(ctx, serviceID, force); err != nil {
			// For non-running services, log but don't fail — the goal is to
			// transition to stopped status regardless.
			if service.Status != ServiceStatusRunning {
				slog.Warn("ignoring stop error for non-running service", "service_id", serviceID, "status", service.Status, "error", err)
			} else {
				ec.PublishFailed(err)
				return nil, fmt.Errorf("stop service %s: %w", serviceID, err)
			}
		}
	}

	service.Status = ServiceStatusStopped
	service.UpdatedAt = time.Now().Unix()

	if err := c.store.Update(ctx, service); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("update service %s: %w", serviceID, err)
	}

	output := map[string]any{"success": true}
	ec.PublishCompleted(output)
	return output, nil
}
