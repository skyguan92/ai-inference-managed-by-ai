package eventbus

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type mockEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func (e *mockEvent) Type() string          { return e.eventType }
func (e *mockEvent) Domain() string        { return e.domain }
func (e *mockEvent) Payload() any          { return e.payload }
func (e *mockEvent) Timestamp() time.Time  { return e.timestamp }
func (e *mockEvent) CorrelationID() string { return e.correlationID }

func newMockEvent(eventType, domain string) *mockEvent {
	return &mockEvent{
		eventType:     eventType,
		domain:        domain,
		payload:       map[string]any{"test": "data"},
		timestamp:     time.Now(),
		correlationID: "test-correlation-id",
	}
}

func TestInMemoryEventBus_PublishSubscribe(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	var receivedCount int64
	var mu sync.Mutex
	receivedEvents := []unit.Event{}

	handler := func(event unit.Event) error {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
		atomic.AddInt64(&receivedCount, 1)
		return nil
	}

	_, err := bus.Subscribe(handler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	event := newMockEvent("test.event", "test")
	err = bus.Publish(event)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt64(&receivedCount) != 1 {
		t.Errorf("Expected 1 event received, got %d", receivedCount)
	}

	mu.Lock()
	if len(receivedEvents) != 1 {
		t.Errorf("Expected 1 event in slice, got %d", len(receivedEvents))
	}
	mu.Unlock()
}

func TestInMemoryEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	var counter int64

	handler := func(event unit.Event) error {
		atomic.AddInt64(&counter, 1)
		return nil
	}

	for i := 0; i < 5; i++ {
		_, err := bus.Subscribe(handler)
		if err != nil {
			t.Fatalf("Subscribe failed: %v", err)
		}
	}

	event := newMockEvent("test.event", "test")
	err := bus.Publish(event)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt64(&counter) != 5 {
		t.Errorf("Expected 5 events received, got %d", counter)
	}
}

func TestInMemoryEventBus_FilterByType(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	var receivedCount int64

	handler := func(event unit.Event) error {
		atomic.AddInt64(&receivedCount, 1)
		return nil
	}

	_, err := bus.Subscribe(handler, FilterByType("target.event"))
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	bus.Publish(newMockEvent("target.event", "test"))
	bus.Publish(newMockEvent("other.event", "test"))
	bus.Publish(newMockEvent("target.event", "test"))
	bus.Publish(newMockEvent("another.event", "test"))

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt64(&receivedCount) != 2 {
		t.Errorf("Expected 2 events received, got %d", receivedCount)
	}
}

func TestInMemoryEventBus_FilterByDomain(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	var receivedCount int64

	handler := func(event unit.Event) error {
		atomic.AddInt64(&receivedCount, 1)
		return nil
	}

	_, err := bus.Subscribe(handler, FilterByDomain("model"))
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	bus.Publish(newMockEvent("event1", "model"))
	bus.Publish(newMockEvent("event2", "engine"))
	bus.Publish(newMockEvent("event3", "model"))
	bus.Publish(newMockEvent("event4", "device"))

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt64(&receivedCount) != 2 {
		t.Errorf("Expected 2 events received, got %d", receivedCount)
	}
}

func TestInMemoryEventBus_FilterByTypes(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	var receivedCount int64

	handler := func(event unit.Event) error {
		atomic.AddInt64(&receivedCount, 1)
		return nil
	}

	_, err := bus.Subscribe(handler, FilterByTypes("event1", "event2"))
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	bus.Publish(newMockEvent("event1", "test"))
	bus.Publish(newMockEvent("event2", "test"))
	bus.Publish(newMockEvent("event3", "test"))
	bus.Publish(newMockEvent("event4", "test"))

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt64(&receivedCount) != 2 {
		t.Errorf("Expected 2 events received, got %d", receivedCount)
	}
}

func TestInMemoryEventBus_FilterByDomains(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	var receivedCount int64

	handler := func(event unit.Event) error {
		atomic.AddInt64(&receivedCount, 1)
		return nil
	}

	_, err := bus.Subscribe(handler, FilterByDomains("model", "engine"))
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	bus.Publish(newMockEvent("event1", "model"))
	bus.Publish(newMockEvent("event2", "engine"))
	bus.Publish(newMockEvent("event3", "device"))
	bus.Publish(newMockEvent("event4", "inference"))

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt64(&receivedCount) != 2 {
		t.Errorf("Expected 2 events received, got %d", receivedCount)
	}
}

func TestInMemoryEventBus_CombinedFilters(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	var receivedEvents []unit.Event
	var mu sync.Mutex

	handler := func(event unit.Event) error {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
		return nil
	}

	_, err := bus.Subscribe(handler, FilterByDomain("model"), FilterByType("model.created"))
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	bus.Publish(newMockEvent("model.created", "model"))
	bus.Publish(newMockEvent("model.deleted", "model"))
	bus.Publish(newMockEvent("model.created", "engine"))
	bus.Publish(newMockEvent("model.created", "model"))

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if len(receivedEvents) != 2 {
		t.Errorf("Expected 2 events received, got %d", len(receivedEvents))
	}
	for _, e := range receivedEvents {
		if e.Domain() != "model" || e.Type() != "model.created" {
			t.Errorf("Event doesn't match filter: domain=%s, type=%s", e.Domain(), e.Type())
		}
	}
	mu.Unlock()
}

