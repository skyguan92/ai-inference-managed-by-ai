# Scenario 10: Pipeline DAG Execution

**Difficulty**: Medium
**Estimated Duration**: 5 minutes
**Prerequisites**: Scenario 1 completed (binary built), HTTP server running
**Tier**: 1 — Foundation (no Docker/GPU required)

## User Story

"I want to create automated pipelines that chain multiple AIMA operations together — like 'pull a model, create a service, start the service, run inference'. I need DAG-style execution where steps can have dependencies, and I want to be able to cancel a running pipeline and check its status."

## Success Criteria

1. [ ] 3-step DAG pipeline created successfully (step A → step B → step C, where B depends on A, C depends on B)
2. [ ] Circular dependency rejected with clear error (step A depends on B, B depends on A)
3. [ ] Duplicate step IDs in the same pipeline rejected
4. [ ] Async pipeline run returns a `run_id` immediately
5. [ ] Status polling shows transitions: pending → running → completed
6. [ ] Cancel of a running pipeline works (transitions to "cancelled")
7. [ ] Delete of a running pipeline is rejected (must cancel or wait for completion first)
8. [ ] List shows correct statuses for all pipelines

## Environment Setup

```bash
# Start AIMA HTTP server (no Docker needed for pipeline logic)
/tmp/aima start &
AIMA_PID=$!
sleep 2

# Verify server is up
curl -s http://localhost:9090/health
```

### Test 1: Create a 3-Step DAG Pipeline

```bash
curl -s -X POST http://localhost:9090/api/pipeline/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "deploy-and-test",
    "description": "Pull model, create service, test inference",
    "steps": [
      {"id": "pull", "unit": "model.pull", "params": {"source": "ollama", "repo": "tinyllama"}, "depends_on": []},
      {"id": "create-svc", "unit": "service.create", "params": {"model_id": "$pull.model_id"}, "depends_on": ["pull"]},
      {"id": "test-infer", "unit": "inference.chat", "params": {"message": "hello"}, "depends_on": ["create-svc"]}
    ]
  }' | jq .
# Expected: pipeline created with ID
```

### Test 2: Circular Dependency

```bash
curl -s -X POST http://localhost:9090/api/pipeline/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "circular",
    "steps": [
      {"id": "a", "unit": "model.list", "depends_on": ["b"]},
      {"id": "b", "unit": "model.list", "depends_on": ["a"]}
    ]
  }' | jq .
# Expected: error about circular dependency
```

### Test 3: Duplicate Step IDs

```bash
curl -s -X POST http://localhost:9090/api/pipeline/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "dupes",
    "steps": [
      {"id": "step1", "unit": "model.list", "depends_on": []},
      {"id": "step1", "unit": "engine.list", "depends_on": []}
    ]
  }' | jq .
# Expected: error about duplicate step ID "step1"
```

### Test 4: Run Pipeline Async

```bash
# Use the pipeline ID from Test 1
PIPELINE_ID="<from test 1>"

curl -s -X POST http://localhost:9090/api/pipeline/run \
  -H "Content-Type: application/json" \
  -d "{\"pipeline_id\": \"$PIPELINE_ID\"}" | jq .
# Expected: returns run_id immediately
```

### Test 5: Poll Status

```bash
RUN_ID="<from test 4>"

# Poll multiple times
for i in 1 2 3; do
  curl -s http://localhost:9090/api/pipeline/status?run_id=$RUN_ID | jq .status
  sleep 1
done
# Expected: shows status progression
```

### Test 6: Cancel Pipeline

```bash
# Start a new run and cancel it immediately
curl -s -X POST http://localhost:9090/api/pipeline/run \
  -H "Content-Type: application/json" \
  -d "{\"pipeline_id\": \"$PIPELINE_ID\"}" | jq .run_id

NEW_RUN_ID="<from above>"
curl -s -X POST http://localhost:9090/api/pipeline/cancel \
  -H "Content-Type: application/json" \
  -d "{\"run_id\": \"$NEW_RUN_ID\"}" | jq .
# Expected: cancelled successfully
```

### Test 7: Delete Running Pipeline

```bash
curl -s -X DELETE http://localhost:9090/api/pipeline/delete \
  -H "Content-Type: application/json" \
  -d "{\"pipeline_id\": \"$PIPELINE_ID\"}" | jq .
# Expected: error — cannot delete pipeline with active runs
```

### Test 8: List All Pipelines

```bash
curl -s http://localhost:9090/api/pipeline/list | jq .
# Expected: shows all created pipelines with correct statuses
```

## Cleanup

```bash
kill $AIMA_PID 2>/dev/null
```

## Hints for the Operator

- Pipeline domain code: `pkg/unit/pipeline/`
- Routes: check `pkg/gateway/routes.go` for pipeline-related route definitions
- `pipeline.validate` may use wrong HTTP method type
- Watch for InputMapper key mismatches between route definition and handler expectations
- The executor may run steps sequentially even when DAG allows parallelism

## Known Pitfalls

- **`pipeline.validate` route bug**: The route is defined with `TypeQuery` but uses POST method. The gateway may reject this combination or route it incorrectly. Check `pkg/gateway/routes.go` around line 156.
- **`pipeline.cancel` URL mapping**: The route maps `{id}` URL parameter to `run_id`, but users will naturally pass `pipeline_id`. Check if the InputMapper correctly maps the URL parameter.
- **Sequential execution**: The `PipelineExecutor` likely runs steps one-at-a-time even when the DAG would allow parallel execution (e.g., if steps B and C both depend only on A, they could run concurrently).
- **`ValidateSteps` cycle detection**: The cycle detection algorithm may reset its visited set per top-level step, missing cycles that span across branches. Specifically check `pkg/unit/pipeline/commands.go`.
- **Memory-only store**: Pipeline store is likely in-memory. All pipelines will be lost on restart.

## Key Code to Inspect

| File | What to Check |
|------|---------------|
| `pkg/unit/pipeline/commands.go` | Pipeline creation, validation, cycle detection |
| `pkg/unit/pipeline/queries.go` | Status, list, cancel queries |
| `pkg/gateway/routes.go` | Pipeline route definitions (method types, InputMapper) |
| `pkg/unit/pipeline/executor.go` | DAG execution logic (parallel vs sequential) |
| `pkg/unit/pipeline/store.go` | Memory vs persistent store |

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Whether DAG execution is truly parallel or sequential
- Any route/InputMapper mismatches found
