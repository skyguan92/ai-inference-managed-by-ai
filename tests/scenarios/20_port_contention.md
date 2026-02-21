# Scenario 20: Port Contention & Resource Exhaustion

**Difficulty**: Medium
**Estimated Duration**: 10 minutes
**Prerequisites**: Scenarios 1-2 completed, Docker available
**Tier**: 4 — Resilience

## User Story

"I'm running AIMA on a shared server where other processes may occupy ports. I want to verify that AIMA handles port conflicts gracefully — clear error messages, proper cleanup, no resource leaks — and that repeated start/fail cycles don't corrupt the system state."

## Success Criteria

1. [ ] Port occupied by external process → clear error message (not cryptic Docker error)
2. [ ] After port conflict, service status transitions to "failed" (not stuck at "creating")
3. [ ] After failure, Docker container is cleaned up (no orphaned containers)
4. [ ] Docker daemon unavailable → clear error (not hung forever)
5. [ ] Retry error messages include details from all retry attempts
6. [ ] `service stop` works on "failed" services (cleans up any residual state)
7. [ ] No resource leaks after 3 consecutive start/fail cycles (no orphaned containers, no port leaks)

## Environment Setup

```bash
/tmp/aima start &
AIMA_PID=$!
sleep 2

# Create a model and service for testing
/tmp/aima model create --name "port-test-model" --source ollama --repo "tinyllama"
MODEL_ID=$(/tmp/aima model list --output json | jq -r '.items[-1].id')
```

### Test 1: External Port Conflict

```bash
# Block port 8000 with an external process
python3 -c "
import socket, time
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
s.bind(('0.0.0.0', 8000))
s.listen(1)
print('Blocking port 8000')
time.sleep(300)
" &
BLOCKER_PID=$!
sleep 1

# Verify port is blocked
ss -tlnp | grep 8000

# Try to start a service (will try port 8000 first)
/tmp/aima service create --model-id "$MODEL_ID" --engine-type vllm
SVC_ID=$(/tmp/aima service list --output json | jq -r '.items[-1].id')

/tmp/aima service start "$SVC_ID" --wait --timeout 60
echo "Exit code: $?"
# Expected: failure with clear message about port conflict
# BUG WATCH: fatalStartError should trigger fast-fail for non-AIMA port conflict

# Release the port
kill $BLOCKER_PID 2>/dev/null
```

### Test 2: Service Status After Failure

```bash
/tmp/aima service list --output json | jq ".items[] | select(.id==\"$SVC_ID\") | .status"
# Expected: "failed" (not "creating")
```

### Test 3: Container Cleanup After Failure

```bash
docker ps -a --filter label=aima.engine --format "{{.Names}} {{.Status}}"
# Expected: no orphaned containers from the failed start
# The pre-start cleanup + failure cleanup should have removed everything
```

### Test 4: Docker Daemon Unavailable

```bash
# This test requires temporarily making Docker unavailable
# Option A: Stop Docker (if safe to do)
# Option B: Use a config with wrong Docker host

# Create a service and try to start with wrong Docker socket
DOCKER_HOST=tcp://127.0.0.1:99999 /tmp/aima service start "$SVC_ID" --wait --timeout 30
echo "Exit code: $?"
# Expected: clear error about Docker connection failure
# Not: hung forever or cryptic timeout
```

### Test 5: Retry Error Aggregation

```bash
# The start attempt from Test 1 should have retried multiple times
# Check if the error message includes details from each retry attempt
# Look at the output from Test 1 — it should mention retry count and individual errors

# Re-test with a fresh service
/tmp/aima service create --model-id "$MODEL_ID" --engine-type vllm
SVC2=$(/tmp/aima service list --output json | jq -r '.items[-1].id')

# Block port again
python3 -c "import socket,time;s=socket.socket();s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1);s.bind(('0.0.0.0',8000));s.listen(1);time.sleep(60)" &
BLOCKER2=$!
sleep 1

/tmp/aima service start "$SVC2" --wait --timeout 60 2>&1
# Expected: error message mentions retry attempts and individual failure reasons

kill $BLOCKER2 2>/dev/null
```

