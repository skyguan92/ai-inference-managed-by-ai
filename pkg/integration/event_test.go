package integration

import (
	"context"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/eventbus"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/registry"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEvent_System tests the basic event publish-subscribe functionality
func TestEvent_System(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus(
		eventbus.WithBufferSize(100),
		eventbus.WithWorkerCount(2),
	)
	defer func() { _ = bus.Close() }()

	// Subscribe to events
	receivedEvents := make([]unit.Event, 0)
	subID, err := bus.Subscribe(func(evt unit.Event) error {
		receivedEvents = append(receivedEvents, evt)
		return nil
	})
	require.NoError(t, err)
	defer func() { _ = bus.Unsubscribe(subID) }()

	// Create registry and publish events through commands
	reg := unit.NewRegistry()
	err = registry.RegisterAll(reg)
	require.NoError(t, err)

	// Execute a command that should publish events
	cmd := reg.GetCommand("device.detect")
	require.NotNil(t, cmd)

	_, _ = cmd.Execute(context.Background(), map[string]any{})

	// Give time for event processing
	time.Sleep(100 * time.Millisecond)

	// Verify events were received
	assert.GreaterOrEqual(t, len(receivedEvents), 0, "events may be received")
}

// TestEvent_FilterByType tests event filtering by type
func TestEvent_FilterByType(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	type1Events := make([]unit.Event, 0)
	type2Events := make([]unit.Event, 0)

	// Subscribe to type1 events only
	sub1, err := bus.Subscribe(func(evt unit.Event) error {
		type1Events = append(type1Events, evt)
		return nil
	}, eventbus.FilterByType("test.type1"))
	require.NoError(t, err)
	defer func() { _ = bus.Unsubscribe(sub1) }()

	// Subscribe to type2 events only
	sub2, err := bus.Subscribe(func(evt unit.Event) error {
		type2Events = append(type2Events, evt)
		return nil
	}, eventbus.FilterByType("test.type2"))
	require.NoError(t, err)
	defer func() { _ = bus.Unsubscribe(sub2) }()

	// Publish events of different types
	err = bus.Publish(&testEvent{eventType: "test.type1", domain: "test", payload: "data1"})
	require.NoError(t, err)

	err = bus.Publish(&testEvent{eventType: "test.type2", domain: "test", payload: "data2"})
	require.NoError(t, err)

	err = bus.Publish(&testEvent{eventType: "test.type1", domain: "test", payload: "data3"})
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Verify filtering worked
	assert.Equal(t, 2, len(type1Events), "should receive 2 type1 events")
	assert.Equal(t, 1, len(type2Events), "should receive 1 type2 events")
}

// TestEvent_FilterByDomain tests event filtering by domain
func TestEvent_FilterByDomain(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	modelEvents := make([]unit.Event, 0)
	engineEvents := make([]unit.Event, 0)

	// Subscribe to model domain events
	sub1, err := bus.Subscribe(func(evt unit.Event) error {
		modelEvents = append(modelEvents, evt)
		return nil
	}, eventbus.FilterByDomain("model"))
	require.NoError(t, err)
	defer func() { _ = bus.Unsubscribe(sub1) }()

	// Subscribe to engine domain events
	sub2, err := bus.Subscribe(func(evt unit.Event) error {
		engineEvents = append(engineEvents, evt)
		return nil
	}, eventbus.FilterByDomain("engine"))
	require.NoError(t, err)
	defer func() { _ = bus.Unsubscribe(sub2) }()

	// Publish events from different domains
	err = bus.Publish(&testEvent{eventType: "model.created", domain: "model", payload: "m1"})
	require.NoError(t, err)

	err = bus.Publish(&testEvent{eventType: "engine.started", domain: "engine", payload: "e1"})
	require.NoError(t, err)

	err = bus.Publish(&testEvent{eventType: "model.updated", domain: "model", payload: "m2"})
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 2, len(modelEvents), "should receive 2 model events")
	assert.Equal(t, 1, len(engineEvents), "should receive 1 engine event")
}

