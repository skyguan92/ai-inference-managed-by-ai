# Scenario 19: Process Restart Data Survival

**Difficulty**: Medium
**Estimated Duration**: 10 minutes
**Prerequisites**: Scenarios 1-2 completed, Docker available
**Tier**: 4 — Resilience

## User Story

"My AIMA server process crashed and I need to restart it. I want to know: what data survives the restart? Are my models still there? Are my running Docker containers still accessible? Do I need to re-create everything from scratch? This is critical for production reliability."

## Success Criteria

1. [ ] Models survive restart (SQLite-persisted)
2. [ ] Services survive restart (SQLite-persisted, status may need reconciliation)
3. [ ] Docker containers continue running after AIMA process restart
4. [ ] `service stop` works correctly after restart (finds containers by Docker labels)
5. [ ] Engines are re-seeded from YAML assets (correct behavior, not a bug)
6. [ ] Catalog recipes are lost on restart (memory-only store — gap confirmed)
7. [ ] Skills are lost on restart (memory-only store — gap confirmed)
8. [ ] Pipelines are lost on restart (memory-only store — gap confirmed)

## Environment Setup

### Phase 1: Create Data in All Domains

```bash
/tmp/aima start &
AIMA_PID=$!
sleep 2

# Model (SQLite-persisted)
/tmp/aima model create --name "restart-test-model" --source ollama --repo "tinyllama"

# Service (SQLite-persisted)
MODEL_ID=$(/tmp/aima model list --output json | jq -r '.items[] | select(.name=="restart-test-model") | .id')
/tmp/aima service create --model-id "$MODEL_ID" --engine-type vllm

# Engine (YAML-seeded, always re-loaded)
/tmp/aima engine list

# Catalog recipe (likely memory-only)
curl -s -X POST http://localhost:9090/api/catalog/create \
  -H "Content-Type: application/json" \
  -d '{"name": "restart-test-recipe", "engine_type": "vllm", "hardware": {"gpu_vendor": "NVIDIA"}, "config": {}}' | jq .

# Skill (likely memory-only)
curl -s -X POST http://localhost:9090/api/skill/create \
  -H "Content-Type: application/json" \
  -d '{"name": "restart-test-skill", "description": "Test skill for restart survival", "keywords": ["restart", "test"], "category": "testing", "content": "test content"}' | jq .

# Pipeline (likely memory-only)
curl -s -X POST http://localhost:9090/api/pipeline/create \
  -H "Content-Type: application/json" \
  -d '{"name": "restart-test-pipeline", "steps": [{"id": "step1", "unit": "model.list", "depends_on": []}]}' | jq .

# Alert rule (likely memory-only)
curl -s -X POST http://localhost:9090/api/alert/create \
  -H "Content-Type: application/json" \
  -d '{"name": "restart-test-alert", "condition": "gpu_temp > 80", "action": "notify"}' | jq .

# Record what we created
echo "=== Before Restart ==="
echo "Models:" && /tmp/aima model list
echo "Services:" && /tmp/aima service list
echo "Engines:" && /tmp/aima engine list
echo "Catalogs:" && curl -s http://localhost:9090/api/catalog/list | jq .
echo "Skills:" && curl -s http://localhost:9090/api/skill/list | jq .
echo "Pipelines:" && curl -s http://localhost:9090/api/pipeline/list | jq .
echo "Alerts:" && curl -s http://localhost:9090/api/alert/list | jq .
```

### Phase 2: Kill and Restart

```bash
# Kill the AIMA process (simulates crash)
kill -9 $AIMA_PID
sleep 2

# Verify Docker containers are still running (if any were started)
docker ps --filter label=aima.engine

# Restart AIMA
/tmp/aima start &
AIMA_PID=$!
sleep 2
```

### Phase 3: Check What Survived