### Test 6: Stop Failed Services

```bash
/tmp/aima service stop "$SVC_ID"
echo "Stop result: $?"
# Expected: success — gracefully handles "failed" state

/tmp/aima service stop "$SVC2"
echo "Stop result: $?"
# Expected: success
```

### Test 7: Repeated Start/Fail Cycles — No Leaks

```bash
/tmp/aima service create --model-id "$MODEL_ID" --engine-type vllm
CYCLE_SVC=$(/tmp/aima service list --output json | jq -r '.items[-1].id')

# Block port
python3 -c "import socket,time;s=socket.socket();s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1);s.bind(('0.0.0.0',8000));s.listen(1);time.sleep(120)" &
BLOCKER3=$!
sleep 1

# 3 consecutive start/fail cycles
for i in 1 2 3; do
  echo "=== Cycle $i ==="
  /tmp/aima service start "$CYCLE_SVC" --wait --timeout 30
  echo "Exit code: $?"
  docker ps -a --filter label=aima.engine --format "{{.Names}} {{.Status}}" | wc -l
  echo "---"
done

# Check for resource leaks
docker ps -a --filter label=aima.engine
# Expected: NO orphaned containers (each cycle should clean up after itself)

kill $BLOCKER3 2>/dev/null

# Clean up
/tmp/aima service stop "$CYCLE_SVC"
```

## Cleanup

```bash
# Kill any lingering blockers
kill $BLOCKER_PID $BLOCKER2 $BLOCKER3 2>/dev/null

# Stop services
/tmp/aima service list --output json | jq -r '.items[].id' | while read id; do
  /tmp/aima service stop "$id" 2>/dev/null
done

# Verify clean Docker state
docker ps -a --filter label=aima.engine
docker ps -a --filter label=aima.engine -q | xargs -r docker rm -f

kill $AIMA_PID 2>/dev/null
```

## Hints for the Operator

- Port conflict handling: `pkg/infra/provider/hybrid_engine_provider.go` (fatalStartError)
- Container cleanup: Pre-start cleanup + failure cleanup in same file
- Service state transitions: `pkg/unit/service/commands.go`
- Docker client error handling: `pkg/infra/docker/sdk_client.go`

## Known Pitfalls

- **fatalStartError for non-AIMA conflicts**: Port conflicts from non-AIMA processes should trigger `fatalStartError` which fast-fails (<1s) instead of retrying 5 times (~50s). This was fixed in commit 4d57394 but verify it works.
- **Bug #31 — Port release timing**: Docker's proxy process may hold a port briefly after container stop. Between cycles, the previous container's port may not be released yet.
- **Container accumulation**: Without proper cleanup, each retry creates a new container. After 3 retries × 3 cycles = 9 potential orphans. The pre-start cleanup by label should prevent this.
- **Service state corruption**: Repeated start/fail cycles may leave the service in an inconsistent state where neither start nor stop works. The "failed" status handling should prevent this.
- **Docker daemon error handling**: If the Docker daemon itself is down, the Docker SDK client should return a clear error quickly, not wait for a TCP timeout.

## Key Code to Inspect

| File | What to Check |
|------|---------------|
| `pkg/infra/provider/hybrid_engine_provider.go` | fatalStartError, pre-start cleanup, retry logic |
| `pkg/unit/service/commands.go` | Status transitions for failed starts |
| `pkg/infra/docker/sdk_client.go` | Error handling for connection failures |
| `pkg/infra/docker/simple_client.go` | CLI fallback error handling |

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- How fast the port conflict is detected (should be <1s with fatalStartError)
- Number of orphaned containers after all tests
- Whether stop works reliably on failed services
