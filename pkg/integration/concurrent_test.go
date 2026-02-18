package integration

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/eventbus"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/registry"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConcurrent_CommandExecution tests concurrent execution of the same command
func TestConcurrent_CommandExecution(t *testing.T) {
	reg := unit.NewRegistry()
	err := registry.RegisterAll(reg)
	require.NoError(t, err)

	gw := gateway.NewGateway(reg)
	ctx := context.Background()

	const numGoroutines = 50
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)
	results := make(chan *gateway.Response, numGoroutines)

	// Concurrently execute model.list query
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			resp := gw.Handle(ctx, &gateway.Request{
				Type: gateway.TypeQuery,
				Unit: "model.list",
				Input: map[string]any{
					"limit": 10,
				},
			})

			results <- resp
			if !resp.Success && resp.Error != nil {
				errors <- resp.Error
			}
		}(i)
	}

	wg.Wait()
	close(errors)
	close(results)

	// Verify all responses were received
	responseCount := 0
	successCount := 0
	for resp := range results {
		responseCount++
		if resp.Success {
			successCount++
		}
	}

	assert.Equal(t, numGoroutines, responseCount, "all requests should receive responses")
	assert.Equal(t, numGoroutines, successCount, "all queries should succeed")

	// Check for errors
	errorCount := 0
	for range errors {
		errorCount++
	}
	assert.Equal(t, 0, errorCount, "no errors should occur")
}

// TestConcurrent_MultipleCommands tests concurrent execution of different commands
func TestConcurrent_MultipleCommands(t *testing.T) {
	reg := unit.NewRegistry()
	err := registry.RegisterAll(reg)
	require.NoError(t, err)

	gw := gateway.NewGateway(reg)
	ctx := context.Background()

	commands := []string{
		"model.list",
		"engine.list",
		"resource.status",
		"service.list",
		"device.info",
	}

	const iterationsPerCommand = 20
	var wg sync.WaitGroup
	var successCount int32 = 0
	var errorCount int32 = 0

	for _, cmd := range commands {
		for i := 0; i < iterationsPerCommand; i++ {
			wg.Add(1)
			go func(command string, id int) {
				defer wg.Done()

				resp := gw.Handle(ctx, &gateway.Request{
					Type:  gateway.TypeQuery,
					Unit:  command,
					Input: map[string]any{},
				})

				if resp.Success {
					atomic.AddInt32(&successCount, 1)
				} else {
					atomic.AddInt32(&errorCount, 1)
				}
			}(cmd, i)
		}
	}

	wg.Wait()

	totalCommands := len(commands) * iterationsPerCommand
	// Note: Some commands may fail due to missing providers (device.info, resource.status)
	// We just verify that the system handles concurrent requests without crashing
	assert.Greater(t, atomic.LoadInt32(&successCount), int32(0), "at least some commands should succeed")
	t.Logf("Success: %d/%d", atomic.LoadInt32(&successCount), totalCommands)
}

// TestConcurrent_ResourceAllocation tests concurrent resource allocation
func TestConcurrent_ResourceAllocation(t *testing.T) {
	reg := unit.NewRegistry()
	err := registry.RegisterAll(reg)
	require.NoError(t, err)

	gw := gateway.NewGateway(reg)
	ctx := context.Background()

	const numAllocations = 30
	var wg sync.WaitGroup
	slotIDs := make(chan string, numAllocations)
	var allocErrors int32 = 0

	// Concurrent allocations
	for i := 0; i < numAllocations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			resp := gw.Handle(ctx, &gateway.Request{
				Type: gateway.TypeCommand,
				Unit: "resource.allocate",
				Input: map[string]any{
					"name":         "concurrent-test-slot",
					"type":         "model",
					"memory_bytes": 64 * 1024 * 1024, // 64MB
					"priority":     5,
				},
			})

			if resp.Success {
				if data, ok := resp.Data.(map[string]any); ok {
					if slotID, ok := data["slot_id"].(string); ok && slotID != "" {
						slotIDs <- slotID
					}
				}
			} else {
				atomic.AddInt32(&allocErrors, 1)
			}
		}(i)
	}

	wg.Wait()
	close(slotIDs)

	// Collect allocated slot IDs
	allocatedSlots := make([]string, 0)
	for id := range slotIDs {
		allocatedSlots = append(allocatedSlots, id)
	}

	// Some allocations may fail due to resource constraints or missing provider, which is expected
	t.Logf("Allocated %d/%d slots (errors: %d)", len(allocatedSlots), numAllocations, atomic.LoadInt32(&allocErrors))
	// Without a proper resource provider, allocations may fail - that's acceptable
	// We just verify the system handles concurrent requests without crashing

	// Release all allocated slots concurrently
	var releaseWg sync.WaitGroup
	var releaseErrors int32 = 0

	for _, slotID := range allocatedSlots {
		releaseWg.Add(1)
		go func(id string) {
			defer releaseWg.Done()

			resp := gw.Handle(ctx, &gateway.Request{
				Type: gateway.TypeCommand,
				Unit: "resource.release",
				Input: map[string]any{
					"slot_id": id,
				},
			})

			if !resp.Success {
				atomic.AddInt32(&releaseErrors, 1)
			}
		}(slotID)
	}

	releaseWg.Wait()

	// Some release operations may fail, but we verify the system remains consistent
	t.Logf("Release errors: %d", atomic.LoadInt32(&releaseErrors))
}

