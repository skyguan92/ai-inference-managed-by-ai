package eventbus

import (
	"context"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"database/sql"
	_ "modernc.org/sqlite"
)

type testEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func (e *testEvent) Type() string          { return e.eventType }
func (e *testEvent) Domain() string        { return e.domain }
func (e *testEvent) Payload() any          { return e.payload }
func (e *testEvent) Timestamp() time.Time  { return e.timestamp }
func (e *testEvent) CorrelationID() string { return e.correlationID }

var _ unit.Event = (*testEvent)(nil)

func setupTestDB(t *testing.T) *SQLiteEventStore {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`
		CREATE TABLE events (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			domain TEXT NOT NULL,
			correlation_id TEXT,
			payload BLOB,
			timestamp INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX idx_events_domain ON events(domain);
		CREATE INDEX idx_events_type ON events(type);
		CREATE INDEX idx_events_correlation ON events(correlation_id);
		CREATE INDEX idx_events_timestamp ON events(timestamp);
	`)
	require.NoError(t, err)

	return NewSQLiteEventStore(db)
}

func TestSQLiteEventStore_Save(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	event := &testEvent{
		eventType:     "model.created",
		domain:        "model",
		payload:       map[string]string{"name": "test-model"},
		timestamp:     time.Now(),
		correlationID: "corr-123",
	}

	err := store.Save(ctx, event)
	require.NoError(t, err)
}

func TestSQLiteEventStore_SaveBatch(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	events := []unit.Event{
		&testEvent{eventType: "model.created", domain: "model", payload: map[string]string{"name": "model1"}, timestamp: time.Now()},
		&testEvent{eventType: "model.updated", domain: "model", payload: map[string]string{"name": "model2"}, timestamp: time.Now()},
		&testEvent{eventType: "engine.started", domain: "engine", payload: map[string]string{"name": "engine1"}, timestamp: time.Now()},
	}

	err := store.SaveBatch(ctx, events)
	require.NoError(t, err)

	results, err := store.Query(ctx, EventQueryFilter{})
	require.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestSQLiteEventStore_Query(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	now := time.Now()
	events := []unit.Event{
		&testEvent{eventType: "model.created", domain: "model", correlationID: "corr-1", payload: map[string]string{}, timestamp: now.Add(-time.Hour)},
		&testEvent{eventType: "model.updated", domain: "model", correlationID: "corr-1", payload: map[string]string{}, timestamp: now.Add(-30 * time.Minute)},
		&testEvent{eventType: "engine.started", domain: "engine", correlationID: "corr-2", payload: map[string]string{}, timestamp: now},
	}

	for _, e := range events {
		err := store.Save(ctx, e)
		require.NoError(t, err)
	}

	t.Run("query by domain", func(t *testing.T) {
		results, err := store.Query(ctx, EventQueryFilter{Domain: "model"})
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("query by type", func(t *testing.T) {
		results, err := store.Query(ctx, EventQueryFilter{Type: "model.created"})
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("query by correlation ID", func(t *testing.T) {
		results, err := store.Query(ctx, EventQueryFilter{CorrelationID: "corr-1"})
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("query with limit", func(t *testing.T) {
		results, err := store.Query(ctx, EventQueryFilter{Limit: 1})
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("query by time range", func(t *testing.T) {
		results, err := store.Query(ctx, EventQueryFilter{
			StartTime: now.Add(-45 * time.Minute),
			EndTime:   now.Add(5 * time.Minute),
		})
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})
}

func TestSQLiteEventStore_GetByID(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	_, err := store.GetByID(ctx, "non-existent")
	assert.Error(t, err)
}
