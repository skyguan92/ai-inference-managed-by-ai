# Scenario 18: Service Monitoring & Events

**Difficulty**: Medium
**Estimated Duration**: 10 minutes
**Prerequisites**: Scenarios 1-2 completed, Docker + GPU available
**Tier**: 3 — Production

## User Story

"I'm managing a production AIMA deployment and need observability. When a service starts, I want to see real-time progress (image pull, container create, health check). I want to view container logs for debugging. I need the status state machine to be reliable — if a service fails during startup, the status should reflect that, not get stuck at 'creating' forever."

## Success Criteria

1. [ ] `service start --wait` shows real-time progress phases (pulling image, creating container, starting, health checking)
2. [ ] `service logs` shows actual container output (vLLM startup logs)
3. [ ] Service status correctly reports current state (creating → running or creating → failed)
4. [ ] Failed startup → service status transitions to "failed" (not stuck at "creating")
5. [ ] Progress events are delivered via EventBus (or gap confirmed: EventBus not wired)
6. [ ] Service list shows accurate real-time status for all services
7. [ ] Status polling via HTTP API matches CLI output

## Environment Setup

```bash
/tmp/aima start &
AIMA_PID=$!
sleep 2

# Ensure we have a model to deploy
/tmp/aima model list
MODEL_ID=$(/tmp/aima model list --output json | jq -r '.items[0].id')
```

### Test 1: Service Start with Progress

```bash
# Create a new service
/tmp/aima service create --model-id "$MODEL_ID" --engine-type vllm
SVC_ID=$(/tmp/aima service list --output json | jq -r '.items[-1].id')

# Start with --wait to see progress
/tmp/aima service start "$SVC_ID" --wait --timeout 600
# Expected: shows phase transitions:
#   Phase 1: Cleaning up stale containers
#   Phase 2: Pulling image (if not cached)
#   Phase 3: Creating container
#   Phase 4: Starting container
#   Phase 5: Waiting for health check
# Record: which phases are visible, timing of each phase
```

### Test 2: Service Logs

```bash
# While service is running, check logs
/tmp/aima service logs "$SVC_ID"
# Expected: vLLM startup output (model loading, server ready, etc.)
# Check: does this show Docker container stdout/stderr?
```

```bash
# Also try via HTTP API
curl -s http://localhost:9090/api/service/logs?service_id=$SVC_ID | jq .
# Expected: same log content as CLI
```

### Test 3: Status State Machine

```bash
# Check status via CLI
/tmp/aima service list --output json | jq ".items[] | select(.id==\"$SVC_ID\") | .status"
# Expected: "running" (after successful start)

# Check via HTTP API
curl -s http://localhost:9090/api/service/status?service_id=$SVC_ID | jq .status
# Expected: matches CLI output
```

### Test 4: Failed Startup → "failed" Status

```bash
# Create a service that will fail (e.g., nonexistent model path)
/tmp/aima service create --model-id "nonexistent-model-id" --engine-type vllm
FAIL_SVC=$(/tmp/aima service list --output json | jq -r '.items[-1].id')

# Start it — should fail
/tmp/aima service start "$FAIL_SVC" --wait --timeout 60
echo "Exit code: $?"

# Check status
/tmp/aima service list --output json | jq ".items[] | select(.id==\"$FAIL_SVC\") | .status"
# Expected: "failed" (not "creating")
```

### Test 5: EventBus Delivery

```bash
# Check if events are published during service operations
# The EventBus should emit StartProgressEvent for each phase

# Code analysis check:
grep -n "EventBus" /home/qujing/projects/ai-inference-managed-by-ai/pkg/cli/root.go
grep -n "eventBus" /home/qujing/projects/ai-inference-managed-by-ai/pkg/registry/register.go
# BUG WATCH: EventBus may not be passed to RegisterAll
# Expected finding: EventBus is created but never passed to the registry
```

### Test 6: Service List Accuracy

```bash
# List all services and verify each status is accurate
/tmp/aima service list
# Cross-reference with Docker:
docker ps --filter label=aima.engine --format "{{.Names}} {{.Status}}"
# Expected: AIMA status matches Docker reality
```

### Test 7: HTTP API Status Polling

```bash
# Poll status via HTTP during a new service start
/tmp/aima service create --model-id "$MODEL_ID" --engine-type vllm
NEW_SVC=$(/tmp/aima service list --output json | jq -r '.items[-1].id')

# Start in background
/tmp/aima service start "$NEW_SVC" --wait --timeout 600 &
START_PID=$!

# Poll status every 2 seconds
for i in $(seq 1 10); do
  STATUS=$(curl -s http://localhost:9090/api/service/status?service_id=$NEW_SVC | jq -r .status)
  echo "$(date +%H:%M:%S) Status: $STATUS"
  sleep 2
done

wait $START_PID
```

## Cleanup

```bash
# Stop all running services
/tmp/aima service list --output json | jq -r '.items[] | select(.status=="running") | .id' | while read id; do
  /tmp/aima service stop "$id"
done

kill $AIMA_PID 2>/dev/null
```

## Hints for the Operator

- Progress events: `pkg/unit/engine/events.go` (StartProgressEvent)
- EventBus: `pkg/unit/events.go` (InMemoryEventBus)
- Service logs: `pkg/cli/service.go` (logs command)
- Status state machine: `pkg/unit/service/commands.go`
- Docker logs: `pkg/infra/docker/sdk_client.go` (ContainerLogs)

## Known Pitfalls

- **HIGH CONFIDENCE BUG — EventBus not passed to RegisterAll**: In `pkg/cli/root.go:184-196`, the EventBus is created but NOT passed to `RegisterAll()`. This means `Publish(unit.Event)` calls in the engine provider are silently no-ops. Progress events are generated but never delivered to any subscriber.
- **`service logs` wiring**: The `service logs` CLI command was added in Phase 9.3, but it may not be fully wired to the Docker ContainerLogs API. Check if it actually returns container output or just metadata.
- **State machine gap on provider crash**: If the Docker daemon crashes mid-startup (after container create but before health check), the service may be stuck in "creating" state forever. There's no timeout-based cleanup for this edge case.
- **EventBus type mismatch**: `InMemoryEventBus.Publish(unit.Event)` has a different signature than `unit.EventPublisher.Publish(any)`. This type mismatch may have prevented EventBus integration in `RegisterAll`.
- **Status polling race**: The status may briefly show stale data if polled between the Docker state change and the service store update.

## Key Code to Inspect

| File | What to Check |
|------|---------------|
| `pkg/unit/engine/events.go` | StartProgressEvent definition |
| `pkg/cli/root.go` | EventBus creation and (non-)passing to RegisterAll |
| `pkg/unit/events.go` | InMemoryEventBus implementation |
| `pkg/cli/service.go` | `service logs` CLI command |
| `pkg/unit/service/commands.go` | Status transitions |

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Which progress phases are visible during `--wait`
- Whether EventBus integration gap is confirmed
- Whether `service logs` returns actual Docker output
