package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/eventbus"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/remote"
)

func TestRemoteService_NewRemoteService(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	tests := []struct {
		name     string
		registry *unit.Registry
		store    remote.RemoteStore
		provider remote.RemoteProvider
		bus      *eventbus.InMemoryEventBus
	}{
		{
			name:     "with all dependencies",
			registry: unit.NewRegistry(),
			store:    store,
			provider: provider,
			bus:      bus,
		},
		{
			name:     "with nil bus",
			registry: unit.NewRegistry(),
			store:    store,
			provider: provider,
			bus:      nil,
		},
		{
			name:     "with nil provider",
			registry: unit.NewRegistry(),
			store:    store,
			provider: nil,
			bus:      bus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewRemoteService(tt.registry, tt.store, tt.provider, tt.bus)
			if svc == nil {
				t.Error("expected non-nil RemoteService")
			}
		})
	}
}

func TestRemoteService_EnableWithVerify_Success(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&mockCommand{
		name: "remote.enable",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"tunnel_id":  "tunnel-123",
				"public_url": "https://test.tunnel.example.com",
			}, nil
		},
	})
	_ = registry.RegisterQuery(&mockQuery{
		name: "remote.status",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"enabled": true}, nil
		},
	})

	svc := NewRemoteService(registry, store, provider, bus)

	result, err := svc.EnableWithVerify(context.Background(), remote.TunnelProviderFRP, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.TunnelID == "" {
		t.Error("expected non-empty tunnel_id")
	}
	if result.PublicURL == "" {
		t.Error("expected non-empty public_url")
	}
}

func TestRemoteService_EnableWithVerify_CommandNotFound(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	svc := NewRemoteService(registry, store, provider, bus)

	result, err := svc.EnableWithVerify(context.Background(), remote.TunnelProviderFRP, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestRemoteService_EnableWithVerify_EnableFails(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&mockCommand{
		name: "remote.enable",
		execute: func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("enable failed")
		},
	})

	svc := NewRemoteService(registry, store, provider, bus)

	result, err := svc.EnableWithVerify(context.Background(), remote.TunnelProviderFRP, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestRemoteService_DisableWithCleanup_Success(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	_ = store.SetTunnel(context.Background(), &remote.TunnelInfo{
		ID:        "tunnel-123",
		Status:    remote.TunnelStatusConnected,
		Provider:  remote.TunnelProviderFRP,
		PublicURL: "https://test.tunnel.example.com",
		StartedAt: time.Now(),
	})

	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&mockCommand{
		name: "remote.disable",
		execute: func(ctx context.Context, input any) (any, error) {
			_ = store.DeleteTunnel(ctx)
			return map[string]any{"success": true}, nil
		},
	})

	svc := NewRemoteService(registry, store, provider, bus)

	result, err := svc.DisableWithCleanup(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success=true")
	}
	if result.TunnelID == "" {
		t.Error("expected non-empty tunnel_id")
	}
}

func TestRemoteService_DisableWithCleanup_CommandNotFound(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	svc := NewRemoteService(registry, store, provider, bus)

	result, err := svc.DisableWithCleanup(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestRemoteService_ExecWithTimeout_Success(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&mockCommand{
		name: "remote.exec",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"stdout":    "command output",
				"stderr":    "",
				"exit_code": 0,
			}, nil
		},
	})

	svc := NewRemoteService(registry, store, provider, bus)

	result, err := svc.ExecWithTimeout(context.Background(), "ls -la", 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Timeout {
		t.Error("expected timeout=false")
	}
	if result.Stdout == "" {
		t.Error("expected non-empty stdout")
	}
}

func TestRemoteService_ExecWithTimeout_TimeoutClamp(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&mockCommand{
		name: "remote.exec",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			timeout := inputMap["timeout"].(int)
			if timeout > 3600 {
				return nil, errors.New("timeout should be clamped to 3600")
			}
			return map[string]any{"stdout": "ok", "exit_code": 0}, nil
		},
	})

	svc := NewRemoteService(registry, store, provider, bus)

	result, err := svc.ExecWithTimeout(context.Background(), "ls", 5000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestRemoteService_ExecWithTimeout_CommandNotFound(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	svc := NewRemoteService(registry, store, provider, bus)

	result, err := svc.ExecWithTimeout(context.Background(), "ls", 30)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestRemoteService_ExecWithTimeout_DefaultTimeout(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&mockCommand{
		name: "remote.exec",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			timeout := inputMap["timeout"].(int)
			if timeout != 30 {
				return nil, errors.New("expected default timeout of 30")
			}
			return map[string]any{"stdout": "ok", "exit_code": 0}, nil
		},
	})

	svc := NewRemoteService(registry, store, provider, bus)

	result, err := svc.ExecWithTimeout(context.Background(), "ls", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestRemoteService_GetStatus_Success(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	_ = store.SetTunnel(context.Background(), &remote.TunnelInfo{
		ID:        "tunnel-123",
		Status:    remote.TunnelStatusConnected,
		Provider:  remote.TunnelProviderFRP,
		PublicURL: "https://test.tunnel.example.com",
		StartedAt: time.Now(),
	})

	registry := unit.NewRegistry()
	_ = registry.RegisterQuery(&mockQuery{
		name: "remote.status",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"enabled":        true,
				"provider":       "frp",
				"public_url":     "https://test.tunnel.example.com",
				"uptime_seconds": int64(3600),
			}, nil
		},
	})

	svc := NewRemoteService(registry, store, provider, bus)

	result, err := svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Enabled {
		t.Error("expected enabled=true")
	}
	if result.Provider != remote.TunnelProviderFRP {
		t.Errorf("expected provider=frp, got %s", result.Provider)
	}
}

