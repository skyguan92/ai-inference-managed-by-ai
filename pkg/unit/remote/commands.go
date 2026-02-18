package remote

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type EnableCommand struct {
	store    RemoteStore
	provider RemoteProvider
	events   unit.EventPublisher
}

func NewEnableCommand(store RemoteStore, provider RemoteProvider) *EnableCommand {
	return &EnableCommand{store: store, provider: provider}
}

func NewEnableCommandWithEvents(store RemoteStore, provider RemoteProvider, events unit.EventPublisher) *EnableCommand {
	return &EnableCommand{store: store, provider: provider, events: events}
}

func (c *EnableCommand) Name() string {
	return "remote.enable"
}

func (c *EnableCommand) Domain() string {
	return "remote"
}

func (c *EnableCommand) Description() string {
	return "Enable remote access tunnel"
}

func (c *EnableCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"provider": {
				Name: "provider",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Tunnel provider (frp, cloudflare, tailscale)",
					Enum:        []any{string(TunnelProviderFRP), string(TunnelProviderCloudflare), string(TunnelProviderTailscale)},
				},
			},
			"config": {
				Name: "config",
				Schema: unit.Schema{
					Type:        "object",
					Description: "Optional tunnel configuration",
					Properties: map[string]unit.Field{
						"server":     {Name: "server", Schema: unit.Schema{Type: "string"}},
						"token":      {Name: "token", Schema: unit.Schema{Type: "string"}},
						"expose_api": {Name: "expose_api", Schema: unit.Schema{Type: "boolean"}},
						"expose_mcp": {Name: "expose_mcp", Schema: unit.Schema{Type: "boolean"}},
					},
				},
			},
		},
		Required: []string{"provider"},
	}
}

func (c *EnableCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"tunnel_id":  {Name: "tunnel_id", Schema: unit.Schema{Type: "string"}},
			"public_url": {Name: "public_url", Schema: unit.Schema{Type: "string"}},
		},
	}
}

func (c *EnableCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"provider": "cloudflare"},
			Output:      map[string]any{"tunnel_id": "tunnel-abc123", "public_url": "https://test.tunnel.example.com"},
			Description: "Enable remote access via Cloudflare tunnel",
		},
		{
			Input: map[string]any{
				"provider": "frp",
				"config": map[string]any{
					"server":     "frp.example.com:7000",
					"expose_api": true,
					"expose_mcp": true,
				},
			},
			Output:      map[string]any{"tunnel_id": "tunnel-def456", "public_url": "https://app.frp.example.com"},
			Description: "Enable remote access via FRP with custom configuration",
		},
	}
}

func (c *EnableCommand) Execute(ctx context.Context, input any) (any, error) {
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

	providerStr, _ := inputMap["provider"].(string)
	if providerStr == "" {
		err := fmt.Errorf("provider is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}
	provider := TunnelProvider(providerStr)

	existingTunnel, err := c.store.GetTunnel(ctx)
	if err == nil && existingTunnel != nil && existingTunnel.Status == TunnelStatusConnected {
		ec.PublishFailed(ErrTunnelAlreadyEnabled)
		return nil, ErrTunnelAlreadyEnabled
	}

	config := TunnelConfig{Provider: provider}
	if configMap, ok := inputMap["config"].(map[string]any); ok {
		if server, ok := configMap["server"].(string); ok {
			config.Server = server
		}
		if token, ok := configMap["token"].(string); ok {
			config.Token = token
		}
		if exposeAPI, ok := configMap["expose_api"].(bool); ok {
			config.ExposeAPI = exposeAPI
		}
		if exposeMCP, ok := configMap["expose_mcp"].(bool); ok {
			config.ExposeMCP = exposeMCP
		}
	}

	tunnel, err := c.provider.Enable(ctx, config)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("enable tunnel: %w", err)
	}

	if err := c.store.SetTunnel(ctx, tunnel); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("save tunnel: %w", err)
	}

	output := map[string]any{
		"tunnel_id":  tunnel.ID,
		"public_url": tunnel.PublicURL,
	}
	ec.PublishCompleted(output)
	return output, nil
}

type DisableCommand struct {
	store    RemoteStore
	provider RemoteProvider
	events   unit.EventPublisher
}

func NewDisableCommand(store RemoteStore, provider RemoteProvider) *DisableCommand {
	return &DisableCommand{store: store, provider: provider}
}