```bash
echo "=== After Restart ==="

# Test 1: Models (should survive)
echo "Models:" && /tmp/aima model list
/tmp/aima model list --output json | jq '.items[] | select(.name=="restart-test-model")'
# Expected: model still present

# Test 2: Services (should survive)
echo "Services:" && /tmp/aima service list
# Expected: service still present (status may show stale state)

# Test 3: Docker containers (should still be running)
docker ps --filter label=aima.engine
# Expected: containers unaffected by AIMA process restart

# Test 4: Service stop after restart
SVC_ID=$(/tmp/aima service list --output json | jq -r '.items[0].id')
/tmp/aima service stop "$SVC_ID"
# Expected: works — finds container by Docker label fallback

# Test 5: Engines (re-seeded from YAML)
echo "Engines:" && /tmp/aima engine list
# Expected: engines present (re-loaded from embedded YAML assets)

# Test 6: Catalogs (likely LOST)
echo "Catalogs:" && curl -s http://localhost:9090/api/catalog/list | jq .
# Expected: restart-test-recipe is GONE (memory-only store)

# Test 7: Skills (likely LOST)
echo "Skills:" && curl -s http://localhost:9090/api/skill/list | jq .
# Expected: custom restart-test-skill is GONE; built-in skills may be re-loaded from embedded YAML

# Test 8: Pipelines (likely LOST)
echo "Pipelines:" && curl -s http://localhost:9090/api/pipeline/list | jq .
# Expected: restart-test-pipeline is GONE (memory-only store)

echo "Alerts:" && curl -s http://localhost:9090/api/alert/list | jq .
# Expected: restart-test-alert is GONE (memory-only store)
```

## Cleanup

```bash
# Stop any remaining Docker containers
docker ps --filter label=aima.engine -q | xargs -r docker stop
docker ps -a --filter label=aima.engine -q | xargs -r docker rm

kill $AIMA_PID 2>/dev/null
```

## Hints for the Operator

- Store implementations vary by domain — check `pkg/registry/register.go`
- SQLite stores: `pkg/unit/model/store.go`, `pkg/unit/service/store.go`
- Memory stores: likely all others (pipeline, catalog, skill, alert, resource)
- Engine seeding: `pkg/cli/root.go` (seeds EngineStore from YAML at startup)
- Docker label fallback: `pkg/infra/provider/hybrid_engine_provider.go`

## Known Pitfalls

- **HIGH CONFIDENCE — 5/13 domains memory-only**: Pipeline, catalog, skill, alert, and resource domains all likely use in-memory stores. ALL data in these domains is lost on restart. This is a significant production gap.
- **Service status reconciliation**: After restart, services may show stale status (e.g., "creating" when the container is actually running, or "running" when the container crashed). There's no startup reconciliation that checks Docker container status against stored service status.
- **Engine re-seeding**: Engines are re-loaded from embedded YAML assets on every startup. This is correct behavior (engines are configuration, not user data) but may confuse users who modified engine settings via API — those changes would be lost.
- **Port counter reset**: The `portCounter` resets to the base port on restart. If a container from a previous session is still running on port 8000, the new session's first service will also try port 8000, causing a conflict.
- **Built-in skills re-loaded**: Skills embedded via `//go:embed` (from `skills/` directory) will be re-loaded on restart. Custom skills created via API will be lost.

## Key Code to Inspect

| File | What to Check |
|------|---------------|
| `pkg/registry/register.go` | Which stores are memory vs SQLite |
| `pkg/unit/model/store.go` | FileModelStore (SQLite) |
| `pkg/unit/service/store.go` | Service store type |
| `pkg/unit/pipeline/store.go` | Pipeline store type (likely memory) |
| `pkg/unit/catalog/store.go` | Catalog store type (likely memory) |
| `pkg/unit/skill/store.go` | Skill store type (likely memory) |
| `pkg/cli/root.go` | Startup initialization, store creation |

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Complete table: which domains survived restart, which lost data
- Whether service status is accurate after restart (or stale)
- Whether built-in skills are correctly re-loaded
