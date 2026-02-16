package eventbus

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type SubscriptionID string

type EventHandler func(event unit.Event) error

type EventFilter func(event unit.Event) bool

type EventBus interface {
	Publish(event unit.Event) error
	Subscribe(handler EventHandler, filters ...EventFilter) (SubscriptionID, error)
	Unsubscribe(id SubscriptionID) error
	Close() error
}

type InMemoryEventBus struct {
	mu          sync.RWMutex
	subscribers map[SubscriptionID]*subscription
	eventChan   chan unit.Event
	workerCount int
	bufferSize  int
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	closed      bool
}

type subscription struct {
	id      SubscriptionID
	handler EventHandler
	filters []EventFilter
}

func NewInMemoryEventBus(opts ...Option) *InMemoryEventBus {
	config := &config{
		bufferSize:  1000,
		workerCount: 4,
	}

	for _, opt := range opts {
		opt(config)
	}

	ctx, cancel := context.WithCancel(context.Background())

	bus := &InMemoryEventBus{
		subscribers: make(map[SubscriptionID]*subscription),
		eventChan:   make(chan unit.Event, config.bufferSize),
		workerCount: config.workerCount,
		bufferSize:  config.bufferSize,
		ctx:         ctx,
		cancel:      cancel,
	}

	for i := 0; i < bus.workerCount; i++ {
		bus.wg.Add(1)
		go bus.worker()
	}

	return bus
}

type config struct {
	bufferSize  int
	workerCount int
}

type Option func(*config)

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func WithBufferSize(size int) Option {
	return func(c *config) {
		if size > 0 {
			c.bufferSize = size
		}
	}
}

func WithWorkerCount(count int) Option {
	return func(c *config) {
		if count > 0 {
			c.workerCount = count
		}
	}
}

func (b *InMemoryEventBus) Publish(event unit.Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	b.mu.RLock()
	closed := b.closed
	b.mu.RUnlock()

	if closed {
		return fmt.Errorf("eventbus is closed")
	}

	select {
	case b.eventChan <- event:
		return nil
	case <-b.ctx.Done():
		return fmt.Errorf("eventbus is closed")
	}
}

func (b *InMemoryEventBus) Subscribe(handler EventHandler, filters ...EventFilter) (SubscriptionID, error) {
	if handler == nil {
		return "", fmt.Errorf("handler cannot be nil")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return "", fmt.Errorf("eventbus is closed")
	}

	id := SubscriptionID(generateID())
	b.subscribers[id] = &subscription{
		id:      id,
		handler: handler,
		filters: filters,
	}

	return id, nil
}

func (b *InMemoryEventBus) Unsubscribe(id SubscriptionID) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.subscribers[id]; !exists {
		return fmt.Errorf("subscription %s not found", id)
	}

	delete(b.subscribers, id)
	return nil
}

func (b *InMemoryEventBus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.mu.Unlock()

	b.cancel()

	close(b.eventChan)

	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	<-done

	b.mu.Lock()
	b.subscribers = make(map[SubscriptionID]*subscription)
	b.mu.Unlock()

	return nil
}

func (b *InMemoryEventBus) worker() {
	defer b.wg.Done()

	for {
		select {
		case event, ok := <-b.eventChan:
			if !ok {
				return
			}
			b.dispatchEvent(event)
		case <-b.ctx.Done():
			for {
				select {
				case event, ok := <-b.eventChan:
					if !ok {
						return
					}
					b.dispatchEvent(event)
				default:
					return
				}
			}
		}
	}
}

func (b *InMemoryEventBus) dispatchEvent(event unit.Event) {
	if event == nil {
		return
	}
	b.mu.RLock()
	subs := make([]*subscription, 0, len(b.subscribers))
	for _, sub := range b.subscribers {
		subs = append(subs, sub)
	}
	b.mu.RUnlock()

	for _, sub := range subs {
		if !b.matchFilters(event, sub.filters) {
			continue
		}

		_ = sub.handler(event)
	}
}

func (b *InMemoryEventBus) matchFilters(event unit.Event, filters []EventFilter) bool {
	if len(filters) == 0 {
		return true
	}

	for _, filter := range filters {
		if !filter(event) {
			return false
		}
	}

	return true
}

func FilterByType(eventType string) EventFilter {
	return func(event unit.Event) bool {
		return event.Type() == eventType
	}
}

func FilterByDomain(domain string) EventFilter {
	return func(event unit.Event) bool {
		return event.Domain() == domain
	}
}

func FilterByTypes(types ...string) EventFilter {
	typeSet := make(map[string]bool)
	for _, t := range types {
		typeSet[t] = true
	}
	return func(event unit.Event) bool {
		return typeSet[event.Type()]
	}
}

func FilterByDomains(domains ...string) EventFilter {
	domainSet := make(map[string]bool)
	for _, d := range domains {
		domainSet[d] = true
	}
	return func(event unit.Event) bool {
		return domainSet[event.Domain()]
	}
}
