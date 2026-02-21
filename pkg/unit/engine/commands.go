package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/ptrs"
)

type StartCommand struct {
	store    EngineStore
	provider EngineProvider
	events   unit.EventPublisher
}

func NewStartCommand(store EngineStore, provider EngineProvider) *StartCommand {
	return &StartCommand{store: store, provider: provider}
}

func NewStartCommandWithEvents(store EngineStore, provider EngineProvider, events unit.EventPublisher) *StartCommand {
	return &StartCommand{store: store, provider: provider, events: events}
}

func (c *StartCommand) Name() string {
	return "engine.start"
}

func (c *StartCommand) Domain() string {
	return "engine"
}

func (c *StartCommand) Description() string {
	return "Start an inference engine"
}

func (c *StartCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"name": {
				Name: "name",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Engine name (e.g. vllm, ollama, tts, asr)",
					MinLength:   ptrs.Int(1),
				},
			},
			"model_id": {
				Name: "model_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Model ID to serve (looked up from model store for path and metadata)",
				},
			},
			"model_path": {
				Name: "model_path",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Direct path to model files (alternative to model_id)",
				},
			},
			"config": {
				Name: "config",
				Schema: unit.Schema{
					Type:        "object",
					Description: "Additional engine configuration (gpu, device, gpu_memory_utilization, etc.)",
				},
			},
		},
		Required: []string{"name"},
	}
}

func (c *StartCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"process_id": {
				Name:   "process_id",
				Schema: unit.Schema{Type: "string"},
			},
			"status": {
				Name:   "status",
				Schema: unit.Schema{Type: "string"},
			},
		},
	}
}

func (c *StartCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"name": "vllm", "model_id": "model-abc123"},
			Output:      map[string]any{"process_id": "proc-abc123", "status": "running"},
			Description: "Start vLLM engine for a specific model",
		},
		{
			Input:       map[string]any{"name": "vllm", "model_path": "/mnt/data/models/Qwen3-8B", "config": map[string]any{"gpu_memory_utilization": 0.9}},
			Output:      map[string]any{"process_id": "proc-def456", "status": "running"},
			Description: "Start vLLM engine with explicit model path and config",
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

	name, _ := inputMap["name"].(string)
	if name == "" {
		err := fmt.Errorf("name is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	engine, err := c.store.Get(ctx, name)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get engine %s: %w", name, err)
	}

	if engine.Status == EngineStatusRunning {
		err := ErrEngineAlreadyRunning
		ec.PublishFailed(err)
		return nil, err
	}

	var config map[string]any
	if cfg, ok := inputMap["config"].(map[string]any); ok {
		config = cfg
	}
	if config == nil {
		config = map[string]any{}
	}

	// Merge top-level model_id/model_path into config so the engine provider
	// can look up the model path and mount it into Docker containers.
	if mid, ok := inputMap["model_id"].(string); ok && mid != "" {
		if _, exists := config["model_id"]; !exists {
			config["model_id"] = mid
		}
	}
	if mp, ok := inputMap["model_path"].(string); ok && mp != "" {
		if _, exists := config["model_path"]; !exists {
			config["model_path"] = mp
		}
	}

	result, err := c.provider.Start(ctx, name, config)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("start engine %s: %w", name, err)
	}

	engine.Status = result.Status
	engine.ProcessID = result.ProcessID
	engine.UpdatedAt = time.Now().Unix()
	if config != nil {
		engine.Config = config
	}

	if err := c.store.Update(ctx, engine); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("update engine %s: %w", name, err)
	}

	output := map[string]any{
		"process_id": result.ProcessID,
		"status":     string(result.Status),
	}
	ec.PublishCompleted(output)
	return output, nil
}

type StopCommand struct {
	store    EngineStore
	provider EngineProvider
	events   unit.EventPublisher
}

func NewStopCommand(store EngineStore, provider EngineProvider) *StopCommand {
	return &StopCommand{store: store, provider: provider}
}

func NewStopCommandWithEvents(store EngineStore, provider EngineProvider, events unit.EventPublisher) *StopCommand {
	return &StopCommand{store: store, provider: provider, events: events}
}

func (c *StopCommand) Name() string {
	return "engine.stop"
}

func (c *StopCommand) Domain() string {
	return "engine"
}

func (c *StopCommand) Description() string {
	return "Stop an inference engine"
}

func (c *StopCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"name": {
				Name: "name",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Engine name",
					MinLength:   ptrs.Int(1),
				},
			},
			"force": {
				Name: "force",
				Schema: unit.Schema{
					Type:        "boolean",
					Description: "Force stop the engine",
				},
			},
			"timeout": {
				Name: "timeout",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Timeout in seconds for graceful shutdown",
				},
			},
		},
		Required: []string{"name"},
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
			Input:       map[string]any{"name": "ollama"},
			Output:      map[string]any{"success": true},
			Description: "Stop Ollama engine gracefully",
		},
		{
			Input:       map[string]any{"name": "vllm", "force": true},
			Output:      map[string]any{"success": true},
			Description: "Force stop vLLM engine",
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

	name, _ := inputMap["name"].(string)
	if name == "" {
		err := fmt.Errorf("name is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	engine, err := c.store.Get(ctx, name)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get engine %s: %w", name, err)
	}

	if engine.Status != EngineStatusRunning {
		err := ErrEngineNotRunning
		ec.PublishFailed(err)
		return nil, err
	}

	force := false
	if f, ok := inputMap["force"].(bool); ok {
		force = f
	}

	timeout := 30
	if t, ok := toInt(inputMap["timeout"]); ok && t > 0 {
		timeout = t
	}

	result, err := c.provider.Stop(ctx, name, force, timeout)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("stop engine %s: %w", name, err)
	}

	engine.Status = EngineStatusStopped
	engine.ProcessID = ""
	engine.UpdatedAt = time.Now().Unix()

	if err := c.store.Update(ctx, engine); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("update engine %s: %w", name, err)
	}

	output := map[string]any{"success": result.Success}
	ec.PublishCompleted(output)
	return output, nil
}

