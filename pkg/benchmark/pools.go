package benchmark

import (
	"sync"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

// RequestPool is a sync.Pool for reusing Request objects.
// This significantly reduces GC pressure in high-throughput scenarios.
type RequestPool struct {
	pool sync.Pool
}

// NewRequestPool creates a new RequestPool.
func NewRequestPool() *RequestPool {
	return &RequestPool{
		pool: sync.Pool{
			New: func() any {
				return &Request{
					Input: make(map[string]any, 8),
				}
			},
		},
	}
}

// Request is a reusable request object.
type Request struct {
	Type    string
	Unit    string
	Input   map[string]any
	Options RequestOptions
}

// RequestOptions contains optional request settings.
type RequestOptions struct {
	Timeout int64
	Async   bool
	TraceID string
}

// Get retrieves a Request from the pool.
func (p *RequestPool) Get() *Request {
	r := p.pool.Get().(*Request)
	r.Input = make(map[string]any, 8)
	return r
}

// Put returns a Request to the pool.
func (p *RequestPool) Put(r *Request) {
	if r == nil {
		return
	}
	// Clear fields to avoid memory leaks
	r.Type = ""
	r.Unit = ""
	r.Options = RequestOptions{}
	// Reset map but keep capacity
	for k := range r.Input {
		delete(r.Input, k)
	}
	p.pool.Put(r)
}

// MapPool is a sync.Pool for reusing map[string]any objects.
type MapPool struct {
	pool sync.Pool
}

// NewMapPool creates a new MapPool with pre-allocated maps.
func NewMapPool() *MapPool {
	return &MapPool{
		pool: sync.Pool{
			New: func() any {
				return make(map[string]any, 8)
			},
		},
	}
}

// Get retrieves a map from the pool.
func (p *MapPool) Get() map[string]any {
	return p.pool.Get().(map[string]any)
}

// Put returns a map to the pool.
func (p *MapPool) Put(m map[string]any) {
	if m == nil {
		return
	}
	// Clear map but keep capacity
	for k := range m {
		delete(m, k)
	}
	p.pool.Put(m)
}

// SchemaCache caches validated schemas to avoid repeated validation.
type SchemaCache struct {
	cache map[string]*unit.Schema
	mu    sync.RWMutex
}

// NewSchemaCache creates a new SchemaCache.
func NewSchemaCache() *SchemaCache {
	return &SchemaCache{
		cache: make(map[string]*unit.Schema),
	}
}

// Get retrieves a schema from cache by key.
func (sc *SchemaCache) Get(key string) (*unit.Schema, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	s, ok := sc.cache[key]
	return s, ok
}

// Put stores a schema in cache.
func (sc *SchemaCache) Put(key string, schema *unit.Schema) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.cache[key] = schema
}

// EventPool is a sync.Pool for reusing ExecutionEvent objects.
type EventPool struct {
	pool sync.Pool
}

// NewEventPool creates a new EventPool.
func NewEventPool() *EventPool {
	return &EventPool{
		pool: sync.Pool{
			New: func() any {
				return &unit.ExecutionEvent{}
			},
		},
	}
}

// Get retrieves an ExecutionEvent from the pool.
func (p *EventPool) Get() *unit.ExecutionEvent {
	return p.pool.Get().(*unit.ExecutionEvent)
}

// Put returns an ExecutionEvent to the pool.
func (p *EventPool) Put(e *unit.ExecutionEvent) {
	if e == nil {
		return
	}
	// Clear fields
	e.EventType = ""
	e.Domain = ""
	e.UnitName = ""
	e.Input = nil
	e.Output = nil
	e.Error = ""
	e.DurationMs = 0
	p.pool.Put(e)
}
