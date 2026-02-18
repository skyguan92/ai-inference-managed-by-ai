package benchmark

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/registry"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
)

// setupRegistry creates a registry with all domains registered
func setupRegistry() *unit.Registry {
	r := unit.NewRegistry()
	_ = registry.RegisterAll(r)
	return r
}

// setupGateway creates a gateway with all domains registered
func setupGateway() *gateway.Gateway {
	r := setupRegistry()
	return gateway.NewGateway(r)
}

// BenchmarkRegistry_GetCommand tests registry command lookup performance
func BenchmarkRegistry_GetCommand(b *testing.B) {
	r := setupRegistry()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.GetCommand("model.list")
		}
	})
}

// BenchmarkRegistry_GetQuery tests registry query lookup performance
func BenchmarkRegistry_GetQuery(b *testing.B) {
	r := setupRegistry()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.GetQuery("model.list")
		}
	})
}

// BenchmarkRegistry_GetResource tests registry resource lookup performance
func BenchmarkRegistry_GetResource(b *testing.B) {
	r := setupRegistry()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.GetResource("model")
		}
	})
}

// BenchmarkRegistry_Get tests generic registry lookup performance
func BenchmarkRegistry_Get(b *testing.B) {
	r := setupRegistry()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.Get("model.list")
		}
	})
}

// BenchmarkRegistry_ConcurrentReads tests concurrent read performance
func BenchmarkRegistry_ConcurrentReads(b *testing.B) {
	r := setupRegistry()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.GetCommand("model.list")
			r.GetQuery("model.list")
			r.Get("engine.list")
		}
	})
}

// BenchmarkGateway_ExecuteQuery tests gateway query execution performance
func BenchmarkGateway_ExecuteQuery(b *testing.B) {
	gw := setupGateway()
	ctx := context.Background()
	req := &gateway.Request{
		Type:  gateway.TypeQuery,
		Unit:  "model.list",
		Input: map[string]any{},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gw.Handle(ctx, req)
	}
}

// BenchmarkGateway_ExecuteQuery_Parallel tests parallel gateway query execution
func BenchmarkGateway_ExecuteQuery_Parallel(b *testing.B) {
	gw := setupGateway()
	ctx := context.Background()
	req := &gateway.Request{
		Type:  gateway.TypeQuery,
		Unit:  "model.list",
		Input: map[string]any{},
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			gw.Handle(ctx, req)
		}
	})
}

// BenchmarkGateway_ExecuteCommand tests gateway command execution performance
func BenchmarkGateway_ExecuteCommand(b *testing.B) {
	gw := setupGateway()
	ctx := context.Background()
	req := &gateway.Request{
		Type: gateway.TypeCommand,
		Unit: "model.create",
		Input: map[string]any{
			"name":   "test-model",
			"source": "ollama",
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gw.Handle(ctx, req)
	}
}

// BenchmarkEventBus_Publish tests event publishing performance
type mockEventPublisher struct {
	events []any
	mu     sync.RWMutex
}

func (m *mockEventPublisher) Publish(event any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

func BenchmarkEventBus_Publish(b *testing.B) {
	publisher := &mockEventPublisher{}
	event := &unit.ExecutionEvent{
		EventType:     string(unit.ExecutionStarted),
		Domain:        "model",
		UnitName:      "model.list",
		Timestamp:     time.Now(),
		CorrelationID: "test-correlation-id",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = publisher.Publish(event)
	}
}

// BenchmarkEventBus_Publish_Parallel tests parallel event publishing
func BenchmarkEventBus_Publish_Parallel(b *testing.B) {
	publisher := &mockEventPublisher{}
	event := &unit.ExecutionEvent{
		EventType:     string(unit.ExecutionStarted),
		Domain:        "model",
		UnitName:      "model.list",
		Timestamp:     time.Now(),
		CorrelationID: "test-correlation-id",
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = publisher.Publish(event)
		}
	})
}

// BenchmarkSchema_Validate_String tests string schema validation performance
func BenchmarkSchema_Validate_String(b *testing.B) {
	schema := unit.StringSchema()
	input := "test-string-value"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = schema.Validate(input)
	}
}