func TestInMemoryEventBus_Unsubscribe(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	var counter int64

	handler := func(event unit.Event) error {
		atomic.AddInt64(&counter, 1)
		return nil
	}

	subID, err := bus.Subscribe(handler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	bus.Publish(newMockEvent("test.event", "test"))
	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt64(&counter) != 1 {
		t.Errorf("Expected 1 event received before unsubscribe, got %d", counter)
	}

	err = bus.Unsubscribe(subID)
	if err != nil {
		t.Fatalf("Unsubscribe failed: %v", err)
	}

	bus.Publish(newMockEvent("test.event", "test"))
	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt64(&counter) != 1 {
		t.Errorf("Expected 1 event received after unsubscribe, got %d", counter)
	}
}

func TestInMemoryEventBus_UnsubscribeNotFound(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	err := bus.Unsubscribe("non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent subscription")
	}
}

func TestInMemoryEventBus_PublishNilEvent(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	err := bus.Publish(nil)
	if err == nil {
		t.Error("Expected error for nil event")
	}
}

func TestInMemoryEventBus_SubscribeNilHandler(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	_, err := bus.Subscribe(nil)
	if err == nil {
		t.Error("Expected error for nil handler")
	}
}

func TestInMemoryEventBus_Close(t *testing.T) {
	bus := NewInMemoryEventBus()

	var counter int64
	handler := func(event unit.Event) error {
		atomic.AddInt64(&counter, 1)
		return nil
	}

	_, err := bus.Subscribe(handler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	bus.Publish(newMockEvent("test.event", "test"))
	time.Sleep(50 * time.Millisecond)

	err = bus.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	err = bus.Publish(newMockEvent("test.event", "test"))
	if err == nil {
		t.Error("Expected error when publishing to closed bus")
	}

	_, err = bus.Subscribe(handler)
	if err == nil {
		t.Error("Expected error when subscribing to closed bus")
	}
}

func TestInMemoryEventBus_CloseIdempotent(t *testing.T) {
	bus := NewInMemoryEventBus()

	err := bus.Close()
	if err != nil {
		t.Fatalf("First close failed: %v", err)
	}

	err = bus.Close()
	if err != nil {
		t.Fatalf("Second close failed: %v", err)
	}
}

func TestInMemoryEventBus_Options(t *testing.T) {
	bus := NewInMemoryEventBus(
		WithBufferSize(500),
		WithWorkerCount(2),
	)
	defer bus.Close()

	if bus.bufferSize != 500 {
		t.Errorf("Expected buffer size 500, got %d", bus.bufferSize)
	}

	if bus.workerCount != 2 {
		t.Errorf("Expected worker count 2, got %d", bus.workerCount)
	}
}

func TestInMemoryEventBus_Concurrency(t *testing.T) {
	bus := NewInMemoryEventBus(
		WithBufferSize(10000),
		WithWorkerCount(8),
	)
	defer bus.Close()

	var eventCount int64
	handler := func(event unit.Event) error {
		atomic.AddInt64(&eventCount, 1)
		return nil
	}

	_, err := bus.Subscribe(handler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	var wg sync.WaitGroup
	numPublishers := 10
	eventsPerPublisher := 100

	for i := 0; i < numPublishers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < eventsPerPublisher; j++ {
				bus.Publish(newMockEvent("test.event", "test"))
			}
		}()
	}

	wg.Wait()
	time.Sleep(500 * time.Millisecond)

	expected := int64(numPublishers * eventsPerPublisher)
	if atomic.LoadInt64(&eventCount) != expected {
		t.Errorf("Expected %d events, got %d", expected, eventCount)
	}
}

func TestInMemoryEventBus_MultipleFiltersAllMustMatch(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	var receivedCount int64

	handler := func(event unit.Event) error {
		atomic.AddInt64(&receivedCount, 1)
		return nil
	}

	onlyEngineStarted := func(event unit.Event) bool {
		return event.Domain() == "engine" && event.Type() == "engine.started"
	}

	_, err := bus.Subscribe(handler, onlyEngineStarted)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	bus.Publish(newMockEvent("engine.started", "engine"))
	bus.Publish(newMockEvent("engine.stopped", "engine"))
	bus.Publish(newMockEvent("engine.started", "model"))
	bus.Publish(newMockEvent("model.created", "engine"))

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt64(&receivedCount) != 1 {
		t.Errorf("Expected 1 event received, got %d", receivedCount)
	}
}

func TestInMemoryEventBus_CustomFilter(t *testing.T) {
	bus := NewInMemoryEventBus()
	defer bus.Close()

	var receivedCount int64

	handler := func(event unit.Event) error {
		atomic.AddInt64(&receivedCount, 1)
		return nil
	}

	customFilter := func(event unit.Event) bool {
		return event.CorrelationID() == "target-correlation-id"
	}

	_, err := bus.Subscribe(handler, customFilter)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	bus.Publish(&mockEvent{
		eventType:     "test.event",
		domain:        "test",
		correlationID: "target-correlation-id",
	})
	bus.Publish(&mockEvent{
		eventType:     "test.event",
		domain:        "test",
		correlationID: "other-correlation-id",
	})

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt64(&receivedCount) != 1 {
		t.Errorf("Expected 1 event received, got %d", receivedCount)
	}
}

func TestInMemoryEventBus_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	bus := &InMemoryEventBus{
		subscribers: make(map[SubscriptionID]*subscription),
		eventChan:   make(chan unit.Event, 100),
		workerCount: 2,
		bufferSize:  100,
		ctx:         ctx,
		cancel:      cancel,
	}

	for i := 0; i < bus.workerCount; i++ {
		bus.wg.Add(1)
		go bus.worker()
	}

	var counter int64
	handler := func(event unit.Event) error {
		atomic.AddInt64(&counter, 1)
		return nil
	}

	bus.Subscribe(handler)

	bus.Publish(newMockEvent("test", "test"))
	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt64(&counter) != 1 {
		t.Errorf("Expected 1 event before cancellation, got %d", counter)
	}

	cancel()
	bus.wg.Wait()
}
