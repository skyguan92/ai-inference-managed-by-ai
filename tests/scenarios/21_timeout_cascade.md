# Scenario 21: Timeout Cascade

**Difficulty**: Medium
**Estimated Duration**: 10 minutes
**Prerequisites**: Scenarios 1-2 completed, Docker + GPU available
**Tier**: 4 — Resilience

## User Story

"I'm deploying large language models that take 5-10 minutes to load into GPU memory. I need to verify that AIMA's timeout chain works correctly — from CLI flags to gateway to Docker — and that timeouts at each level result in proper cleanup, not orphaned resources. I also need to make sure that long-running operations don't block other requests."

## Success Criteria

1. [ ] CLI `--timeout` flag is propagated through gateway to Docker operations
2. [ ] MCP tool `timeout` parameter works for service.start
3. [ ] Short timeout → container is cleaned up (not left running orphaned)
4. [ ] After timeout, service status transitions to "failed"
5. [ ] Retry after timeout works without manual cleanup
6. [ ] Agent chat (10min timeout) is NOT killed by HTTP WriteTimeout (30s)
7. [ ] Concurrent requests are served during long-running service start

## Environment Setup

```bash
/tmp/aima start &
AIMA_PID=$!
sleep 2

# Create a model/service for testing
/tmp/aima model create --name "timeout-test" --source ollama --repo "tinyllama"
MODEL_ID=$(/tmp/aima model list --output json | jq -r '.items[-1].id')
```

### Test 1: CLI Timeout Propagation

```bash
# Create service
/tmp/aima service create --model-id "$MODEL_ID" --engine-type vllm
SVC_ID=$(/tmp/aima service list --output json | jq -r '.items[-1].id')

# Start with short timeout to force a timeout
/tmp/aima service start "$SVC_ID" --wait --timeout 5
echo "Exit code: $?"
# Expected: times out after ~5 seconds with clear timeout error
# The 5s is too short for vLLM to start, so it should timeout
```

### Test 2: MCP Tool Timeout

```bash
# Test via MCP protocol (simulate tool call with timeout)
(
echo '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2024-11-05", "capabilities": {}, "clientInfo": {"name": "test", "version": "1.0"}}}'
sleep 0.5
echo '{"jsonrpc": "2.0", "method": "notifications/initialized"}'
sleep 0.5
echo '{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": {"name": "service.start", "arguments": {"service_id": "'$SVC_ID'", "timeout": 5}}}'
sleep 10
) | /tmp/aima mcp stdio 2>/dev/null
# Expected: timeout error after ~5 seconds
```

### Test 3: Container Cleanup After Timeout

```bash
# After the timeout from Test 1, check for orphaned containers
sleep 2
docker ps -a --filter label=aima.engine --format "{{.Names}} {{.Status}}"
# Expected: NO containers from the timed-out start (Bug #27 fix ensures cleanup)
```

### Test 4: Status After Timeout

```bash
/tmp/aima service list --output json | jq ".items[] | select(.id==\"$SVC_ID\") | .status"
# Expected: "failed" (not stuck at "creating")
```

### Test 5: Retry After Timeout

```bash
# Try starting the same service again (with longer timeout)
/tmp/aima service start "$SVC_ID" --wait --timeout 600
echo "Exit code: $?"
# Expected: starts successfully without manual cleanup
# The pre-start cleanup should handle any residual containers from the timeout
```

### Test 6: Agent Chat vs WriteTimeout

```bash
# The HTTP WriteTimeout is 30s, but agent chat needs 10min
# Test that a long agent chat request doesn't get killed

# Option A: Direct HTTP test with long-running request
timeout 35 curl -s -X POST http://localhost:9090/api/agent/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "What services are running? Check their status and tell me about each one."}' \
  -w "\nHTTP %{http_code} after %{time_total}s\n"
# Expected: Should NOT return 0 bytes / connection reset after exactly 30s
# If WriteTimeout kills it, you'll see empty response at 30s

# Option B: Check the server config
grep -n "WriteTimeout" /home/qujing/projects/ai-inference-managed-by-ai/pkg/gateway/server.go
# BUG WATCH: WriteTimeout=30s is too short for agent chat (10min) and long inference
```

### Test 7: Concurrent Requests During Long Start

```bash
# Start a service with long timeout in background
/tmp/aima service create --model-id "$MODEL_ID" --engine-type vllm
LONG_SVC=$(/tmp/aima service list --output json | jq -r '.items[-1].id')
/tmp/aima service start "$LONG_SVC" --wait --timeout 600 &
START_PID=$!

# While that's running, verify other endpoints still respond
sleep 3
for i in 1 2 3 4 5; do
  TIME=$(curl -s -o /dev/null -w "%{time_total}" http://localhost:9090/api/model/list)
  echo "Request $i: ${TIME}s"
done
# Expected: all complete quickly (<1s), not blocked by the long-running start

wait $START_PID
```

## Cleanup

```bash
# Stop services
/tmp/aima service list --output json | jq -r '.items[] | select(.status=="running") | .id' | while read id; do
  /tmp/aima service stop "$id" 2>/dev/null
done

# Clean up Docker
docker ps -a --filter label=aima.engine -q | xargs -r docker rm -f

kill $AIMA_PID 2>/dev/null
```

## Hints for the Operator

- Timeout propagation: CLI → `RequestOptions.Timeout` → gateway context → Docker context
- HTTP server config: `pkg/gateway/server.go` (WriteTimeout, ReadTimeout)
- Agent chat timeout: `pkg/cli/agent.go` (agentChatTimeout = 10*time.Minute)
- Container cleanup on timeout: `pkg/infra/provider/hybrid_engine_provider.go` (waitForHealth)

## Known Pitfalls

- **HIGH CONFIDENCE BUG — WriteTimeout kills agent chat**: HTTP `WriteTimeout=30s` in `pkg/gateway/server.go:87` terminates any response that takes longer than 30 seconds to write. Agent chat (10min) and large model loading (7min) both exceed this. The fix is to either increase WriteTimeout or use per-handler timeouts.
- **Timeout cleanup verified**: Bug #27 fix ensures that when `waitForHealth` times out, the container is cleaned up using `context.Background()` (independent of the expired request context). Verify this still works.
- **Context propagation chain**: The timeout must flow: CLI `--timeout 600` → HTTP request with deadline → gateway context → Docker context for health check. Any break in this chain means the timeout isn't enforced.
- **Concurrent request starvation**: If the gateway uses a single-threaded handler or a limited goroutine pool, long-running starts could starve other requests. Go's `net/http` default server uses goroutine-per-request, so this should be fine.
- **MCP timeout vs gateway timeout**: The MCP tool's `timeout` parameter creates a new `context.WithTimeout`, but the gateway may also impose its own timeout. The shorter of the two wins.

## Key Code to Inspect

| File | What to Check |
|------|---------------|
| `pkg/gateway/server.go` | WriteTimeout, ReadTimeout settings |
| `pkg/cli/service.go` | CLI --timeout flag, RequestOptions |
| `pkg/cli/agent.go` | agentChatTimeout constant |
| `pkg/unit/service/commands.go` | Timeout parameter handling |
| `pkg/infra/provider/hybrid_engine_provider.go` | waitForHealth, context propagation |

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Whether WriteTimeout kills agent chat (and what the actual timeout value is)
- Container cleanup confirmation after timeout
- Concurrent request latency during long start operations