type RestartCommand struct {
	store    EngineStore
	provider EngineProvider
	events   unit.EventPublisher
}

func NewRestartCommand(store EngineStore, provider EngineProvider) *RestartCommand {
	return &RestartCommand{store: store, provider: provider}
}

func NewRestartCommandWithEvents(store EngineStore, provider EngineProvider, events unit.EventPublisher) *RestartCommand {
	return &RestartCommand{store: store, provider: provider, events: events}
}

func (c *RestartCommand) Name() string {
	return "engine.restart"
}

func (c *RestartCommand) Domain() string {
	return "engine"
}

func (c *RestartCommand) Description() string {
	return "Restart an inference engine"
}

func (c *RestartCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"name": {
				Name: "name",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Engine name",
					MinLength:   ptrs.Int(1),
				},
			},
		},
		Required: []string{"name"},
	}
}

func (c *RestartCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"process_id": {
				Name:   "process_id",
				Schema: unit.Schema{Type: "string"},
			},
			"status": {
				Name:   "status",
				Schema: unit.Schema{Type: "string"},
			},
		},
	}
}

func (c *RestartCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"name": "ollama"},
			Output:      map[string]any{"process_id": "proc-abc123", "status": "running"},
			Description: "Restart Ollama engine",
		},
	}
}

func (c *RestartCommand) Execute(ctx context.Context, input any) (any, error) {
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

	name, _ := inputMap["name"].(string)
	if name == "" {
		err := fmt.Errorf("name is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	engine, err := c.store.Get(ctx, name)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get engine %s: %w", name, err)
	}

	if engine.Status == EngineStatusRunning {
		_, err := c.provider.Stop(ctx, name, false, 30)
		if err != nil {
			ec.PublishFailed(err)
			return nil, fmt.Errorf("stop engine %s during restart: %w", name, err)
		}
	}

	config := engine.Config
	result, err := c.provider.Start(ctx, name, config)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("start engine %s during restart: %w", name, err)
	}

	engine.Status = result.Status
	engine.ProcessID = result.ProcessID
	engine.UpdatedAt = time.Now().Unix()

	if err := c.store.Update(ctx, engine); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("update engine %s: %w", name, err)
	}

	output := map[string]any{
		"process_id": result.ProcessID,
		"status":     string(result.Status),
	}
	ec.PublishCompleted(output)
	return output, nil
}

type InstallCommand struct {
	store    EngineStore
	provider EngineProvider
	events   unit.EventPublisher
}

func NewInstallCommand(store EngineStore, provider EngineProvider) *InstallCommand {
	return &InstallCommand{store: store, provider: provider}
}

func NewInstallCommandWithEvents(store EngineStore, provider EngineProvider, events unit.EventPublisher) *InstallCommand {
	return &InstallCommand{store: store, provider: provider, events: events}
}

func (c *InstallCommand) Name() string {
	return "engine.install"
}

func (c *InstallCommand) Domain() string {
	return "engine"
}

func (c *InstallCommand) Description() string {
	return "Install an inference engine"
}

func (c *InstallCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"name": {
				Name: "name",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Engine name (ollama, vllm, sglang, etc.)",
					Enum: []any{
						string(EngineTypeOllama),
						string(EngineTypeVLLM),
						string(EngineTypeSGLang),
						string(EngineTypeWhisper),
						string(EngineTypeTTS),
						string(EngineTypeDiffusion),
						string(EngineTypeTransformers),
						string(EngineTypeHuggingFace),
						string(EngineTypeVideo),
						string(EngineTypeRerank),
					},
				},
			},
			"version": {
				Name: "version",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Engine version (optional)",
				},
			},
		},
		Required: []string{"name"},
	}
}

func (c *InstallCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"success": {
				Name:   "success",
				Schema: unit.Schema{Type: "boolean"},
			},
			"path": {
				Name:   "path",
				Schema: unit.Schema{Type: "string"},
			},
		},
	}
}

func (c *InstallCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"name": "ollama"},
			Output:      map[string]any{"success": true, "path": "/usr/local/bin/ollama"},
			Description: "Install Ollama engine",
		},
		{
			Input:       map[string]any{"name": "vllm", "version": "0.4.0"},
			Output:      map[string]any{"success": true, "path": "/usr/local/bin/vllm"},
			Description: "Install specific version of vLLM",
		},
	}
}

func (c *InstallCommand) Execute(ctx context.Context, input any) (any, error) {
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

	name, _ := inputMap["name"].(string)
	if name == "" {
		err := fmt.Errorf("name is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	version, _ := inputMap["version"].(string)

	result, err := c.provider.Install(ctx, name, version)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("install engine %s: %w", name, err)
	}

	now := time.Now().Unix()
	engine := &Engine{
		ID:           "engine-" + uuid.New().String()[:8],
		Name:         name,
		Type:         EngineType(name),
		Status:       EngineStatusStopped,
		Version:      version,
		Path:         result.Path,
		CreatedAt:    now,
		UpdatedAt:    now,
		Models:       []string{},
		Capabilities: []string{},
	}

	if err := c.store.Create(ctx, engine); err != nil {
		if err != ErrEngineAlreadyExists {
			ec.PublishFailed(err)
			return nil, fmt.Errorf("save engine %s: %w", name, err)
		}
	}

	output := map[string]any{
		"success": result.Success,
		"path":    result.Path,
	}
	ec.PublishCompleted(output)
	return output, nil
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
