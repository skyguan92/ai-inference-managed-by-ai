# Scenario 22: Graceful Shutdown Under Load

**Difficulty**: Medium
**Estimated Duration**: 10 minutes
**Prerequisites**: Scenarios 1-2 completed, Docker + GPU available
**Tier**: 4 — Resilience

## User Story

"I'm running AIMA in production and need to do a rolling upgrade. When I send SIGTERM, I need in-flight requests to complete, Docker containers to keep running independently, and SQLite data to remain intact. I also need to make sure a hard kill (SIGKILL) doesn't corrupt the database."

## Success Criteria

1. [ ] SIGTERM stops accepting new connections immediately
2. [ ] In-flight requests complete before process exits (within ShutdownTimeout)
3. [ ] Docker containers continue running after AIMA process exits
4. [ ] SQLite database is not corrupted after SIGTERM
5. [ ] Process can restart successfully after SIGTERM
6. [ ] SQLite database is not corrupted even after SIGKILL

## Environment Setup

```bash
# Start AIMA with a running service
/tmp/aima start &
AIMA_PID=$!
sleep 2

# Create and start a service for testing
/tmp/aima model create --name "shutdown-test" --source ollama --repo "tinyllama"
MODEL_ID=$(/tmp/aima model list --output json | jq -r '.items[-1].id')
/tmp/aima service create --model-id "$MODEL_ID" --engine-type vllm
SVC_ID=$(/tmp/aima service list --output json | jq -r '.items[-1].id')

# Optionally start the service (if you want to test with a running container)
# /tmp/aima service start "$SVC_ID" --wait --timeout 600
```

### Test 1: SIGTERM Stops New Connections

```bash
# Send SIGTERM
kill -TERM $AIMA_PID

# Immediately try a new request
sleep 0.5
curl -s -w "\nHTTP %{http_code}\n" --connect-timeout 2 http://localhost:9090/api/model/list
# Expected: connection refused or timeout (server stopped accepting new connections)

# Wait for process to exit
wait $AIMA_PID 2>/dev/null
echo "Process exit code: $?"
```

### Test 2: In-Flight Request Completion

```bash
# Restart AIMA
/tmp/aima start &
AIMA_PID=$!
sleep 2

# Start a long-running request in background
curl -s http://localhost:9090/api/model/list > /tmp/inflight-result.json &
CURL_PID=$!
sleep 0.1

# Send SIGTERM while request is in-flight
kill -TERM $AIMA_PID

# Check if the in-flight request completed
wait $CURL_PID
CURL_EXIT=$?
echo "Curl exit code: $CURL_EXIT"
cat /tmp/inflight-result.json | jq .
# Expected: response received (request completed before shutdown)
# Note: For a fast query like model.list, this may complete before SIGTERM is processed

wait $AIMA_PID 2>/dev/null
```

### Test 3: Docker Container Independence

```bash
# Restart AIMA and start a service with Docker container
/tmp/aima start &
AIMA_PID=$!
sleep 2

# Check Docker containers before shutdown
docker ps --filter label=aima.engine --format "{{.Names}} {{.Status}}"
CONTAINER_COUNT_BEFORE=$(docker ps --filter label=aima.engine -q | wc -l)

# Send SIGTERM
kill -TERM $AIMA_PID
wait $AIMA_PID 2>/dev/null

# Check Docker containers after shutdown
docker ps --filter label=aima.engine --format "{{.Names}} {{.Status}}"
CONTAINER_COUNT_AFTER=$(docker ps --filter label=aima.engine -q | wc -l)

echo "Containers before: $CONTAINER_COUNT_BEFORE, after: $CONTAINER_COUNT_AFTER"
# Expected: same count — Docker containers are independent of AIMA process
```

### Test 4: SQLite Integrity After SIGTERM

```bash
# Check SQLite database integrity
SQLITE_PATH="$HOME/.aima/data/aima.db"  # Adjust path as needed
# Find the actual SQLite file
find ~/.aima/ -name "*.db" -o -name "*.sqlite" 2>/dev/null

# If SQLite file exists, check integrity
sqlite3 $SQLITE_PATH "PRAGMA integrity_check;"
# Expected: "ok"

# Verify data is readable
sqlite3 $SQLITE_PATH "SELECT COUNT(*) FROM models;" 2>/dev/null
# Expected: returns a count (not an error)
```

### Test 5: Successful Restart After SIGTERM

```bash
/tmp/aima start &
AIMA_PID=$!
sleep 2

# Verify everything works
/tmp/aima model list
/tmp/aima service list
echo "Restart successful"

kill -TERM $AIMA_PID
wait $AIMA_PID 2>/dev/null
```

### Test 6: SQLite Integrity After SIGKILL

```bash
# Restart and add some data
/tmp/aima start &
AIMA_PID=$!
sleep 2

# Write some data
/tmp/aima model create --name "sigkill-test" --source ollama --repo "phi"

# Hard kill (SIGKILL — no cleanup)
kill -9 $AIMA_PID
wait $AIMA_PID 2>/dev/null

# Check SQLite integrity
sqlite3 $SQLITE_PATH "PRAGMA integrity_check;"
# Expected: "ok" (SQLite WAL should handle this)

# Verify data
/tmp/aima start &
AIMA_PID=$!
sleep 2
/tmp/aima model list --output json | jq '.items[] | select(.name=="sigkill-test")'
# Expected: model survived SIGKILL (SQLite WAL preserved it)

kill -TERM $AIMA_PID
wait $AIMA_PID 2>/dev/null
```

## Cleanup

```bash
# Stop Docker containers
docker ps --filter label=aima.engine -q | xargs -r docker stop
docker ps -a --filter label=aima.engine -q | xargs -r docker rm

kill $AIMA_PID 2>/dev/null
```

## Hints for the Operator

- Shutdown handler: `pkg/gateway/server.go` (graceful shutdown with context)
- ShutdownTimeout: check for a configurable timeout (default may be 10s)
- SQLite WAL: `PRAGMA journal_mode=WAL` should be enabled for crash safety
- Signal handling: check `pkg/cli/start.go` for signal registration

## Known Pitfalls

- **ShutdownTimeout too short**: If the shutdown timeout is 10 seconds and an inference request takes 30 seconds, in-flight inference will be killed. The timeout should be at least as long as the longest expected request.
- **Pipeline goroutines orphaned**: Running pipeline executors may be orphaned during shutdown. There's no context cancellation propagation to pipeline goroutine pools.
- **SQLite WAL integrity**: SQLite with WAL journaling should survive SIGKILL without corruption. But if the WAL file grows very large (due to long-running transactions) and SIGKILL happens, the WAL replay on restart could be slow.
- **Docker socket cleanup**: If AIMA has open connections to the Docker daemon, SIGTERM should close them cleanly. SIGKILL will leave them open (Docker daemon handles this gracefully).
- **File descriptor leaks**: After SIGKILL, open file descriptors (SQLite, config file, log file) are abandoned. The OS cleans these up, but verify no lock files are left behind.

## Key Code to Inspect

| File | What to Check |
|------|---------------|
| `pkg/gateway/server.go` | Graceful shutdown, ShutdownTimeout |
| `pkg/cli/start.go` | Signal handling, cleanup logic |
| Store implementations | SQLite WAL mode, connection closing |
| `pkg/infra/docker/sdk_client.go` | Client connection cleanup |

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Actual ShutdownTimeout value found in code
- Whether in-flight requests complete or are terminated
- SQLite integrity check results after both SIGTERM and SIGKILL
