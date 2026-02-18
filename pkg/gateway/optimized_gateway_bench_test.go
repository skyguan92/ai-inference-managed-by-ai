package gateway

import (
	"context"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/registry"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func setupRegistry() *unit.Registry {
	r := unit.NewRegistry()
	_ = registry.RegisterAll(r)
	return r
}

// BenchmarkGateway_Standard tests standard gateway performance
func BenchmarkGateway_Standard(b *testing.B) {
	r := setupRegistry()
	gw := NewGateway(r)
	ctx := context.Background()
	req := &Request{
		Type:  TypeQuery,
		Unit:  "model.list",
		Input: map[string]any{},
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		resp := gw.Handle(ctx, req)
		_ = resp
	}
}

// BenchmarkGateway_Optimized tests optimized gateway performance
func BenchmarkGateway_Optimized(b *testing.B) {
	r := setupRegistry()
	gw := NewOptimizedGateway(r)
	ctx := context.Background()
	req := &Request{
		Type:  TypeQuery,
		Unit:  "model.list",
		Input: map[string]any{},
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		resp := gw.Handle(ctx, req)
		gw.ReleaseResponse(resp)
	}
}

// BenchmarkGateway_Standard_Parallel tests standard gateway with parallel load
func BenchmarkGateway_Standard_Parallel(b *testing.B) {
	r := setupRegistry()
	gw := NewGateway(r)
	ctx := context.Background()
	req := &Request{
		Type:  TypeQuery,
		Unit:  "model.list",
		Input: map[string]any{},
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp := gw.Handle(ctx, req)
			_ = resp
		}
	})
}

// BenchmarkGateway_Optimized_Parallel tests optimized gateway with parallel load
func BenchmarkGateway_Optimized_Parallel(b *testing.B) {
	r := setupRegistry()
	gw := NewOptimizedGateway(r)
	ctx := context.Background()
	req := &Request{
		Type:  TypeQuery,
		Unit:  "model.list",
		Input: map[string]any{},
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp := gw.Handle(ctx, req)
			gw.ReleaseResponse(resp)
		}
	})
}

// BenchmarkGateway_Standard_Command tests standard gateway command execution
func BenchmarkGateway_Standard_Command(b *testing.B) {
	r := setupRegistry()
	gw := NewGateway(r)
	ctx := context.Background()
	req := &Request{
		Type: TypeCommand,
		Unit: "model.create",
		Input: map[string]any{
			"name":   "test-model",
			"source": "ollama",
		},
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		resp := gw.Handle(ctx, req)
		_ = resp
	}
}

// BenchmarkGateway_Optimized_Command tests optimized gateway command execution
func BenchmarkGateway_Optimized_Command(b *testing.B) {
	r := setupRegistry()
	gw := NewOptimizedGateway(r)
	ctx := context.Background()
	req := &Request{
		Type: TypeCommand,
		Unit: "model.create",
		Input: map[string]any{
			"name":   "test-model",
			"source": "ollama",
		},
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		resp := gw.Handle(ctx, req)
		gw.ReleaseResponse(resp)
	}
}

// BenchmarkComparison_StandardVsOptimized compares both implementations
func BenchmarkComparison_Gateway(b *testing.B) {
	b.Run("Standard", func(b *testing.B) {
		r := setupRegistry()
		gw := NewGateway(r)
		ctx := context.Background()
		req := &Request{
			Type:  TypeQuery,
			Unit:  "model.list",
			Input: map[string]any{},
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = gw.Handle(ctx, req)
		}
	})
	
	b.Run("Optimized", func(b *testing.B) {
		r := setupRegistry()
		gw := NewOptimizedGateway(r)
		ctx := context.Background()
		req := &Request{
			Type:  TypeQuery,
			Unit:  "model.list",
			Input: map[string]any{},
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp := gw.Handle(ctx, req)
			gw.ReleaseResponse(resp)
		}
	})
}