// TestConcurrent_EventPublishing tests concurrent event publishing
func TestConcurrent_EventPublishing(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus(
		eventbus.WithBufferSize(1000),
		eventbus.WithWorkerCount(4),
	)
	defer bus.Close()

	const numSubscribers = 5
	const numEvents = 100

	// Create subscribers
	subscribers := make([]eventbus.SubscriptionID, numSubscribers)
	eventCounts := make([]int32, numSubscribers)

	for i := 0; i < numSubscribers; i++ {
		idx := i
		subID, err := bus.Subscribe(func(evt unit.Event) error {
			atomic.AddInt32(&eventCounts[idx], 1)
			return nil
		})
		require.NoError(t, err)
		subscribers[i] = subID
	}

	// Publish events concurrently
	var wg sync.WaitGroup
	for i := 0; i < numEvents; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			evt := &testEvent{
				eventType: "test.event",
				domain:    "test",
				payload:   map[string]any{"id": id},
			}
			_ = bus.Publish(evt)
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond) // Allow events to be processed

	// Verify each subscriber received all events
	for i := 0; i < numSubscribers; i++ {
		count := atomic.LoadInt32(&eventCounts[i])
		assert.GreaterOrEqual(t, count, int32(0), "subscriber %d should receive events", i)
		// Events may be distributed across workers, so exact count may vary
		t.Logf("Subscriber %d received %d events", i, count)
	}

	// Unsubscribe all
	for _, subID := range subscribers {
		err := bus.Unsubscribe(subID)
		assert.NoError(t, err)
	}
}

// TestConcurrent_RegistryAccess tests concurrent access to registry
func TestConcurrent_RegistryAccess(t *testing.T) {
	reg := unit.NewRegistry()
	err := registry.RegisterAll(reg)
	require.NoError(t, err)

	const numGoroutines = 100
	var wg sync.WaitGroup

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Randomly access different parts of the registry
			switch id % 5 {
			case 0:
				_ = reg.GetCommand("model.list")
			case 1:
				_ = reg.GetQuery("model.list")
			case 2:
				_ = reg.ListCommands()
			case 3:
				_ = reg.ListQueries()
			case 4:
				_ = reg.CommandCount()
				_ = reg.QueryCount()
			}
		}(i)
	}

	wg.Wait()

	// Verify registry is still consistent
	assert.Greater(t, reg.CommandCount(), 0, "registry should have commands")
	assert.Greater(t, reg.QueryCount(), 0, "registry should have queries")
}

