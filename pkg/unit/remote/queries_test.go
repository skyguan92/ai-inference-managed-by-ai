package remote

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestStatusQuery_Name(t *testing.T) {
	q := NewStatusQuery(nil)
	if q.Name() != "remote.status" {
		t.Errorf("expected name 'remote.status', got '%s'", q.Name())
	}
}

func TestStatusQuery_Domain(t *testing.T) {
	q := NewStatusQuery(nil)
	if q.Domain() != "remote" {
		t.Errorf("expected domain 'remote', got '%s'", q.Domain())
	}
}

func TestStatusQuery_Schemas(t *testing.T) {
	q := NewStatusQuery(nil)

	inputSchema := q.InputSchema()
	if inputSchema.Type != "object" {
		t.Errorf("expected input schema type 'object', got '%s'", inputSchema.Type)
	}

	outputSchema := q.OutputSchema()
	if outputSchema.Type != "object" {
		t.Errorf("expected output schema type 'object', got '%s'", outputSchema.Type)
	}
}

func TestStatusQuery_Execute(t *testing.T) {
	tests := []struct {
		name    string
		store   RemoteStore
		input   any
		wantErr bool
	}{
		{
			name:    "status with tunnel",
			store:   createStoreWithTunnel(),
			input:   map[string]any{},
			wantErr: false,
		},
		{
			name:    "status without tunnel",
			store:   NewMemoryStore(),
			input:   map[string]any{},
			wantErr: false,
		},
		{
			name:    "nil store",
			store:   nil,
			input:   map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewStatusQuery(tt.store)
			result, err := q.Execute(context.Background(), tt.input)

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
			if _, exists := resultMap["uptime_seconds"]; !exists {
				t.Error("expected field 'uptime_seconds' not found")
			}
		})
	}
}

func TestAuditQuery_Name(t *testing.T) {
	q := NewAuditQuery(nil)
	if q.Name() != "remote.audit" {
		t.Errorf("expected name 'remote.audit', got '%s'", q.Name())
	}
}

func TestAuditQuery_Domain(t *testing.T) {
	q := NewAuditQuery(nil)
	if q.Domain() != "remote" {
		t.Errorf("expected domain 'remote', got '%s'", q.Domain())
	}
}

func TestAuditQuery_Schemas(t *testing.T) {
	q := NewAuditQuery(nil)

	inputSchema := q.InputSchema()
	if inputSchema.Type != "object" {
		t.Errorf("expected input schema type 'object', got '%s'", inputSchema.Type)
	}

	outputSchema := q.OutputSchema()
	if outputSchema.Type != "object" {
		t.Errorf("expected output schema type 'object', got '%s'", outputSchema.Type)
	}
}

func TestAuditQuery_Execute(t *testing.T) {
	tests := []struct {
		name    string
		store   RemoteStore
		input   any
		wantErr bool
	}{
		{
			name:    "audit with records",
			store:   createStoreWithAuditRecords(),
			input:   map[string]any{},
			wantErr: false,
		},
		{
			name:    "audit with limit",
			store:   createStoreWithAuditRecords(),
			input:   map[string]any{"limit": 1},
			wantErr: false,
		},
		{
			name:    "audit with since filter",
			store:   createStoreWithAuditRecords(),
			input:   map[string]any{"since": time.Now().Add(-24 * time.Hour).Format(time.RFC3339)},
			wantErr: false,
		},
		{
			name:    "audit empty store",
			store:   NewMemoryStore(),
			input:   map[string]any{},
			wantErr: false,
		},
		{
			name:    "nil store",
			store:   nil,
			input:   map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewAuditQuery(tt.store)
			result, err := q.Execute(context.Background(), tt.input)

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

func TestQuery_Description(t *testing.T) {
	if NewStatusQuery(nil).Description() == "" {
		t.Error("expected non-empty description for StatusQuery")
	}
	if NewAuditQuery(nil).Description() == "" {
		t.Error("expected non-empty description for AuditQuery")
	}
}

func TestQuery_Examples(t *testing.T) {
	if len(NewStatusQuery(nil).Examples()) == 0 {
		t.Error("expected at least one example for StatusQuery")
	}
	if len(NewAuditQuery(nil).Examples()) == 0 {
		t.Error("expected at least one example for AuditQuery")
	}
}

func TestQueryImplementsInterface(t *testing.T) {
	var _ unit.Query = NewStatusQuery(nil)
	var _ unit.Query = NewAuditQuery(nil)
}

func createStoreWithAuditRecords() RemoteStore {
	store := NewMemoryStore()
	_ = store.AddAuditRecord(context.Background(), &AuditRecord{
		ID:        "audit-1",
		Command:   "ls -la",
		ExitCode:  0,
		Timestamp: time.Now().Add(-1 * time.Hour),
		Duration:  50,
	})
	_ = store.AddAuditRecord(context.Background(), &AuditRecord{
		ID:        "audit-2",
		Command:   "cat /etc/hosts",
		ExitCode:  0,
		Timestamp: time.Now().Add(-30 * time.Minute),
		Duration:  20,
	})
	return store
}

func TestStatusQuery_ExecuteWithDisconnectedTunnel(t *testing.T) {
	store := NewMemoryStore()
	_ = store.SetTunnel(context.Background(), &TunnelInfo{
		ID:        "tunnel-test",
		Status:    TunnelStatusDisconnected,
		Provider:  TunnelProviderCloudflare,
		PublicURL: "",
		StartedAt: time.Time{},
	})

	q := NewStatusQuery(store)
	result, err := q.Execute(context.Background(), map[string]any{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	resultMap := result.(map[string]any)
	if resultMap["enabled"].(bool) != false {
		t.Error("expected enabled=false for disconnected tunnel")
	}
}

func TestAuditQuery_ExecuteWithLimit(t *testing.T) {
	store := NewMemoryStore()
	for i := 0; i < 5; i++ {
		_ = store.AddAuditRecord(context.Background(), &AuditRecord{
			ID:        "audit-" + string(rune('0'+i)),
			Command:   "test",
			ExitCode:  0,
			Timestamp: time.Now().Add(-time.Duration(i) * time.Minute),
			Duration:  10,
		})
	}

	q := NewAuditQuery(store)
	result, err := q.Execute(context.Background(), map[string]any{"limit": 2})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	resultMap := result.(map[string]any)
	records := resultMap["records"].([]map[string]any)
	if len(records) > 2 {
		t.Errorf("expected at most 2 records, got %d", len(records))
	}
}

func TestStore_Error(t *testing.T) {
	store := &errorStore{}
	q := NewAuditQuery(store)
	_, err := q.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error from error store")
	}
}

type errorStore struct{}

func (s *errorStore) GetTunnel(ctx context.Context) (*TunnelInfo, error) {
	return nil, errors.New("store error")
}
func (s *errorStore) SetTunnel(ctx context.Context, tunnel *TunnelInfo) error {
	return errors.New("store error")
}
func (s *errorStore) DeleteTunnel(ctx context.Context) error {
	return errors.New("store error")
}
func (s *errorStore) AddAuditRecord(ctx context.Context, record *AuditRecord) error {
	return errors.New("store error")
}
func (s *errorStore) ListAuditRecords(ctx context.Context, filter AuditFilter) ([]AuditRecord, error) {
	return nil, errors.New("store error")
}
