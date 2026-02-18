package app

import (
	"context"
	"fmt"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type InstallCommand struct {
	store    AppStore
	provider AppProvider
	events   unit.EventPublisher
}

func NewInstallCommand(store AppStore, provider AppProvider) *InstallCommand {
	return &InstallCommand{store: store, provider: provider}
}

func NewInstallCommandWithEvents(store AppStore, provider AppProvider, events unit.EventPublisher) *InstallCommand {
	return &InstallCommand{store: store, provider: provider, events: events}
}

func (c *InstallCommand) Name() string {
	return "app.install"
}

func (c *InstallCommand) Domain() string {
	return "app"
}

func (c *InstallCommand) Description() string {
	return "Install an application from a template"
}

func (c *InstallCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"template": {
				Name: "template",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Template ID to install from",
					MinLength:   ptrInt(1),
				},
			},
			"name": {
				Name: "name",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Application name (optional, defaults to template name)",
				},
			},
			"config": {
				Name: "config",
				Schema: unit.Schema{
					Type:        "object",
					Description: "Application configuration",
				},
			},
		},
		Required: []string{"template"},
	}
}

func (c *InstallCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"app_id": {
				Name:   "app_id",
				Schema: unit.Schema{Type: "string"},
			},
		},
	}
}

func (c *InstallCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"template": "open-webui"},
			Output:      map[string]any{"app_id": "app-abc123"},
			Description: "Install Open WebUI application",
		},
		{
			Input:       map[string]any{"template": "grafana", "name": "my-monitoring", "config": map[string]any{"port": 3000}},
			Output:      map[string]any{"app_id": "app-def456"},
			Description: "Install Grafana with custom name and config",
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

	template, _ := inputMap["template"].(string)
	if template == "" {
		err := fmt.Errorf("template is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	name, _ := inputMap["name"].(string)
	if name == "" {
		name = template
	}

	var config map[string]any
	if cfg, ok := inputMap["config"].(map[string]any); ok {
		config = cfg
	}

	result, err := c.provider.Install(ctx, template, name, config)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("install app from template %s: %w", template, err)
	}

	now := time.Now().Unix()
	app := &App{
		ID:        result.AppID,
		Name:      name,
		Template:  template,
		Status:    AppStatusInstalled,
		Config:    config,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := c.store.Create(ctx, app); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("save app %s: %w", name, err)
	}

	output := map[string]any{"app_id": result.AppID}
	ec.PublishCompleted(output)
	return output, nil
}

type UninstallCommand struct {
	store    AppStore
	provider AppProvider
	events   unit.EventPublisher
}

func NewUninstallCommand(store AppStore, provider AppProvider) *UninstallCommand {
	return &UninstallCommand{store: store, provider: provider}
}

func NewUninstallCommandWithEvents(store AppStore, provider AppProvider, events unit.EventPublisher) *UninstallCommand {
	return &UninstallCommand{store: store, provider: provider, events: events}
}

func (c *UninstallCommand) Name() string {
	return "app.uninstall"
}

func (c *UninstallCommand) Domain() string {
	return "app"
}

func (c *UninstallCommand) Description() string {
	return "Uninstall an application"
}

func (c *UninstallCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"app_id": {
				Name: "app_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Application ID to uninstall",
					MinLength:   ptrInt(1),
				},
			},
			"remove_data": {
				Name: "remove_data",
				Schema: unit.Schema{
					Type:        "boolean",
					Description: "Remove application data",
				},
			},
		},
		Required: []string{"app_id"},
	}
}

func (c *UninstallCommand) OutputSchema() unit.Schema {
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

func (c *UninstallCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"app_id": "app-abc123"},
			Output:      map[string]any{"success": true},
			Description: "Uninstall application",
		},
		{
			Input:       map[string]any{"app_id": "app-abc123", "remove_data": true},
			Output:      map[string]any{"success": true},
			Description: "Uninstall application and remove data",
		},
	}
}

