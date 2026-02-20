package remote

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

func newTestTunnel(id string) *TunnelInfo {
	return &TunnelInfo{
		ID:        id,
		Status:    TunnelStatusConnected,
		Provider:  TunnelProviderFRP,
		PublicURL: "https://test.example.com",
		StartedAt: time.Now(),
	}
}

func newTestAuditRecord(cmd string, exitCode int) *AuditRecord {
	return &AuditRecord{
		ID:        "rec-" + cmd,
		Command:   cmd,
		ExitCode:  exitCode,
		Timestamp: time.Now(),
		Duration:  100,
	}
}

// --- NewMemoryStore ---

func TestNewMemoryStore_NotNil(t *testing.T) {
	s := NewMemoryStore()
	require.NotNil(t, s)
}

// --- SetTunnel / GetTunnel ---

func TestMemoryStore_SetAndGetTunnel(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	tunnel := newTestTunnel("tunnel-1")
	require.NoError(t, s.SetTunnel(ctx, tunnel))

	got, err := s.GetTunnel(ctx)
	require.NoError(t, err)
	assert.Equal(t, "tunnel-1", got.ID)
	assert.Equal(t, TunnelStatusConnected, got.Status)
	assert.Equal(t, "https://test.example.com", got.PublicURL)
}

func TestMemoryStore_GetTunnel_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	_, err := s.GetTunnel(ctx)
	assert.ErrorIs(t, err, ErrTunnelNotFound)
}

func TestMemoryStore_SetTunnel_Overwrites(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.SetTunnel(ctx, newTestTunnel("tunnel-1")))

	tunnel2 := newTestTunnel("tunnel-2")
	require.NoError(t, s.SetTunnel(ctx, tunnel2))

	got, err := s.GetTunnel(ctx)
	require.NoError(t, err)
	assert.Equal(t, "tunnel-2", got.ID)
}

// --- DeleteTunnel ---

func TestMemoryStore_DeleteTunnel_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.SetTunnel(ctx, newTestTunnel("tunnel-1")))
	require.NoError(t, s.DeleteTunnel(ctx))

	_, err := s.GetTunnel(ctx)
	assert.ErrorIs(t, err, ErrTunnelNotFound)
}

func TestMemoryStore_DeleteTunnel_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	err := s.DeleteTunnel(ctx)
	assert.ErrorIs(t, err, ErrTunnelNotFound)
}

func TestMemoryStore_DeleteTunnel_CanSetAgain(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.SetTunnel(ctx, newTestTunnel("tunnel-1")))
	require.NoError(t, s.DeleteTunnel(ctx))

	// Can set a new tunnel after deletion
	require.NoError(t, s.SetTunnel(ctx, newTestTunnel("tunnel-2")))
	got, err := s.GetTunnel(ctx)
	require.NoError(t, err)
	assert.Equal(t, "tunnel-2", got.ID)
}

// --- AddAuditRecord / ListAuditRecords ---

func TestMemoryStore_AddAuditRecord_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	record := newTestAuditRecord("ls -la", 0)
	require.NoError(t, s.AddAuditRecord(ctx, record))

	records, err := s.ListAuditRecords(ctx, AuditFilter{})
	require.NoError(t, err)
	assert.Len(t, records, 1)
}

func TestMemoryStore_ListAuditRecords_All(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.AddAuditRecord(ctx, newTestAuditRecord("cmd1", 0)))
	require.NoError(t, s.AddAuditRecord(ctx, newTestAuditRecord("cmd2", 1)))
	require.NoError(t, s.AddAuditRecord(ctx, newTestAuditRecord("cmd3", 0)))

	records, err := s.ListAuditRecords(ctx, AuditFilter{})
	require.NoError(t, err)
	assert.Len(t, records, 3)
}

func TestMemoryStore_ListAuditRecords_ReverseOrder(t *testing.T) {
	// Records are returned in reverse insertion order (newest first)
	s := NewMemoryStore()
	ctx := context.Background()

	t1 := time.Now().Add(-3 * time.Second)
	t2 := time.Now().Add(-2 * time.Second)
	t3 := time.Now().Add(-1 * time.Second)

	rec1 := &AuditRecord{ID: "r1", Command: "cmd1", Timestamp: t1}
	rec2 := &AuditRecord{ID: "r2", Command: "cmd2", Timestamp: t2}
	rec3 := &AuditRecord{ID: "r3", Command: "cmd3", Timestamp: t3}

	require.NoError(t, s.AddAuditRecord(ctx, rec1))
	require.NoError(t, s.AddAuditRecord(ctx, rec2))
	require.NoError(t, s.AddAuditRecord(ctx, rec3))

	records, err := s.ListAuditRecords(ctx, AuditFilter{})
	require.NoError(t, err)
	require.Len(t, records, 3)
	// Newest first (r3 was added last)
	assert.Equal(t, "r3", records[0].ID)
	assert.Equal(t, "r2", records[1].ID)
	assert.Equal(t, "r1", records[2].ID)
}

func TestMemoryStore_ListAuditRecords_FilterBySince(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	old := &AuditRecord{ID: "old", Command: "old-cmd", Timestamp: time.Now().Add(-1 * time.Hour)}
	recent := &AuditRecord{ID: "recent", Command: "recent-cmd", Timestamp: time.Now()}

	require.NoError(t, s.AddAuditRecord(ctx, old))
	require.NoError(t, s.AddAuditRecord(ctx, recent))

	since := time.Now().Add(-30 * time.Minute)
	records, err := s.ListAuditRecords(ctx, AuditFilter{Since: since})
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "recent", records[0].ID)
}

func TestMemoryStore_ListAuditRecords_Limit(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		cmd := "cmd-" + string(rune('0'+i))
		require.NoError(t, s.AddAuditRecord(ctx, newTestAuditRecord(cmd, 0)))
	}

	records, err := s.ListAuditRecords(ctx, AuditFilter{Limit: 3})
	require.NoError(t, err)
	assert.Len(t, records, 3)
}

func TestMemoryStore_ListAuditRecords_Empty(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	records, err := s.ListAuditRecords(ctx, AuditFilter{})
	require.NoError(t, err)
	assert.Empty(t, records)
}

// --- Concurrent access ---

func TestMemoryStore_ConcurrentTunnelOps_NoRace(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	var wg sync.WaitGroup

	// Writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = s.SetTunnel(ctx, newTestTunnel("tunnel"))
		}(i)
	}

	// Readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = s.GetTunnel(ctx)
		}()
	}

	wg.Wait()
}

func TestMemoryStore_ConcurrentAuditOps_NoRace(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			record := newTestAuditRecord("concurrent-cmd", 0)
			_ = s.AddAuditRecord(ctx, record)
			_, _ = s.ListAuditRecords(ctx, AuditFilter{})
		}(i)
	}
	wg.Wait()
}
