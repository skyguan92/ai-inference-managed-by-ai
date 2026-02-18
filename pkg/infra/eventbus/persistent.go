package eventbus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type PersistentEventBus struct {
	memory      *InMemoryEventBus
	store       EventStore
	buffer      chan unit.Event
	batchSize   int
	flushPeriod time.Duration
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	closed      bool
	mu          sync.RWMutex
}

func NewPersistentEventBus(store EventStore, opts ...PersistentOption) *PersistentEventBus {
	config := &persistentConfig{
		bufferSize:  1000,
		batchSize:   100,
		flushPeriod: 1 * time.Second,
		workerCount: 4,
	}

	for _, opt := range opts {
		opt(config)
	}

	ctx, cancel := context.WithCancel(context.Background())

	bus := &PersistentEventBus{
		memory:      NewInMemoryEventBus(WithBufferSize(config.bufferSize), WithWorkerCount(config.workerCount)),
		store:       store,
		buffer:      make(chan unit.Event, config.bufferSize),
		batchSize:   config.batchSize,
		flushPeriod: config.flushPeriod,
		ctx:         ctx,
		cancel:      cancel,
	}

	bus.wg.Add(1)
	go bus.persistenceWorker()

	return bus
}

type persistentConfig struct {
	bufferSize  int
	batchSize   int
	flushPeriod time.Duration
	workerCount int
}

type PersistentOption func(*persistentConfig)

func WithPersistentBufferSize(size int) PersistentOption {
	return func(c *persistentConfig) {
		if size > 0 {
			c.bufferSize = size
		}
	}
}

func WithBatchSize(size int) PersistentOption {
	return func(c *persistentConfig) {
		if size > 0 {
			c.batchSize = size
		}
	}

}

func WithFlushPeriod(period time.Duration) PersistentOption {
	return func(c *persistentConfig) {
		if period > 0 {
			c.flushPeriod = period
		}
	}
}

func WithPersistentWorkerCount(count int) PersistentOption {
	return func(c *persistentConfig) {
		if count > 0 {
			c.workerCount = count
		}
	}
}

func (b *PersistentEventBus) Publish(event unit.Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	b.mu.RLock()
	closed := b.closed
	b.mu.RUnlock()

	if closed {
		return fmt.Errorf("eventbus is closed")
	}

	if err := b.memory.Publish(event); err != nil {
		return err
	}

	select {
	case b.buffer <- event:
		return nil
	case <-b.ctx.Done():
		return fmt.Errorf("eventbus is closed")
	}
}

func (b *PersistentEventBus) Subscribe(handler EventHandler, filters ...EventFilter) (SubscriptionID, error) {
	return b.memory.Subscribe(handler, filters...)
}

func (b *PersistentEventBus) Unsubscribe(id SubscriptionID) error {
	return b.memory.Unsubscribe(id)
}

func (b *PersistentEventBus) Query(ctx context.Context, filter EventQueryFilter) ([]unit.Event, error) {
	return b.store.Query(ctx, filter)
}

func (b *PersistentEventBus) Replay(ctx context.Context, correlationID string, handler EventHandler) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	events, err := b.store.Query(ctx, EventQueryFilter{
		CorrelationID: correlationID,
	})
	if err != nil {
		return fmt.Errorf("query events: %w", err)
	}

	for i := len(events) - 1; i >= 0; i-- {
		if err := handler(events[i]); err != nil {
			return fmt.Errorf("handle event: %w", err)
		}
	}

	return nil
}

func (b *PersistentEventBus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.mu.Unlock()

	b.cancel()

	close(b.buffer)

	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}

	return b.memory.Close()
}

func (b *PersistentEventBus) persistenceWorker() {
	defer b.wg.Done()

	batch := make([]unit.Event, 0, b.batchSize)
	ticker := time.NewTicker(b.flushPeriod)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}

		if store, ok := b.store.(*SQLiteEventStore); ok {
			_ = store.SaveBatch(context.Background(), batch)
		} else {
			for _, event := range batch {
				_ = b.store.Save(context.Background(), event)
			}
		}
		batch = batch[:0]
	}

	for {
		select {
		case event, ok := <-b.buffer:
			if !ok {
				flush()
				return
			}
			batch = append(batch, event)
			if len(batch) >= b.batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}