func (c *UninstallCommand) Execute(ctx context.Context, input any) (any, error) {
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

	appID, _ := inputMap["app_id"].(string)
	if appID == "" {
		err := fmt.Errorf("app_id is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	app, err := c.store.Get(ctx, appID)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get app %s: %w", appID, err)
	}

	removeData := false
	if rd, ok := inputMap["remove_data"].(bool); ok {
		removeData = rd
	}

	result, err := c.provider.Uninstall(ctx, appID, removeData)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("uninstall app %s: %w", appID, err)
	}

	if err := c.store.Delete(ctx, appID); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("delete app %s from store: %w", appID, err)
	}

	_ = app

	output := map[string]any{"success": result.Success}
	ec.PublishCompleted(output)
	return output, nil
}

type StartCommand struct {
	store    AppStore
	provider AppProvider
	events   unit.EventPublisher
}

func NewStartCommand(store AppStore, provider AppProvider) *StartCommand {
	return &StartCommand{store: store, provider: provider}
}

func NewStartCommandWithEvents(store AppStore, provider AppProvider, events unit.EventPublisher) *StartCommand {
	return &StartCommand{store: store, provider: provider, events: events}
}

func (c *StartCommand) Name() string {
	return "app.start"
}

func (c *StartCommand) Domain() string {
	return "app"
}

func (c *StartCommand) Description() string {
	return "Start an application"
}

func (c *StartCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"app_id": {
				Name: "app_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Application ID to start",
					MinLength:   ptrInt(1),
				},
			},
		},
		Required: []string{"app_id"},
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
			Input:       map[string]any{"app_id": "app-abc123"},
			Output:      map[string]any{"success": true},
			Description: "Start application",
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

	appID, _ := inputMap["app_id"].(string)
	if appID == "" {
		err := fmt.Errorf("app_id is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	app, err := c.store.Get(ctx, appID)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get app %s: %w", appID, err)
	}

	if app.Status == AppStatusRunning {
		ec.PublishFailed(ErrAppAlreadyRunning)
		return nil, ErrAppAlreadyRunning
	}

	result, err := c.provider.Start(ctx, appID)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("start app %s: %w", appID, err)
	}

	app.Status = AppStatusRunning
	app.UpdatedAt = time.Now().Unix()

	if err := c.store.Update(ctx, app); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("update app %s: %w", appID, err)
	}

	output := map[string]any{"success": result.Success}
	ec.PublishCompleted(output)
	return output, nil
}

type StopCommand struct {
	store    AppStore
	provider AppProvider
	events   unit.EventPublisher
}

func NewStopCommand(store AppStore, provider AppProvider) *StopCommand {
	return &StopCommand{store: store, provider: provider}
}

func NewStopCommandWithEvents(store AppStore, provider AppProvider, events unit.EventPublisher) *StopCommand {
	return &StopCommand{store: store, provider: provider, events: events}
}

func (c *StopCommand) Name() string {
	return "app.stop"
}

func (c *StopCommand) Domain() string {
	return "app"
}

func (c *StopCommand) Description() string {
	return "Stop an application"
}

func (c *StopCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"app_id": {
				Name: "app_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Application ID to stop",
					MinLength:   ptrInt(1),
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
		Required: []string{"app_id"},
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
			Input:       map[string]any{"app_id": "app-abc123"},
			Output:      map[string]any{"success": true},
			Description: "Stop application",
		},
		{
			Input:       map[string]any{"app_id": "app-abc123", "timeout": 60},
			Output:      map[string]any{"success": true},
			Description: "Stop application with timeout",
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

	appID, _ := inputMap["app_id"].(string)
	if appID == "" {
		err := fmt.Errorf("app_id is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	app, err := c.store.Get(ctx, appID)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get app %s: %w", appID, err)
	}

	if app.Status != AppStatusRunning {
		ec.PublishFailed(ErrAppNotRunning)
		return nil, ErrAppNotRunning
	}

	timeout := 30
	if t, ok := toInt(inputMap["timeout"]); ok && t > 0 {
		timeout = t
	}

	result, err := c.provider.Stop(ctx, appID, timeout)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("stop app %s: %w", appID, err)
	}

	app.Status = AppStatusStopped
	app.UpdatedAt = time.Now().Unix()

	if err := c.store.Update(ctx, app); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("update app %s: %w", appID, err)
	}

	output := map[string]any{"success": result.Success}
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

func ptrInt(v int) *int {
	return &v
}
