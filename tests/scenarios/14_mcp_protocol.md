# Scenario 14: MCP Protocol Compliance

**Difficulty**: Medium
**Estimated Duration**: 8 minutes
**Prerequisites**: Scenario 1 completed (binary built)
**Tier**: 2 — API Surface (binary only, no Docker/GPU)

## User Story

"I'm connecting AIMA as a tool provider to Claude Code via the Model Context Protocol (MCP). I need to verify the MCP server implementation: does it properly handle the JSONRPC 2.0 handshake? Does `tools/list` return all available tools? Can I execute tools and get structured responses? I also need to check the SSE transport for web-based clients."

## Success Criteria

1. [ ] MCP `initialize` handshake returns server capabilities (tools, resources, prompts)
2. [ ] `tools/list` returns all registered units from all 13 domains
3. [ ] `tools/call` for `model.list` returns a valid response with proper JSON encoding
4. [ ] Non-existent tool call returns JSONRPC error code -32601 (Method not found)
5. [ ] Malformed JSONRPC request returns parse error (-32700) or invalid request (-32600)
6. [ ] SSE transport establishes a session and returns events
7. [ ] SSE buffer overflow (>100 events) returns 503 (or gap confirmed)

## Environment Setup

The MCP server runs over stdio (for CLI integration) or SSE (for web integration).

### Stdio Mode Testing

```bash
# Create a JSONRPC request file for initialize
cat > /tmp/mcp-init.jsonl << 'EOF'
{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2024-11-05", "capabilities": {}, "clientInfo": {"name": "test-client", "version": "1.0"}}}
EOF

# Send to MCP stdio server
echo '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2024-11-05", "capabilities": {}, "clientInfo": {"name": "test-client", "version": "1.0"}}}' | /tmp/aima mcp stdio 2>/dev/null
# Expected: JSONRPC response with server capabilities
```

### Test 1: Initialize Handshake

```bash
# Full handshake: initialize + initialized notification
(
echo '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2024-11-05", "capabilities": {}, "clientInfo": {"name": "test", "version": "1.0"}}}'
sleep 0.5
echo '{"jsonrpc": "2.0", "method": "notifications/initialized"}'
sleep 0.5
) | /tmp/aima mcp stdio 2>/dev/null | head -5
# Expected: response with serverInfo, capabilities (tools, resources, etc.)
```

### Test 2: Tools List

```bash
(
echo '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2024-11-05", "capabilities": {}, "clientInfo": {"name": "test", "version": "1.0"}}}'
sleep 0.5
echo '{"jsonrpc": "2.0", "method": "notifications/initialized"}'
sleep 0.5
echo '{"jsonrpc": "2.0", "id": 2, "method": "tools/list"}'
sleep 1
) | /tmp/aima mcp stdio 2>/dev/null
# Expected: list of all tools (model.create, model.pull, engine.start, inference.chat, etc.)
# Count: should have tools from all 13 domains
```

### Test 3: Tool Call — model.list

```bash
(
echo '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2024-11-05", "capabilities": {}, "clientInfo": {"name": "test", "version": "1.0"}}}'
sleep 0.5
echo '{"jsonrpc": "2.0", "method": "notifications/initialized"}'
sleep 0.5
echo '{"jsonrpc": "2.0", "id": 3, "method": "tools/call", "params": {"name": "model.list", "arguments": {}}}'
sleep 1
) | /tmp/aima mcp stdio 2>/dev/null
# Expected: response with model list (may be empty array)
# Check: proper JSON encoding of Go `any` typed outputs
```

### Test 4: Non-Existent Tool

```bash
(
echo '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2024-11-05", "capabilities": {}, "clientInfo": {"name": "test", "version": "1.0"}}}'
sleep 0.5
echo '{"jsonrpc": "2.0", "method": "notifications/initialized"}'
sleep 0.5
echo '{"jsonrpc": "2.0", "id": 4, "method": "tools/call", "params": {"name": "nonexistent.tool", "arguments": {}}}'
sleep 1
) | /tmp/aima mcp stdio 2>/dev/null
# Expected: JSONRPC error with code -32601 (Method not found)
```

### Test 5: Malformed JSONRPC

```bash
echo 'this is not json at all' | /tmp/aima mcp stdio 2>/dev/null
# Expected: JSONRPC parse error (-32700)
```

```bash
echo '{"not": "a valid jsonrpc request"}' | /tmp/aima mcp stdio 2>/dev/null
# Expected: JSONRPC invalid request error (-32600)
```

### Test 6: SSE Transport

```bash
# Start AIMA server first (if not already running)
/tmp/aima start &
AIMA_PID=$!
sleep 2

# Connect to SSE endpoint
curl -s -N -H "Accept: text/event-stream" http://localhost:9090/mcp/sse &
SSE_PID=$!
sleep 2

# Send a request to the SSE message endpoint
# (The exact URL format depends on the SSE session ID returned)
# Expected: SSE stream establishes connection and returns session info

kill $SSE_PID 2>/dev/null
```

### Test 7: SSE Buffer Overflow

```bash
# This tests the SSE event buffer limit (100 events)
# Connect to SSE but don't consume events
curl -s -N -H "Accept: text/event-stream" http://localhost:9090/mcp/sse > /dev/null &
SSE_PID=$!
sleep 1

# Send many rapid requests to fill the buffer
for i in $(seq 1 150); do
  curl -s -X POST http://localhost:9090/mcp/message \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc": "2.0", "id": '$i', "method": "tools/list"}' &
done
wait

# Check if the SSE connection got a 503
kill $SSE_PID 2>/dev/null
# Expected: 503 after buffer overflow, or events dropped silently
```

## Cleanup

```bash
kill $AIMA_PID 2>/dev/null
```

## Hints for the Operator

- MCP adapter code: `pkg/gateway/mcp.go` or `pkg/gateway/mcp_adapter.go`
- JSONRPC handling: check the MCP library used (likely `github.com/mark3labs/mcp-go`)
- SSE implementation: check for session management, buffer limits
- Tool registration: `pkg/registry/register.go` (RegisterAll)
- The MCP stdio mode is typically used by Claude Code; SSE by web clients

## Known Pitfalls

- **SSE session memory leak**: SSE sessions may never expire. In a long-running server, abandoned SSE connections accumulate in memory. Check if there's a session timeout or cleanup mechanism.
- **Buffer overflow**: The SSE event buffer is likely capped at 100 events. When exceeded, the server returns 503 with no recovery path. The client must reconnect.
- **Stdio race condition**: When stdin reaches EOF (client disconnects), the MCP stdio handler may race between cleanup and in-flight tool calls.
- **`any` type encoding**: Go's `any` (interface{}) type can produce unexpected JSON when different tool outputs use different underlying types. Some may serialize as `null`, others as empty objects.
- **MCP version**: The protocol version in the handshake must match. Using an older version string may cause initialization failure.

## Key Code to Inspect

| File | What to Check |
|------|---------------|
| `pkg/gateway/mcp_adapter.go` | MCP server initialization, tool registration |
| `pkg/gateway/server.go` | SSE endpoint registration |
| `pkg/registry/register.go` | Which units are registered as MCP tools |
| `go.mod` | MCP library version (mark3labs/mcp-go) |

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Total number of tools returned by `tools/list`
- Whether SSE transport works at all
- Any encoding issues in tool call responses