func NewDisableCommandWithEvents(store RemoteStore, provider RemoteProvider, events unit.EventPublisher) *DisableCommand {
	return &DisableCommand{store: store, provider: provider, events: events}
}

func (c *DisableCommand) Name() string {
	return "remote.disable"
}

func (c *DisableCommand) Domain() string {
	return "remote"
}

func (c *DisableCommand) Description() string {
	return "Disable remote access tunnel"
}

func (c *DisableCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type:       "object",
		Properties: map[string]unit.Field{},
	}
}

func (c *DisableCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"success": {Name: "success", Schema: unit.Schema{Type: "boolean"}},
		},
	}
}

func (c *DisableCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{},
			Output:      map[string]any{"success": true},
			Description: "Disable remote access",
		},
	}
}

func (c *DisableCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	if c.store == nil || c.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	_, err := c.store.GetTunnel(ctx)
	if err != nil {
		ec.PublishFailed(ErrTunnelNotConnected)
		return nil, ErrTunnelNotConnected
	}

	if err := c.provider.Disable(ctx); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("disable tunnel: %w", err)
	}

	if err := c.store.DeleteTunnel(ctx); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("delete tunnel: %w", err)
	}

	output := map[string]any{"success": true}
	ec.PublishCompleted(output)
	return output, nil
}

type ExecCommand struct {
	store    RemoteStore
	provider RemoteProvider
	events   unit.EventPublisher
}

func NewExecCommand(store RemoteStore, provider RemoteProvider) *ExecCommand {
	return &ExecCommand{store: store, provider: provider}
}

func NewExecCommandWithEvents(store RemoteStore, provider RemoteProvider, events unit.EventPublisher) *ExecCommand {
	return &ExecCommand{store: store, provider: provider, events: events}
}

func (c *ExecCommand) Name() string {
	return "remote.exec"
}

func (c *ExecCommand) Domain() string {
	return "remote"
}

func (c *ExecCommand) Description() string {
	return "Execute a remote command through the tunnel"
}

func (c *ExecCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"command": {
				Name: "command",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Command to execute",
					MinLength:   ptrInt(1),
				},
			},
			"timeout": {
				Name: "timeout",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Timeout in seconds",
					Min:         ptrFloat(1),
					Max:         ptrFloat(3600),
					Default:     30,
				},
			},
		},
		Required: []string{"command"},
	}
}

func (c *ExecCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"stdout":    {Name: "stdout", Schema: unit.Schema{Type: "string"}},
			"stderr":    {Name: "stderr", Schema: unit.Schema{Type: "string"}},
			"exit_code": {Name: "exit_code", Schema: unit.Schema{Type: "number"}},
		},
	}
}

func (c *ExecCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"command": "ls -la"},
			Output:      map[string]any{"stdout": "total 0\n", "stderr": "", "exit_code": 0},
			Description: "Execute a simple command",
		},
		{
			Input:       map[string]any{"command": "sleep 5", "timeout": 10},
			Output:      map[string]any{"stdout": "", "stderr": "", "exit_code": 0},
			Description: "Execute command with custom timeout",
		},
	}
}

func (c *ExecCommand) Execute(ctx context.Context, input any) (any, error) {
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

	command, _ := inputMap["command"].(string)
	if command == "" {
		err := fmt.Errorf("command is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	_, err := c.store.GetTunnel(ctx)
	if err != nil {
		ec.PublishFailed(ErrTunnelNotConnected)
		return nil, ErrTunnelNotConnected
	}

	timeout := 30
	if t, ok := toInt(inputMap["timeout"]); ok && t > 0 {
		timeout = t
	}

	startTime := time.Now()
	result, err := c.provider.Exec(ctx, command, timeout)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("execute command: %w", err)
	}

	record := &AuditRecord{
		ID:        "audit-" + uuid.New().String()[:8],
		Command:   command,
		ExitCode:  result.ExitCode,
		Timestamp: startTime,
		Duration:  int(time.Since(startTime).Milliseconds()),
	}

	if err := c.store.AddAuditRecord(ctx, record); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("save audit record: %w", err)
	}

	output := map[string]any{
		"stdout":    result.Stdout,
		"stderr":    result.Stderr,
		"exit_code": result.ExitCode,
	}
	ec.PublishCompleted(output)
	return output, nil
}