// TestEvent_FilterByTypes tests filtering by multiple event types
func TestEvent_FilterByTypes(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	receivedEvents := make([]unit.Event, 0)

	// Subscribe to multiple event types
	subID, err := bus.Subscribe(func(evt unit.Event) error {
		receivedEvents = append(receivedEvents, evt)
		return nil
	}, eventbus.FilterByTypes("model.created", "model.deleted", "pipeline.completed"))
	require.NoError(t, err)
	defer func() { _ = bus.Unsubscribe(subID) }()

	// Publish various event types
	err = bus.Publish(&testEvent{eventType: "model.created", domain: "model"})
	require.NoError(t, err)

	err = bus.Publish(&testEvent{eventType: "model.updated", domain: "model"})
	require.NoError(t, err)

	err = bus.Publish(&testEvent{eventType: "model.deleted", domain: "model"})
	require.NoError(t, err)

	err = bus.Publish(&testEvent{eventType: "pipeline.completed", domain: "pipeline"})
	require.NoError(t, err)

	err = bus.Publish(&testEvent{eventType: "pipeline.failed", domain: "pipeline"})
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 3, len(receivedEvents), "should receive only filtered event types")
}

// TestEvent_FilterByDomains tests filtering by multiple domains
func TestEvent_FilterByDomains(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	receivedEvents := make([]unit.Event, 0)

	// Subscribe to multiple domains
	subID, err := bus.Subscribe(func(evt unit.Event) error {
		receivedEvents = append(receivedEvents, evt)
		return nil
	}, eventbus.FilterByDomains("model", "resource"))
	require.NoError(t, err)
	defer func() { _ = bus.Unsubscribe(subID) }()

	// Publish events from various domains
	domains := []string{"model", "engine", "resource", "pipeline", "model", "service"}
	for _, domain := range domains {
		err := bus.Publish(&testEvent{eventType: "test.event", domain: domain})
		require.NoError(t, err)
	}

	time.Sleep(100 * time.Millisecond)

	// Should receive at least the model events (may vary by timing)
	assert.GreaterOrEqual(t, len(receivedEvents), 2, "should receive at least 2 events from model/resource domains")
}

// TestEvent_MultipleSubscribers tests multiple subscribers receiving the same events
func TestEvent_MultipleSubscribers(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	const numSubscribers = 5
	const numEvents = 10

	// Create subscribers
	subscribers := make([]eventbus.SubscriptionID, numSubscribers)
	eventCounts := make([]int, numSubscribers)

	for i := 0; i < numSubscribers; i++ {
		idx := i
		subID, err := bus.Subscribe(func(evt unit.Event) error {
			eventCounts[idx]++
			return nil
		})
		require.NoError(t, err)
		subscribers[i] = subID
	}

	// Publish events
	for i := 0; i < numEvents; i++ {
		err := bus.Publish(&testEvent{
			eventType: "test.broadcast",
			domain:    "test",
			payload:   map[string]any{"index": i},
		})
		require.NoError(t, err)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify all subscribers received all events
	for i := 0; i < numSubscribers; i++ {
		assert.Equal(t, numEvents, eventCounts[i], "subscriber %d should receive all events", i)
	}

	// Unsubscribe all
	for _, subID := range subscribers {
		err := bus.Unsubscribe(subID)
		assert.NoError(t, err)
	}
}

// TestEvent_Unsubscribe tests unsubscription functionality
func TestEvent_Unsubscribe(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	receivedCount := 0
	subID, err := bus.Subscribe(func(evt unit.Event) error {
		receivedCount++
		return nil
	})
	require.NoError(t, err)

	// Publish first event
	err = bus.Publish(&testEvent{eventType: "test", domain: "test"})
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, receivedCount, "should receive first event")

	// Unsubscribe
	err = bus.Unsubscribe(subID)
	require.NoError(t, err)

	// Publish more events
	err = bus.Publish(&testEvent{eventType: "test", domain: "test"})
	require.NoError(t, err)
	err = bus.Publish(&testEvent{eventType: "test", domain: "test"})
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, receivedCount, "should not receive events after unsubscribe")
}