// BenchmarkSchema_Validate_Number tests number schema validation performance
func BenchmarkSchema_Validate_Number(b *testing.B) {
	schema := unit.NumberSchema()
	input := 42.0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = schema.Validate(input)
	}
}

// BenchmarkSchema_Validate_Object tests object schema validation performance
func BenchmarkSchema_Validate_Object(b *testing.B) {
	schema := unit.ObjectSchema(map[string]unit.Field{
		"name":   unit.NewField("name", unit.StringSchema()),
		"source": unit.NewField("source", unit.StringSchema()),
	}, []string{"name"})
	input := map[string]any{
		"name":   "test-model",
		"source": "ollama",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = schema.Validate(input)
	}
}

// BenchmarkSchema_Validate_Array tests array schema validation performance
func BenchmarkSchema_Validate_Array(b *testing.B) {
	schema := unit.ArraySchema(unit.StringSchema())
	input := []any{"item1", "item2", "item3", "item4", "item5"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = schema.Validate(input)
	}
}

// BenchmarkCommand_Execute_MemoryStore tests command execution with memory store
func BenchmarkCommand_Execute_MemoryStore(b *testing.B) {
	store := model.NewMemoryStore()
	cmd := model.NewCreateCommand(store)
	ctx := context.Background()
	input := map[string]any{
		"name":   "test-model",
		"source": "ollama",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cmd.Execute(ctx, input)
	}
}

// BenchmarkCommand_Execute_WithEvents tests command execution with event publishing
func BenchmarkCommand_Execute_WithEvents(b *testing.B) {
	store := model.NewMemoryStore()
	publisher := &mockEventPublisher{}
	cmd := model.NewCreateCommandWithEvents(store, publisher)
	ctx := context.Background()
	input := map[string]any{
		"name":   "test-model",
		"source": "ollama",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cmd.Execute(ctx, input)
	}
}

// BenchmarkMemoryAllocation_ObjectCreation benchmarks object creation allocation
func BenchmarkMemoryAllocation_ObjectCreation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = &gateway.Request{
			Type:  gateway.TypeQuery,
			Unit:  "model.list",
			Input: map[string]any{},
		}
	}
}

// BenchmarkMemoryAllocation_MapCreation benchmarks map creation allocation
func BenchmarkMemoryAllocation_MapCreation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m := make(map[string]any)
		m["key1"] = "value1"
		m["key2"] = 42
		m["key3"] = true
	}
}

// BenchmarkRegistry_GetCommand_WithFactory benchmarks resource factory lookup
func BenchmarkRegistry_GetResource_WithFactory(b *testing.B) {
	r := setupRegistry()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.GetResourceWithFactory("asms://model/test-model")
		}
	})
}

// BenchmarkContext_WithValues tests context value operations
func BenchmarkContext_WithValues(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx = unit.WithRequestID(ctx, fmt.Sprintf("req-%d", i))
		ctx = unit.WithTraceID(ctx, fmt.Sprintf("trace-%d", i))
	}
}

// BenchmarkContext_GetValues tests context value retrieval
func BenchmarkContext_GetValues(b *testing.B) {
	ctx := context.Background()
	ctx = unit.WithRequestID(ctx, "test-request-id")
	ctx = unit.WithTraceID(ctx, "test-trace-id")
	ctx = unit.WithStartTime(ctx, time.Now())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = unit.GetRequestID(ctx)
		_ = unit.GetTraceID(ctx)
		_ = unit.GetStartTime(ctx)
	}
}

// BenchmarkGateway_ValidateRequest tests request validation performance
func BenchmarkGateway_ValidateRequest(b *testing.B) {
	gw := setupGateway()
	req := &gateway.Request{
		Type:  gateway.TypeQuery,
		Unit:  "model.list",
		Input: map[string]any{},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gw.Handle(context.Background(), req)
	}
}

// BenchmarkRegistry_ListCommands tests listing all commands
func BenchmarkRegistry_ListCommands(b *testing.B) {
	r := setupRegistry()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.ListCommands()
	}
}

// BenchmarkRegistry_ListQueries tests listing all queries
func BenchmarkRegistry_ListQueries(b *testing.B) {
	r := setupRegistry()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.ListQueries()
	}
}
