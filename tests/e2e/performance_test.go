//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	"github.com/stretchr/testify/require"
)

func TestPerformance_GatewayThroughput(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	env.seedModel(t, "perf-model-1", "perf-model", model.ModelTypeLLM, "/models/perf")

	const goroutines = 50
	const requestsPerGoroutine = 10
	const totalRequests = goroutines * requestsPerGoroutine

	var successCount atomic.Int64
	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				resp := env.query(ctx, "model.get", map[string]any{
					"model_id": "perf-model-1",
				})
				if resp.Success {
					successCount.Add(1)
				}
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	throughput := float64(totalRequests) / elapsed.Seconds()
	t.Logf("Gateway throughput: %.0f req/s", throughput)

	require.Equal(t, int64(totalRequests), successCount.Load(),
		"all %d requests should succeed", totalRequests)
	require.Greater(t, throughput, float64(1000),
		"throughput should exceed 1000 req/s with in-memory stores")
}

func TestPerformance_InferenceLatency(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	const iterations = 10
	durations := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		env.command(ctx, "inference.chat", map[string]any{
			"model": "perf-model",
			"messages": []any{
				map[string]any{"role": "user", "content": "test"},
			},
		})
		durations[i] = time.Since(start)
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	p50 := durations[4]
	p99 := durations[9]
	t.Logf("Inference latency p50=%v p99=%v", p50, p99)

	require.Less(t, p99, 100*time.Millisecond,
		"p99 latency should be under 100ms with mock provider")
}

func TestPerformance_ServiceCreateLatency(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	const iterations = 20
	durations := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		modelID := fmt.Sprintf("perf-model-%d", i)
		env.seedModel(t, modelID, fmt.Sprintf("perf-model-%d", i), model.ModelTypeLLM, fmt.Sprintf("/models/perf-%d", i))

		start := time.Now()
		env.command(ctx, "service.create", map[string]any{
			"model_id": modelID,
		})
		durations[i] = time.Since(start)
	}

	var total time.Duration
	for _, d := range durations {
		total += d
	}
	avg := total / iterations
	t.Logf("Service create avg latency: %v", avg)

	require.Less(t, avg, 10*time.Millisecond,
		"average service create latency should be under 10ms with mock provider")
}
