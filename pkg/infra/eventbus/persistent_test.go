package eventbus

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"modernc.org/sqlite"
)

func setupPersistentBus(t *testing.T) *PersistentEventBus {
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)

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

	store := NewSQLiteEventStore(db)
	return NewPersistentEventBus(store, WithFlushPeriod(100*time.Millisecond))
}

func TestPersistentEventBus_Publish(t *testing.T) {
	bus := setupPersistentBus(t)
	defer bus.Close()

	event := &testEvent{
		eventType:     "test.event",
		domain:        "test",
		payload:       map[string]string{"key": "value"},
		timestamp:     time.Now(),
		correlationID: "corr-test",
	}

	err := bus.Publish(event)
	require.NoError(t, err)
}

func TestPersistentEventBus_PublishNil(t *testing.T) {
	bus := setupPersistentBus(t)
	defer bus.Close()

	err := bus.Publish(nil)
	assert.Error(t, err)
}

func TestPersistentEventBus_PublishClosed(t *testing.T) {
	bus := setupPersistentBus(t)
	bus.Close()

	event := &testEvent{eventType: "test", domain: "test", timestamp: time.Now()}
	err := bus.Publish(event)
	assert.Error(t, err)
}

func TestPersistentEventBus_SubscribeAndPublish(t *testing.T) {
	bus := setupPersistentBus(t)
	defer bus.Close()

	var received atomic.Int32
	handler := func(event unit.Event) error {
		received.Add(1)
		return nil
	}

	subID, err := bus.Subscribe(handler)
	require.NoError(t, err)
	assert.NotEmpty(t, subID)

	event := &testEvent{eventType: "test", domain: "test", timestamp: time.Now()}
	err = bus.Publish(event)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(1), received.Load())
}

func TestPersistentEventBus_SubscribeWithFilter(t *testing.T) {
	bus := setupPersistentBus(t)
	defer bus.Close()

	var modelEvents atomic.Int32
	var engineEvents atomic.Int32

	_, err := bus.Subscribe(func(event unit.Event) error {
		modelEvents.Add(1)
		return nil
	}, FilterByDomain("model"))
	require.NoError(t, err)

	_, err = bus.Subscribe(func(event unit.Event) error {
		engineEvents.Add(1)
		return nil
	}, FilterByDomain("engine"))
	require.NoError(t, err)

	bus.Publish(&testEvent{eventType: "test", domain: "model", timestamp: time.Now()})
	bus.Publish(&testEvent{eventType: "test", domain: "engine", timestamp: time.Now()})
	bus.Publish(&testEvent{eventType: "test", domain: "model", timestamp: time.Now()})

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, int32(2), modelEvents.Load())
	assert.Equal(t, int32(1), engineEvents.Load())
}

func TestPersistentEventBus_Unsubscribe(t *testing.T) {
	bus := setupPersistentBus(t)
	defer bus.Close()

	subID, err := bus.Subscribe(func(event unit.Event) error { return nil })
	require.NoError(t, err)

	err = bus.Unsubscribe(subID)
	require.NoError(t, err)

	err = bus.Unsubscribe("non-existent")
	assert.Error(t, err)
}

func TestPersistentEventBus_Query(t *testing.T) {
	bus := setupPersistentBus(t)
	defer bus.Close()

	ctx := context.Background()

	event := &testEvent{
		eventType:     "model.created",
		domain:        "model",
		correlationID: "query-test",
		payload:       map[string]string{"name": "test"},
		timestamp:     time.Now(),
	}

	err := bus.Publish(event)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	results, err := bus.Query(ctx, EventQueryFilter{Domain: "model"})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
}

func TestPersistentEventBus_Replay(t *testing.T) {
	bus := setupPersistentBus(t)
	defer bus.Close()

	correlationID := "replay-test"

	for i := 0; i < 3; i++ {
		event := &testEvent{
			eventType:     "step.completed",
			domain:        "workflow",
			correlationID: correlationID,
			payload:       map[string]int{"step": i},
			timestamp:     time.Now(),
		}
		bus.Publish(event)
	}

	time.Sleep(200 * time.Millisecond)

	var received []int
	var mu sync.Mutex
	handler := func(event unit.Event) error {
		mu.Lock()
		defer mu.Unlock()
		if p, ok := event.Payload().(map[string]int); ok {
			received = append(received, p["step"])
		}
		return nil
	}

	ctx := context.Background()
	err := bus.Replay(ctx, correlationID, handler)
	require.NoError(t, err)

	assert.Len(t, received, 3)
}

func TestPersistentEventBus_ReplayNilHandler(t *testing.T) {
	bus := setupPersistentBus(t)
	defer bus.Close()

	ctx := context.Background()
	err := bus.Replay(ctx, "test", nil)
	assert.Error(t, err)
}

func TestPersistentEventBus_Close(t *testing.T) {
	bus := setupPersistentBus(t)

	err := bus.Close()
	require.NoError(t, err)

	err = bus.Close()
	require.NoError(t, err)
}

func TestPersistentEventBus_ConcurrentPublish(t *testing.T) {
	bus := setupPersistentBus(t)
	defer bus.Close()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			event := &testEvent{
				eventType:     "test",
				domain:        "test",
				payload:       map[string]int{"idx": idx},
				timestamp:     time.Now(),
				correlationID: "concurrent",
			}
			bus.Publish(event)
		}(i)
	}

	wg.Wait()
	time.Sleep(300 * time.Millisecond)

	ctx := context.Background()
	results, err := bus.Query(ctx, EventQueryFilter{CorrelationID: "concurrent"})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 90)
}