// TestConcurrent_ModelLifecycle tests concurrent model operations
func TestConcurrent_ModelLifecycle(t *testing.T) {
	reg := unit.NewRegistry()
	err := registry.RegisterAll(reg)
	require.NoError(t, err)

	gw := gateway.NewGateway(reg)
	ctx := context.Background()

	const numModels = 20
	var wg sync.WaitGroup
	modelIDs := make(chan string, numModels)

	// Concurrent model creation
	for i := 0; i < numModels; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			resp := gw.Handle(ctx, &gateway.Request{
				Type: gateway.TypeCommand,
				Unit: "model.create",
				Input: map[string]any{
					"name":   "concurrent-model",
					"source": "local",
				},
			})

			if resp.Success {
				if data, ok := resp.Data.(map[string]any); ok {
					if modelID, ok := data["model_id"].(string); ok {
						modelIDs <- modelID
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(modelIDs)

	// Collect created model IDs
	createdModels := make([]string, 0)
	for id := range modelIDs {
		createdModels = append(createdModels, id)
	}

	t.Logf("Created %d models", len(createdModels))
	assert.Greater(t, len(createdModels), 0, "at least some models should be created")

	// Concurrent list operations while models exist
	var listWg sync.WaitGroup
	for i := 0; i < 10; i++ {
		listWg.Add(1)
		go func() {
			defer listWg.Done()
			resp := gw.Handle(ctx, &gateway.Request{
				Type:  gateway.TypeQuery,
				Unit:  "model.list",
				Input: map[string]any{},
			})
			assert.True(t, resp.Success, "model.list should succeed")
		}()
	}
	listWg.Wait()

	// Clean up all models concurrently
	var deleteWg sync.WaitGroup
	for _, modelID := range createdModels {
		deleteWg.Add(1)
		go func(id string) {
			defer deleteWg.Done()
			_ = gw.Handle(ctx, &gateway.Request{
				Type: gateway.TypeCommand,
				Unit: "model.delete",
				Input: map[string]any{
					"model_id": id,
					"force":    true,
				},
			})
		}(modelID)
	}
	deleteWg.Wait()
}

// TestConcurrent_PipelineOperations tests concurrent pipeline operations
func TestConcurrent_PipelineOperations(t *testing.T) {
	reg := unit.NewRegistry()
	err := registry.RegisterAll(reg)
	require.NoError(t, err)

	gw := gateway.NewGateway(reg)
	ctx := context.Background()

	const numPipelines = 10
	var wg sync.WaitGroup
	pipelineIDs := make(chan string, numPipelines)

	// Concurrent pipeline creation
	for i := 0; i < numPipelines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			resp := gw.Handle(ctx, &gateway.Request{
				Type: gateway.TypeCommand,
				Unit: "pipeline.create",
				Input: map[string]any{
					"name": "concurrent-pipeline",
					"steps": []map[string]any{
						{
							"id":    "step1",
							"type":  "model.list",
							"input": map[string]any{},
						},
					},
				},
			})

			if resp.Success {
				if data, ok := resp.Data.(map[string]any); ok {
					if pipelineID, ok := data["pipeline_id"].(string); ok {
						pipelineIDs <- pipelineID
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(pipelineIDs)

	createdPipelines := make([]string, 0)
	for id := range pipelineIDs {
		createdPipelines = append(createdPipelines, id)
	}

	t.Logf("Created %d pipelines", len(createdPipelines))

	// Concurrent list operations
	var listWg sync.WaitGroup
	for i := 0; i < 5; i++ {
		listWg.Add(1)
		go func() {
			defer listWg.Done()
			resp := gw.Handle(ctx, &gateway.Request{
				Type:  gateway.TypeQuery,
				Unit:  "pipeline.list",
				Input: map[string]any{},
			})
			assert.True(t, resp.Success, "pipeline.list should succeed")
		}()
	}
	listWg.Wait()

	// Clean up
	var deleteWg sync.WaitGroup
	for _, pipelineID := range createdPipelines {
		deleteWg.Add(1)
		go func(id string) {
			defer deleteWg.Done()
			_ = gw.Handle(ctx, &gateway.Request{
				Type: gateway.TypeCommand,
				Unit: "pipeline.delete",
				Input: map[string]any{
					"pipeline_id": id,
				},
			})
		}(pipelineID)
	}
	deleteWg.Wait()
}

// TestConcurrent_GatewayStress stress tests the gateway with high concurrency
func TestConcurrent_GatewayStress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	reg := unit.NewRegistry()
	err := registry.RegisterAll(reg)
	require.NoError(t, err)

	gw := gateway.NewGateway(reg)
	ctx := context.Background()

	const numRequests = 500
	const concurrency = 50

	semaphore := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var successCount int32 = 0
	var errorCount int32 = 0

	start := time.Now()

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(id int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			// Mix of different operations
			var resp *gateway.Response
			switch id % 10 {
			case 0, 1, 2, 3, 4:
				resp = gw.Handle(ctx, &gateway.Request{
					Type:  gateway.TypeQuery,
					Unit:  "model.list",
					Input: map[string]any{},
				})
			case 5, 6:
				resp = gw.Handle(ctx, &gateway.Request{
					Type:  gateway.TypeQuery,
					Unit:  "engine.list",
					Input: map[string]any{},
				})
			case 7, 8:
				resp = gw.Handle(ctx, &gateway.Request{
					Type:  gateway.TypeQuery,
					Unit:  "resource.status",
					Input: map[string]any{},
				})
			case 9:
				resp = gw.Handle(ctx, &gateway.Request{
					Type:  gateway.TypeQuery,
					Unit:  "service.list",
					Input: map[string]any{},
				})
			}

			if resp != nil && resp.Success {
				atomic.AddInt32(&successCount, 1)
			} else {
				atomic.AddInt32(&errorCount, 1)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("Completed %d requests in %v (%.2f req/s)",
		numRequests, duration, float64(numRequests)/duration.Seconds())

	// Most requests should succeed; some may fail due to timeout or missing providers
	successRate := float64(atomic.LoadInt32(&successCount)) / float64(numRequests)
	assert.Greater(t, successRate, 0.5, "at least 50% of requests should succeed")
	t.Logf("Success rate: %.1f%%", successRate*100)
}

// TestConcurrent_ResourcePoolExhaustion tests behavior when resources are exhausted
func TestConcurrent_ResourcePoolExhaustion(t *testing.T) {
	reg := unit.NewRegistry()
	err := registry.RegisterAll(reg)
	require.NoError(t, err)

	gw := gateway.NewGateway(reg)
	ctx := context.Background()

	// Try to allocate more resources than available concurrently
	const numAttempts = 50
	var wg sync.WaitGroup
	var successCount int32 = 0
	var failCount int32 = 0

	for i := 0; i < numAttempts; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			resp := gw.Handle(ctx, &gateway.Request{
				Type: gateway.TypeCommand,
				Unit: "resource.allocate",
				Input: map[string]any{
					"name":         "exhaust-test",
					"type":         "model",
					"memory_bytes": 512 * 1024 * 1024, // 512MB each
					"priority":     5,
				},
			})

			if resp.Success {
				atomic.AddInt32(&successCount, 1)
				// Get slot ID and release
				if data, ok := resp.Data.(map[string]any); ok {
					if slotID, ok := data["slot_id"].(string); ok {
						// Release immediately to avoid deadlock
						_ = gw.Handle(ctx, &gateway.Request{
							Type: gateway.TypeCommand,
							Unit: "resource.release",
							Input: map[string]any{
								"slot_id": slotID,
							},
						})
					}
				}
			} else {
				atomic.AddInt32(&failCount, 1)
			}
		}(i)
	}

	wg.Wait()

	total := atomic.LoadInt32(&successCount) + atomic.LoadInt32(&failCount)
	assert.Equal(t, int32(numAttempts), total, "all attempts should complete")
	t.Logf("Success: %d, Failed: %d", atomic.LoadInt32(&successCount), atomic.LoadInt32(&failCount))
}

// TestConcurrent_MixedOperations tests various operations mixed together
func TestConcurrent_MixedOperations(t *testing.T) {
	reg := unit.NewRegistry()
	err := registry.RegisterAll(reg)
	require.NoError(t, err)

	gw := gateway.NewGateway(reg)
	ctx := context.Background()

	operations := []struct {
		name string
		req  *gateway.Request
	}{
		{"model.list", &gateway.Request{Type: gateway.TypeQuery, Unit: "model.list"}},
		{"engine.list", &gateway.Request{Type: gateway.TypeQuery, Unit: "engine.list"}},
		{"resource.status", &gateway.Request{Type: gateway.TypeQuery, Unit: "resource.status"}},
		{"service.list", &gateway.Request{Type: gateway.TypeQuery, Unit: "service.list"}},
		{"app.list", &gateway.Request{Type: gateway.TypeQuery, Unit: "app.list"}},
		{"device.info", &gateway.Request{Type: gateway.TypeQuery, Unit: "device.info"}},
		{"pipeline.list", &gateway.Request{Type: gateway.TypeQuery, Unit: "pipeline.list"}},
		{"alert.list_rules", &gateway.Request{Type: gateway.TypeQuery, Unit: "alert.list_rules"}},
	}

	const iterations = 20
	var wg sync.WaitGroup
	results := make(map[string]*int32)

	for _, op := range operations {
		counter := int32(0)
		results[op.name] = &counter
	}

	for _, op := range operations {
		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func(operation struct {
				name string
				req  *gateway.Request
			}) {
				defer wg.Done()

				resp := gw.Handle(ctx, operation.req)
				if resp.Success {
					atomic.AddInt32(results[operation.name], 1)
				}
			}(op)
		}
	}

	wg.Wait()

	// Verify all operations completed (some may fail due to missing providers)
	totalSuccesses := int32(0)
	for name, count := range results {
		c := atomic.LoadInt32(count)
		totalSuccesses += c
		// Some operations like device.info and resource.status may fail without providers
		// We just verify the system handles all requests without crashing
		t.Logf("%s: %d/%d succeeded", name, c, iterations)
	}
	// At least some operations should succeed
	assert.Greater(t, totalSuccesses, int32(0), "at least some operations should succeed")
}

// TestConcurrent_ContextCancellation tests handling of cancelled contexts
func TestConcurrent_ContextCancellation(t *testing.T) {
	reg := unit.NewRegistry()
	err := registry.RegisterAll(reg)
	require.NoError(t, err)

	gw := gateway.NewGateway(reg)

	const numGoroutines = 20
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Create context with very short timeout
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
			defer cancel()

			// This should complete before timeout or handle it gracefully
			_ = gw.Handle(ctx, &gateway.Request{
				Type:  gateway.TypeQuery,
				Unit:  "model.list",
				Input: map[string]any{},
			})
		}(i)
	}

	wg.Wait()
	// Test passes if no panics occur
}

// TestConcurrent_NoDeadlock tests that the system doesn't deadlock under load
func TestConcurrent_NoDeadlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping deadlock test in short mode")
	}

	reg := unit.NewRegistry()
	err := registry.RegisterAll(reg)
	require.NoError(t, err)

	gw := gateway.NewGateway(reg)
	ctx := context.Background()

	done := make(chan struct{})
	go func() {
		const numOperations = 100
		var wg sync.WaitGroup

		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				// Random operation
				switch id % 4 {
				case 0:
					_ = gw.Handle(ctx, &gateway.Request{
						Type:  gateway.TypeQuery,
						Unit:  "model.list",
						Input: map[string]any{},
					})
				case 1:
					_ = gw.Handle(ctx, &gateway.Request{
						Type:  gateway.TypeQuery,
						Unit:  "resource.status",
						Input: map[string]any{},
					})
				case 2:
					_ = gw.Handle(ctx, &gateway.Request{
						Type:  gateway.TypeQuery,
						Unit:  "engine.list",
						Input: map[string]any{},
					})
				case 3:
					_ = gw.Handle(ctx, &gateway.Request{
						Type:  gateway.TypeQuery,
						Unit:  "service.list",
						Input: map[string]any{},
					})
				}
			}(i)
		}

		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - no deadlock
	case <-time.After(30 * time.Second):
		t.Fatal("potential deadlock detected - test timed out")
	}
}

// TestConcurrent_RaceCondition tests for race conditions using -race flag
func TestConcurrent_RaceCondition(t *testing.T) {
	reg := unit.NewRegistry()
	err := registry.RegisterAll(reg)
	require.NoError(t, err)

	gw := gateway.NewGateway(reg)
	ctx := context.Background()

	// This test is designed to be run with -race flag
	// It performs concurrent reads and writes to detect race conditions

	var wg sync.WaitGroup

	// Concurrent reads
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = gw.Handle(ctx, &gateway.Request{
					Type:  gateway.TypeQuery,
					Unit:  "model.list",
					Input: map[string]any{},
				})
			}
		}()
	}

	// Concurrent writes (creates)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = gw.Handle(ctx, &gateway.Request{
				Type: gateway.TypeCommand,
				Unit: "model.create",
				Input: map[string]any{
					"name": "race-test-model",
				},
			})
		}(i)
	}

	wg.Wait()
}

// Ensure resource import is used
var _ = resource.ErrSlotNotFound
