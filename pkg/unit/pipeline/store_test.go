package pipeline

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- NewMemoryStore ---

func TestMemoryStore_NotNil(t *testing.T) {
	s := NewMemoryStore()
	require.NotNil(t, s)
}

// --- CreatePipeline ---

func TestMemoryStore_CreatePipeline_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	p := createTestPipeline("pipe-1", "my-pipeline", PipelineStatusIdle)
	err := s.CreatePipeline(ctx, p)
	require.NoError(t, err)
}

func TestMemoryStore_CreatePipeline_Duplicate_ReturnsError(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	p := createTestPipeline("pipe-1", "my-pipeline", PipelineStatusIdle)
	require.NoError(t, s.CreatePipeline(ctx, p))

	err := s.CreatePipeline(ctx, p)
	assert.ErrorIs(t, err, ErrPipelineAlreadyExists)
}

// --- GetPipeline ---

func TestMemoryStore_GetPipeline_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	p := createTestPipeline("pipe-1", "my-pipeline", PipelineStatusIdle)
	require.NoError(t, s.CreatePipeline(ctx, p))

	got, err := s.GetPipeline(ctx, "pipe-1")
	require.NoError(t, err)
	assert.Equal(t, "pipe-1", got.ID)
	assert.Equal(t, "my-pipeline", got.Name)
	assert.Equal(t, PipelineStatusIdle, got.Status)
}

func TestMemoryStore_GetPipeline_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	_, err := s.GetPipeline(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrPipelineNotFound)
}

// --- UpdatePipeline ---

func TestMemoryStore_UpdatePipeline_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	p := createTestPipeline("pipe-1", "old-name", PipelineStatusIdle)
	require.NoError(t, s.CreatePipeline(ctx, p))

	p.Name = "new-name"
	p.Status = PipelineStatusRunning
	require.NoError(t, s.UpdatePipeline(ctx, p))

	got, err := s.GetPipeline(ctx, "pipe-1")
	require.NoError(t, err)
	assert.Equal(t, "new-name", got.Name)
	assert.Equal(t, PipelineStatusRunning, got.Status)
}

func TestMemoryStore_UpdatePipeline_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	p := createTestPipeline("nonexistent", "pipeline", PipelineStatusIdle)
	err := s.UpdatePipeline(ctx, p)
	assert.ErrorIs(t, err, ErrPipelineNotFound)
}

// --- DeletePipeline ---

func TestMemoryStore_DeletePipeline_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	p := createTestPipeline("pipe-1", "my-pipeline", PipelineStatusIdle)
	require.NoError(t, s.CreatePipeline(ctx, p))

	require.NoError(t, s.DeletePipeline(ctx, "pipe-1"))

	_, err := s.GetPipeline(ctx, "pipe-1")
	assert.ErrorIs(t, err, ErrPipelineNotFound)
}

func TestMemoryStore_DeletePipeline_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	err := s.DeletePipeline(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrPipelineNotFound)
}

// --- ListPipelines ---

func TestMemoryStore_ListPipelines_All(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.CreatePipeline(ctx, createTestPipeline("p1", "pipe-1", PipelineStatusIdle)))
	require.NoError(t, s.CreatePipeline(ctx, createTestPipeline("p2", "pipe-2", PipelineStatusRunning)))
	require.NoError(t, s.CreatePipeline(ctx, createTestPipeline("p3", "pipe-3", PipelineStatusIdle)))

	pipelines, total, err := s.ListPipelines(ctx, PipelineFilter{})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, pipelines, 3)
}

func TestMemoryStore_ListPipelines_FilterByStatus(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.CreatePipeline(ctx, createTestPipeline("p1", "pipe-1", PipelineStatusIdle)))
	require.NoError(t, s.CreatePipeline(ctx, createTestPipeline("p2", "pipe-2", PipelineStatusRunning)))

	pipelines, total, err := s.ListPipelines(ctx, PipelineFilter{Status: PipelineStatusIdle})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, PipelineStatusIdle, pipelines[0].Status)
}

func TestMemoryStore_ListPipelines_Limit(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		id := "p" + string(rune('0'+i))
		require.NoError(t, s.CreatePipeline(ctx, createTestPipeline(id, id, PipelineStatusIdle)))
	}

	pipelines, total, err := s.ListPipelines(ctx, PipelineFilter{Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, pipelines, 2)
}

func TestMemoryStore_ListPipelines_Offset(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		id := "p" + string(rune('0'+i))
		require.NoError(t, s.CreatePipeline(ctx, createTestPipeline(id, id, PipelineStatusIdle)))
	}

	pipelines, total, err := s.ListPipelines(ctx, PipelineFilter{Offset: 3})
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Len(t, pipelines, 1)
}

// --- CreateRun ---

