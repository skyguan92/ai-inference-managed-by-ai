package benchmark

import (
	"sync"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/registry"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

// BenchmarkOptimizedRegistry_GetCommand tests optimized registry command lookup
func BenchmarkOptimizedRegistry_GetCommand(b *testing.B) {
	r := unit.NewRegistry()
	_ = registry.RegisterAll(r)
	or := NewOptimizedRegistry(r)

	// Warm up cache
	or.GetCommand("model.list")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			or.GetCommand("model.list")
		}
	})
}

// BenchmarkOptimizedRegistry_GetCommand_ColdCache tests cold cache performance
func BenchmarkOptimizedRegistry_GetCommand_ColdCache(b *testing.B) {
	r := unit.NewRegistry()
	_ = registry.RegisterAll(r)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// Create new optimized registry for each iteration to simulate cold cache
		or := NewOptimizedRegistry(r)
		for pb.Next() {
			or.GetCommand("model.list")
		}
	})
}

// BenchmarkOptimizedRegistry_GetQuery tests optimized registry query lookup
func BenchmarkOptimizedRegistry_GetQuery(b *testing.B) {
	r := unit.NewRegistry()
	_ = registry.RegisterAll(r)
	or := NewOptimizedRegistry(r)
	or.GetQuery("model.list")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			or.GetQuery("model.list")
		}
	})
}

// BenchmarkOptimizedRegistry_GetResource tests optimized registry resource lookup
func BenchmarkOptimizedRegistry_GetResource(b *testing.B) {
	r := unit.NewRegistry()
	_ = registry.RegisterAll(r)
	or := NewOptimizedRegistry(r)
	or.GetResource("model")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			or.GetResource("model")
		}
	})
}

// BenchmarkRequestPool tests RequestPool performance
func BenchmarkRequestPool(b *testing.B) {
	pool := NewRequestPool()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := pool.Get()
			req.Type = "query"
			req.Unit = "model.list"
			req.Input["key"] = "value"
			pool.Put(req)
		}
	})
}

// BenchmarkRequestPool_NoPool compares against no pooling
func BenchmarkRequestPool_NoPool(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := &Request{
				Type:  "query",
				Unit:  "model.list",
				Input: map[string]any{"key": "value"},
			}
			_ = req
		}
	})
}

// BenchmarkMapPool tests MapPool performance
func BenchmarkMapPool(b *testing.B) {
	pool := NewMapPool()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m := pool.Get()
			m["key1"] = "value1"
			m["key2"] = 42
			pool.Put(m)
		}
	})
}

// BenchmarkMapPool_NoPool compares against no pooling
func BenchmarkMapPool_NoPool(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m := make(map[string]any)
			m["key1"] = "value1"
			m["key2"] = 42
			_ = m
		}
	})
}

// BenchmarkEventPool tests EventPool performance
func BenchmarkEventPool(b *testing.B) {
	pool := NewEventPool()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			event := pool.Get()
			event.EventType = "test"
			event.Domain = "model"
			pool.Put(event)
		}
	})
}

// BenchmarkEventPool_NoPool compares against no pooling
func BenchmarkEventPool_NoPool(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			event := &unit.ExecutionEvent{
				EventType: "test",
				Domain:    "model",
			}
			_ = event
		}
	})
}

// BenchmarkSchemaCache tests SchemaCache performance
func BenchmarkSchemaCache(b *testing.B) {
	cache := NewSchemaCache()
	schema := unit.StringSchema()
	cache.Put("string", schema)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = cache.Get("string")
		}
	})
}

// BenchmarkSchemaCache_NoCache compares against no caching
func BenchmarkSchemaCache_NoCache(b *testing.B) {
	schemas := make(map[string]*unit.Schema)
	schemas["string"] = unit.StringSchema()
	mu := sync.RWMutex{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.RLock()
			_ = schemas["string"]
			mu.RUnlock()
		}
	})
}

// BenchmarkOptimizedRegistry_WarmCache tests cache warming performance
func BenchmarkOptimizedRegistry_WarmCache(b *testing.B) {
	r := unit.NewRegistry()
	_ = registry.RegisterAll(r)
	or := NewOptimizedRegistry(r)

	// Warm cache with common units
	or.WarmCache([]string{
		"model.list", "model.get", "model.create",
		"engine.list", "engine.get", "engine.start",
		"device.info", "device.metrics",
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			or.GetCommand("model.list")
			or.GetQuery("engine.list")
		}
	})
}

// BenchmarkMemoryAllocation_WithPool shows allocation reduction with pooling
func BenchmarkMemoryAllocation_WithPool(b *testing.B) {
	pool := NewRequestPool()
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := pool.Get()
		req.Type = "query"
		req.Unit = "model.list"
		pool.Put(req)
	}
}

// BenchmarkMemoryAllocation_WithoutPool shows allocation without pooling
func BenchmarkMemoryAllocation_WithoutPool(b *testing.B) {
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &Request{
			Type:  "query",
			Unit:  "model.list",
			Input: make(map[string]any),
		}
		_ = req
	}
}

// BenchmarkOptimizedRegistry_ConcurrentMixed tests concurrent mixed operations
func BenchmarkOptimizedRegistry_ConcurrentMixed(b *testing.B) {
	r := unit.NewRegistry()
	_ = registry.RegisterAll(r)
	or := NewOptimizedRegistry(r)

	// Pre-warm some entries
	or.WarmCache([]string{"model.list", "engine.list"})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 6 {
			case 0:
				or.GetCommand("model.list")
			case 1:
				or.GetQuery("model.list")
			case 2:
				or.GetCommand("engine.list")
			case 3:
				or.GetQuery("engine.list")
			case 4:
				or.GetResource("model")
			case 5:
				or.GetResource("engine")
			}
			i++
		}
	})
}

// BenchmarkComparison_StandardVsOptimized compares standard vs optimized registry
func BenchmarkComparison_StandardVsOptimized(b *testing.B) {
	b.Run("Standard", func(b *testing.B) {
		r := unit.NewRegistry()
		_ = registry.RegisterAll(r)
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				r.GetCommand("model.list")
			}
		})
	})

	b.Run("Optimized", func(b *testing.B) {
		r := unit.NewRegistry()
		_ = registry.RegisterAll(r)
		or := NewOptimizedRegistry(r)
		or.GetCommand("model.list") // Warm cache
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				or.GetCommand("model.list")
			}
		})
	})
}