func TestRemoteService_GetStatus_QueryNotFound(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	svc := NewRemoteService(registry, store, provider, bus)

	result, err := svc.GetStatus(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestRemoteService_GetAuditLog_Success(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	_ = registry.RegisterQuery(&mockQuery{
		name: "remote.audit",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"records": []any{
					map[string]any{
						"id":          "audit-1",
						"command":     "ls -la",
						"exit_code":   0,
						"timestamp":   "2024-01-01T00:00:00Z",
						"duration_ms": 100,
					},
				},
			}, nil
		},
	})

	svc := NewRemoteService(registry, store, provider, bus)

	result, err := svc.GetAuditLog(context.Background(), time.Time{}, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Total)
	}
	if len(result.Records) != 1 {
		t.Errorf("expected 1 record, got %d", len(result.Records))
	}
}

func TestRemoteService_GetAuditLog_QueryNotFound(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	svc := NewRemoteService(registry, store, provider, bus)

	result, err := svc.GetAuditLog(context.Background(), time.Time{}, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestRemoteService_IsEnabled_True(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	_ = store.SetTunnel(context.Background(), &remote.TunnelInfo{
		ID:        "tunnel-123",
		Status:    remote.TunnelStatusConnected,
		Provider:  remote.TunnelProviderFRP,
		PublicURL: "https://test.tunnel.example.com",
		StartedAt: time.Now(),
	})

	registry := unit.NewRegistry()
	svc := NewRemoteService(registry, store, provider, bus)

	enabled, err := svc.IsEnabled(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !enabled {
		t.Error("expected enabled=true")
	}
}

func TestRemoteService_IsEnabled_False(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	svc := NewRemoteService(registry, store, provider, bus)

	enabled, err := svc.IsEnabled(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if enabled {
		t.Error("expected enabled=false when no tunnel")
	}
}

func TestRemoteService_GetPublicURL_Success(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	_ = store.SetTunnel(context.Background(), &remote.TunnelInfo{
		ID:        "tunnel-123",
		Status:    remote.TunnelStatusConnected,
		Provider:  remote.TunnelProviderFRP,
		PublicURL: "https://test.tunnel.example.com",
		StartedAt: time.Now(),
	})

	registry := unit.NewRegistry()
	svc := NewRemoteService(registry, store, provider, bus)

	url, err := svc.GetPublicURL(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if url != "https://test.tunnel.example.com" {
		t.Errorf("expected url=https://test.tunnel.example.com, got %s", url)
	}
}

func TestRemoteService_GetPublicURL_NotConnected(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	_ = store.SetTunnel(context.Background(), &remote.TunnelInfo{
		ID:        "tunnel-123",
		Status:    remote.TunnelStatusDisconnected,
		Provider:  remote.TunnelProviderFRP,
		PublicURL: "https://test.tunnel.example.com",
		StartedAt: time.Now(),
	})

	registry := unit.NewRegistry()
	svc := NewRemoteService(registry, store, provider, bus)

	url, err := svc.GetPublicURL(context.Background())
	if err == nil {
		t.Fatal("expected error for disconnected tunnel")
	}
	if url != "" {
		t.Errorf("expected empty url, got %s", url)
	}
}

func TestRemoteService_GetPublicURL_NoTunnel(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	svc := NewRemoteService(registry, store, provider, bus)

	url, err := svc.GetPublicURL(context.Background())
	if err == nil {
		t.Fatal("expected error when no tunnel")
	}
	if url != "" {
		t.Errorf("expected empty url, got %s", url)
	}
}

func TestRemoteService_EnableWithVerify_VerificationFails(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&mockCommand{
		name: "remote.enable",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"tunnel_id":  "tunnel-123",
				"public_url": "https://test.tunnel.example.com",
			}, nil
		},
	})
	_ = registry.RegisterQuery(&mockQuery{
		name: "remote.status",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"enabled": false}, nil
		},
	})

	svc := NewRemoteService(registry, store, provider, bus)

	result, err := svc.EnableWithVerify(context.Background(), remote.TunnelProviderFRP, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Verified {
		t.Error("expected verified=false when verification fails")
	}
}

func TestRemoteService_ExecWithTimeout_ExecFails(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&mockCommand{
		name: "remote.exec",
		execute: func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("exec failed")
		},
	})

	svc := NewRemoteService(registry, store, provider, bus)

	result, err := svc.ExecWithTimeout(context.Background(), "invalid-command", 30)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestRemoteService_DisableWithCleanup_NoTunnel(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&mockCommand{
		name: "remote.disable",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"success": true}, nil
		},
	})

	svc := NewRemoteService(registry, store, provider, bus)

	result, err := svc.DisableWithCleanup(context.Background())
	if err == nil {
		t.Fatal("expected error when no tunnel exists")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestRemoteService_GetStatus_QueryFails(t *testing.T) {
	store := remote.NewMemoryStore()
	provider := &remote.MockProvider{}
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	_ = registry.RegisterQuery(&mockQuery{
		name: "remote.status",
		execute: func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("query failed")
		},
	})

	svc := NewRemoteService(registry, store, provider, bus)

	result, err := svc.GetStatus(context.Background())
	if err == nil {
		t.Fatal("expected error for failed query")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}
