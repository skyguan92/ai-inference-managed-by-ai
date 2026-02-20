package remote

import (
	"context"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestStatusResource_URI(t *testing.T) {
	r := NewStatusResource(nil)
	expected := "asms://remote/status"
	if r.URI() != expected {
		t.Errorf("expected URI '%s', got '%s'", expected, r.URI())
	}
}

func TestStatusResource_Domain(t *testing.T) {
	r := NewStatusResource(nil)
	if r.Domain() != "remote" {
		t.Errorf("expected domain 'remote', got '%s'", r.Domain())
	}
}

func TestStatusResource_Schema(t *testing.T) {
	r := NewStatusResource(nil)
	schema := r.Schema()

	if schema.Type != "object" {
		t.Errorf("expected schema type 'object', got '%s'", schema.Type)
	}
	if len(schema.Properties) == 0 {
		t.Error("expected schema to have properties")
	}
}

func TestStatusResource_Get(t *testing.T) {
	tests := []struct {
		name    string
		store   RemoteStore
		wantErr bool
	}{
		{
			name:    "get with tunnel",
			store:   createStoreWithTunnel(),
			wantErr: false,
		},
		{
			name:    "get without tunnel",
			store:   NewMemoryStore(),
			wantErr: false,
		},
		{
			name:    "nil store",
			store:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewStatusResource(tt.store)
			result, err := r.Get(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Error("expected result to be map[string]any")
				return
			}

			if _, exists := resultMap["enabled"]; !exists {
				t.Error("expected field 'enabled' not found")
			}
			if _, exists := resultMap["status"]; !exists {
				t.Error("expected field 'status' not found")
			}
		})
	}
}

func TestStatusResource_Watch(t *testing.T) {
	store := createStoreWithTunnel()
	r := NewStatusResource(store)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	select {
	case update := <-ch:
		if update.URI != r.URI() {
			t.Errorf("expected URI '%s', got '%s'", r.URI(), update.URI)
		}
	case <-ctx.Done():
	}
}

func TestAuditResource_URI(t *testing.T) {
	r := NewAuditResource(nil)
	expected := "asms://remote/audit"
	if r.URI() != expected {
		t.Errorf("expected URI '%s', got '%s'", expected, r.URI())
	}
}

func TestAuditResource_Domain(t *testing.T) {
	r := NewAuditResource(nil)
	if r.Domain() != "remote" {
		t.Errorf("expected domain 'remote', got '%s'", r.Domain())
	}
}

func TestAuditResource_Schema(t *testing.T) {
	r := NewAuditResource(nil)
	schema := r.Schema()

	if schema.Type != "object" {
		t.Errorf("expected schema type 'object', got '%s'", schema.Type)
	}
	if len(schema.Properties) == 0 {
		t.Error("expected schema to have properties")
	}
}

func TestAuditResource_Get(t *testing.T) {
	tests := []struct {
		name    string
		store   RemoteStore
		wantErr bool
	}{
		{
			name:    "get with records",
			store:   createStoreWithAuditRecords(),
			wantErr: false,
		},
		{
			name:    "get empty store",
			store:   NewMemoryStore(),
			wantErr: false,
		},
		{
			name:    "nil store",
			store:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewAuditResource(tt.store)
			result, err := r.Get(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Error("expected result to be map[string]any")
				return
			}

			if _, exists := resultMap["records"]; !exists {
				t.Error("expected field 'records' not found")
			}
		})
	}
}

func TestAuditResource_Watch(t *testing.T) {
	store := createStoreWithAuditRecords()
	r := NewAuditResource(store)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	select {
	case update := <-ch:
		if update.URI != r.URI() {
			t.Errorf("expected URI '%s', got '%s'", r.URI(), update.URI)
		}
	case <-ctx.Done():
	}
}

func TestResourceImplementsInterface(t *testing.T) {
	var _ unit.Resource = NewStatusResource(nil)
	var _ unit.Resource = NewAuditResource(nil)
}

func TestStatusResource_GetWithConnectedTunnel(t *testing.T) {
	store := NewMemoryStore()
	_ = store.SetTunnel(context.Background(), &TunnelInfo{
		ID:        "tunnel-123",
		Status:    TunnelStatusConnected,
		Provider:  TunnelProviderCloudflare,
		PublicURL: "https://test.tunnel.example.com",
		StartedAt: time.Now().Add(-1 * time.Hour),
	})

	r := NewStatusResource(store)
	result, err := r.Get(context.Background())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	resultMap := result.(map[string]any)
	if resultMap["enabled"].(bool) != true {
		t.Error("expected enabled=true for connected tunnel")
	}
	if resultMap["provider"].(string) != "cloudflare" {
		t.Errorf("expected provider 'cloudflare', got '%s'", resultMap["provider"])
	}
	if resultMap["uptime_seconds"].(int64) <= 0 {
		t.Error("expected positive uptime_seconds for running tunnel")
	}
}

func TestAuditResource_GetWithMultipleRecords(t *testing.T) {
	store := NewMemoryStore()
	for i := 0; i < 5; i++ {
		_ = store.AddAuditRecord(context.Background(), &AuditRecord{
			ID:        "audit-" + string(rune('0'+i)),
			Command:   "test command",
			ExitCode:  0,
			Timestamp: time.Now().Add(-time.Duration(i) * time.Minute),
			Duration:  10 * (i + 1),
		})
	}

	r := NewAuditResource(store)
	result, err := r.Get(context.Background())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	resultMap := result.(map[string]any)
	records := resultMap["records"].([]map[string]any)
	if len(records) != 5 {
		t.Errorf("expected 5 records, got %d", len(records))
	}
}

func TestStatusResource_WatchContextCancellation(t *testing.T) {
	store := createStoreWithTunnel()
	r := NewStatusResource(store)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed immediately on cancelled context")
	}
}

func TestAuditResource_WatchContextCancellation(t *testing.T) {
	store := createStoreWithAuditRecords()
	r := NewAuditResource(store)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed immediately on cancelled context")
	}
}

func TestStatusResource_GetDisconnectedTunnel(t *testing.T) {
	store := NewMemoryStore()
	_ = store.SetTunnel(context.Background(), &TunnelInfo{
		ID:        "tunnel-test",
		Status:    TunnelStatusDisconnected,
		Provider:  TunnelProviderCloudflare,
		PublicURL: "",
		StartedAt: time.Time{},
	})

	r := NewStatusResource(store)
	result, err := r.Get(context.Background())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	resultMap := result.(map[string]any)
	if resultMap["enabled"].(bool) != false {
		t.Error("expected enabled=false for disconnected tunnel")
	}
	if resultMap["status"].(string) != "disconnected" {
		t.Errorf("expected status 'disconnected', got '%s'", resultMap["status"])
	}
}
