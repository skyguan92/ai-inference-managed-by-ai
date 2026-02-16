package remote

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrTunnelNotFound       = errors.New("tunnel not found")
	ErrTunnelAlreadyExists  = errors.New("tunnel already exists")
	ErrInvalidInput         = errors.New("invalid input")
	ErrProviderNotSet       = errors.New("remote provider not set")
	ErrTunnelNotConnected   = errors.New("tunnel not connected")
	ErrTunnelAlreadyEnabled = errors.New("tunnel already enabled")
)

type RemoteStore interface {
	GetTunnel(ctx context.Context) (*TunnelInfo, error)
	SetTunnel(ctx context.Context, tunnel *TunnelInfo) error
	DeleteTunnel(ctx context.Context) error
	AddAuditRecord(ctx context.Context, record *AuditRecord) error
	ListAuditRecords(ctx context.Context, filter AuditFilter) ([]AuditRecord, error)
}

type MemoryStore struct {
	tunnel       *TunnelInfo
	auditRecords []AuditRecord
	mu           sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		auditRecords: make([]AuditRecord, 0),
	}
}

func (s *MemoryStore) GetTunnel(ctx context.Context) (*TunnelInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.tunnel == nil {
		return nil, ErrTunnelNotFound
	}
	return s.tunnel, nil
}

func (s *MemoryStore) SetTunnel(ctx context.Context, tunnel *TunnelInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tunnel = tunnel
	return nil
}

func (s *MemoryStore) DeleteTunnel(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tunnel == nil {
		return ErrTunnelNotFound
	}
	s.tunnel = nil
	return nil
}

func (s *MemoryStore) AddAuditRecord(ctx context.Context, record *AuditRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.auditRecords = append(s.auditRecords, *record)
	return nil
}

func (s *MemoryStore) ListAuditRecords(ctx context.Context, filter AuditFilter) ([]AuditRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []AuditRecord
	for i := len(s.auditRecords) - 1; i >= 0; i-- {
		record := s.auditRecords[i]
		if !filter.Since.IsZero() && record.Timestamp.Before(filter.Since) {
			continue
		}
		result = append(result, record)
		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}

	return result, nil
}

type RemoteProvider interface {
	Enable(ctx context.Context, config TunnelConfig) (*TunnelInfo, error)
	Disable(ctx context.Context) error
	Exec(ctx context.Context, command string, timeout int) (*ExecResult, error)
}

type MockProvider struct {
	enableErr  error
	disableErr error
	execErr    error
	execResult *ExecResult
	tunnelInfo *TunnelInfo
}

func (m *MockProvider) Enable(ctx context.Context, config TunnelConfig) (*TunnelInfo, error) {
	if m.enableErr != nil {
		return nil, m.enableErr
	}
	if m.tunnelInfo != nil {
		return m.tunnelInfo, nil
	}
	return &TunnelInfo{
		ID:        "tunnel-" + uuid.New().String()[:8],
		Status:    TunnelStatusConnected,
		Provider:  config.Provider,
		PublicURL: "https://test.tunnel.example.com",
		StartedAt: time.Now(),
	}, nil
}

func (m *MockProvider) Disable(ctx context.Context) error {
	return m.disableErr
}

func (m *MockProvider) Exec(ctx context.Context, command string, timeout int) (*ExecResult, error) {
	if m.execErr != nil {
		return nil, m.execErr
	}
	if m.execResult != nil {
		return m.execResult, nil
	}
	return &ExecResult{
		Stdout:   "command output",
		Stderr:   "",
		ExitCode: 0,
	}, nil
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

func ptrFloat(v float64) *float64 {
	return &v
}