func TestMemoryStore_CreateRun_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	run := createTestRun("run-1", "pipe-1", RunStatusPending)
	err := s.CreateRun(ctx, run)
	require.NoError(t, err)
}

func TestMemoryStore_CreateRun_StoresDeepCopy(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	run := createTestRun("run-1", "pipe-1", RunStatusPending)
	require.NoError(t, s.CreateRun(ctx, run))

	// Mutate original run — store copy should be unaffected
	run.Status = RunStatusCompleted

	stored, err := s.GetRun(ctx, "run-1")
	require.NoError(t, err)
	assert.Equal(t, RunStatusPending, stored.Status, "store should hold original status after mutation")
}

// --- GetRun ---

func TestMemoryStore_GetRun_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	run := createTestRun("run-1", "pipe-1", RunStatusRunning)
	require.NoError(t, s.CreateRun(ctx, run))

	got, err := s.GetRun(ctx, "run-1")
	require.NoError(t, err)
	assert.Equal(t, "run-1", got.ID)
	assert.Equal(t, "pipe-1", got.PipelineID)
	assert.Equal(t, RunStatusRunning, got.Status)
}

func TestMemoryStore_GetRun_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	_, err := s.GetRun(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrRunNotFound)
}

// --- UpdateRun ---

func TestMemoryStore_UpdateRun_Success(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	run := createTestRun("run-1", "pipe-1", RunStatusRunning)
	require.NoError(t, s.CreateRun(ctx, run))

	run.Status = RunStatusCompleted
	now := time.Now()
	run.CompletedAt = &now
	require.NoError(t, s.UpdateRun(ctx, run))

	got, err := s.GetRun(ctx, "run-1")
	require.NoError(t, err)
	assert.Equal(t, RunStatusCompleted, got.Status)
	assert.NotNil(t, got.CompletedAt)
}

func TestMemoryStore_UpdateRun_StoresDeepCopy(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	run := createTestRun("run-1", "pipe-1", RunStatusRunning)
	require.NoError(t, s.CreateRun(ctx, run))

	run.StepResults = map[string]any{"step1": "output"}
	require.NoError(t, s.UpdateRun(ctx, run))

	// Mutate step results after update
	run.StepResults["step1"] = "mutated"

	stored, err := s.GetRun(ctx, "run-1")
	require.NoError(t, err)
	assert.Equal(t, "output", stored.StepResults["step1"], "stored copy should not be affected by mutation")
}

func TestMemoryStore_UpdateRun_NotFound(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	run := createTestRun("nonexistent", "pipe-1", RunStatusCompleted)
	err := s.UpdateRun(ctx, run)
	assert.ErrorIs(t, err, ErrRunNotFound)
}

// --- ListRuns ---

func TestMemoryStore_ListRuns_All(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.CreateRun(ctx, createTestRun("r1", "pipe-1", RunStatusCompleted)))
	require.NoError(t, s.CreateRun(ctx, createTestRun("r2", "pipe-1", RunStatusFailed)))
	require.NoError(t, s.CreateRun(ctx, createTestRun("r3", "pipe-2", RunStatusRunning)))

	runs, err := s.ListRuns(ctx, "")
	require.NoError(t, err)
	assert.Len(t, runs, 3)
}

func TestMemoryStore_ListRuns_FilterByPipelineID(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.CreateRun(ctx, createTestRun("r1", "pipe-1", RunStatusCompleted)))
	require.NoError(t, s.CreateRun(ctx, createTestRun("r2", "pipe-2", RunStatusRunning)))
	require.NoError(t, s.CreateRun(ctx, createTestRun("r3", "pipe-1", RunStatusFailed)))

	runs, err := s.ListRuns(ctx, "pipe-1")
	require.NoError(t, err)
	assert.Len(t, runs, 2)
	for _, r := range runs {
		assert.Equal(t, "pipe-1", r.PipelineID)
	}
}

func TestMemoryStore_ListRuns_Empty(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	runs, err := s.ListRuns(ctx, "")
	require.NoError(t, err)
	assert.Empty(t, runs)
}

// --- Concurrent access ---

func TestMemoryStore_ConcurrentPipelineOps_NoRace(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		id := "pipe-" + string(rune('0'+i))
		require.NoError(t, s.CreatePipeline(ctx, createTestPipeline(id, id, PipelineStatusIdle)))
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = s.ListPipelines(ctx, PipelineFilter{})
			_, _ = s.GetPipeline(ctx, "pipe-0")
		}()
	}
	wg.Wait()
}

func TestMemoryStore_ConcurrentRunOps_NoRace(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	// Pre-seed a run
	run := createTestRun("run-shared", "pipe-1", RunStatusRunning)
	require.NoError(t, s.CreateRun(ctx, run))

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := s.GetRun(ctx, "run-shared")
			if err == nil {
				_ = s.UpdateRun(ctx, r)
			}
		}()
	}
	wg.Wait()
}
