// Package benchmark contains performance benchmarks and optimizations for AIMA.
package benchmark

import (
	"sync"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

// OptimizedRegistry wraps the standard Registry with caching for frequently accessed units.
// This provides significantly faster lookups at the cost of slightly more memory usage.
type OptimizedRegistry struct {
	registry *unit.Registry

	// LRU-style cache for hot lookups
	cmdCache      map[string]unit.Command
	queryCache    map[string]unit.Query
	resourceCache map[string]unit.Resource

	// Fast path for single writer, multiple readers
	cacheMu sync.RWMutex
}

// NewOptimizedRegistry creates a new optimized registry wrapper.
func NewOptimizedRegistry(r *unit.Registry) *OptimizedRegistry {
	return &OptimizedRegistry{
		registry:      r,
		cmdCache:      make(map[string]unit.Command),
		queryCache:    make(map[string]unit.Query),
		resourceCache: make(map[string]unit.Resource),
	}
}

// GetCommand retrieves a Command from cache or underlying registry.
// Uses double-checked locking pattern for thread safety.
func (or *OptimizedRegistry) GetCommand(name string) unit.Command {
	// Fast path: read from cache
	or.cacheMu.RLock()
	if cmd, ok := or.cmdCache[name]; ok {
		or.cacheMu.RUnlock()
		return cmd
	}
	or.cacheMu.RUnlock()

	// Slow path: fetch from registry and cache
	cmd := or.registry.GetCommand(name)
	if cmd != nil {
		or.cacheMu.Lock()
		or.cmdCache[name] = cmd
		or.cacheMu.Unlock()
	}
	return cmd
}

// GetQuery retrieves a Query from cache or underlying registry.
func (or *OptimizedRegistry) GetQuery(name string) unit.Query {
	or.cacheMu.RLock()
	if q, ok := or.queryCache[name]; ok {
		or.cacheMu.RUnlock()
		return q
	}
	or.cacheMu.RUnlock()

	q := or.registry.GetQuery(name)
	if q != nil {
		or.cacheMu.Lock()
		or.queryCache[name] = q
		or.cacheMu.Unlock()
	}
	return q
}

// GetResource retrieves a Resource from cache or underlying registry.
func (or *OptimizedRegistry) GetResource(uri string) unit.Resource {
	or.cacheMu.RLock()
	if res, ok := or.resourceCache[uri]; ok {
		or.cacheMu.RUnlock()
		return res
	}
	or.cacheMu.RUnlock()

	res := or.registry.GetResource(uri)
	if res != nil {
		or.cacheMu.Lock()
		or.resourceCache[uri] = res
		or.cacheMu.Unlock()
	}
	return res
}

// ClearCache clears all caches.
func (or *OptimizedRegistry) ClearCache() {
	or.cacheMu.Lock()
	or.cmdCache = make(map[string]unit.Command)
	or.queryCache = make(map[string]unit.Query)
	or.resourceCache = make(map[string]unit.Resource)
	or.cacheMu.Unlock()
}

// WarmCache pre-populates the cache with frequently accessed units.
func (or *OptimizedRegistry) WarmCache(units []string) {
	for _, name := range units {
		if cmd := or.registry.GetCommand(name); cmd != nil {
			or.cacheMu.Lock()
			or.cmdCache[name] = cmd
			or.cacheMu.Unlock()
		}
		if query := or.registry.GetQuery(name); query != nil {
			or.cacheMu.Lock()
			or.queryCache[name] = query
			or.cacheMu.Unlock()
		}
	}
}