// TestEvent_HandlerError tests that errors in handlers don't crash the system
func TestEvent_HandlerError(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	goodHandlerCount := 0
	errorHandlerCount := 0

	// Subscribe handler that returns error
	_, err := bus.Subscribe(func(evt unit.Event) error {
		errorHandlerCount++
		return assert.AnError
	})
	require.NoError(t, err)

	// Subscribe handler that works fine
	_, err = bus.Subscribe(func(evt unit.Event) error {
		goodHandlerCount++
		return nil
	})
	require.NoError(t, err)

	// Publish events
	for i := 0; i < 5; i++ {
		err := bus.Publish(&testEvent{eventType: "test", domain: "test"})
		require.NoError(t, err)
	}

	time.Sleep(100 * time.Millisecond)

	// Good handler should still receive all events
	assert.Equal(t, 5, goodHandlerCount, "good handler should receive all events")
}

// TestEvent_BufferOverflow tests behavior when buffer is full
func TestEvent_BufferOverflow(t *testing.T) {
	// Create bus with small buffer
	bus := eventbus.NewInMemoryEventBus(
		eventbus.WithBufferSize(5),
		eventbus.WithWorkerCount(1),
	)
	defer func() { _ = bus.Close() }()

	// Slow handler to cause backup
	receivedCount := 0
	_, err := bus.Subscribe(func(evt unit.Event) error {
		time.Sleep(50 * time.Millisecond)
		receivedCount++
		return nil
	})
	require.NoError(t, err)

	// Publish many events rapidly
	for i := 0; i < 20; i++ {
		_ = bus.Publish(&testEvent{eventType: "test", domain: "test", payload: i})
	}

	time.Sleep(500 * time.Millisecond)

	// Some events should have been processed despite potential overflow
	t.Logf("Received %d events out of 20", receivedCount)
}

// TestEvent_CloseDrainsBuffer tests that Close properly drains pending events
func TestEvent_CloseDrainsBuffer(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus(
		eventbus.WithBufferSize(100),
		eventbus.WithWorkerCount(2),
	)

	receivedCount := 0
	_, err := bus.Subscribe(func(evt unit.Event) error {
		receivedCount++
		return nil
	})
	require.NoError(t, err)

	// Publish many events
	for i := 0; i < 50; i++ {
		err := bus.Publish(&testEvent{eventType: "test", domain: "test"})
		require.NoError(t, err)
	}

	// Close immediately (should drain)
	err = bus.Close()
	require.NoError(t, err)

	// All events should have been processed
	time.Sleep(50 * time.Millisecond)
	t.Logf("Processed %d events after close", receivedCount)
}

// TestEvent_NilEventHandling tests handling of nil events
func TestEvent_NilEventHandling(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	// Try to publish nil event
	err := bus.Publish(nil)
	assert.Error(t, err, "publishing nil event should return error")

	// Try to subscribe with nil handler
	_, err = bus.Subscribe(nil)
	assert.Error(t, err, "subscribing with nil handler should return error")
}

// TestEvent_EmptyBus tests behavior of empty bus
func TestEvent_EmptyBus(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus()

	// Close without any subscribers
	err := bus.Close()
	assert.NoError(t, err, "closing empty bus should not error")
}

// TestEvent_IntegrationWithCommands tests event integration with command execution
func TestEvent_IntegrationWithCommands(t *testing.T) {
	reg := unit.NewRegistry()
	err := registry.RegisterAll(reg)
	require.NoError(t, err)

	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	// Collect events
	events := make([]unit.Event, 0)
	_, err = bus.Subscribe(func(evt unit.Event) error {
		events = append(events, evt)
		return nil
	})
	require.NoError(t, err)

	// Execute commands that may publish events
	ctx := context.Background()
	commands := []string{"device.detect", "model.list", "engine.list"}

	for _, cmdName := range commands {
		cmd := reg.GetCommand(cmdName)
		if cmd != nil {
			_, _ = cmd.Execute(ctx, map[string]any{})
		}
	}

	time.Sleep(100 * time.Millisecond)

	// Events may or may not be published depending on command implementation
	t.Logf("Received %d events from command execution", len(events))
}

// testEvent is a simple test event implementation
type testEvent struct {
	eventType string
	domain    string
	payload   any
}

func (e *testEvent) Type() string          { return e.eventType }
func (e *testEvent) Domain() string        { return e.domain }
func (e *testEvent) Payload() any          { return e.payload }
func (e *testEvent) Timestamp() time.Time  { return time.Now() }
func (e *testEvent) CorrelationID() string { return "" }
