# Scenario 8: Full-Stack Integration -- All Interfaces

**Difficulty**: Hard
**Estimated Duration**: 20 minutes
**Prerequisites**: Scenarios 1, 2, and 4 completed (CLI, Docker, and Agent all working)

## User Story

"I want to verify that AIMA works through ALL its interfaces -- CLI, HTTP API, and the AI agent -- and they all see the same state. If I create a model via CLI, I should be able to see it via HTTP and manage it via the agent. Everything should be consistent."

## Success Criteria

1. [ ] Can start the AIMA HTTP server (`aima start --http :8080`)
2. [ ] Can create a model via CLI and verify it appears in the HTTP API (`curl`)
3. [ ] Can query the model list via HTTP API and get a valid JSON response
4. [ ] Can use the AI agent to list models (should see the same model)
5. [ ] Can use the AI agent to perform an operation (e.g., delete a model) and verify the result via CLI
6. [ ] All three interfaces (CLI, HTTP, Agent) share the same state within a single `aima start` session
7. [ ] Can stop the HTTP server cleanly

## Environment Setup

- Kimi API configured (same as Scenario 4)
- Ensure no port conflict on 8080: `ss -tlnp | grep 8080` (use `--http :8081` as fallback)
- The key insight: `aima start --http :8080` runs a long-lived server. CLI commands and agent chat within that same process share state. But running `aima model list` as a separate process has its own state (separate in-memory store).

To test state sharing, you need to either:
- Use the HTTP API (`curl`) to interact with the running server
- Or use the agent chat within the same server process

## Hints for the Operator

- `aima start --http :8080` -- start the HTTP server (runs in foreground)
- `curl http://localhost:8080/api/model/list` -- query models via HTTP
- `curl -X POST http://localhost:8080/api/model/create -d '...'` -- create model via HTTP
- `aima agent chat` -- agent within the same server (if supported), or via HTTP
- `aima model list` -- CLI (note: separate process = separate state)

The HTTP API routes follow the pattern `/api/{domain}/{action}`. For example:
- `/api/model/list`
- `/api/model/create`
- `/api/service/list`
- `/api/device/list`
- `/api/inference/chat`

## Known Pitfalls

- **State isolation**: If you run `aima start --http :8080` in one terminal and `aima model list` in another, they have SEPARATE state (each is a different process with its own in-memory store). To test state sharing, use the HTTP API or agent within the server process.
- **Port 8080 in use**: Another service may be using port 8080. Use `--http :8081` or any free port as a fallback.
- **MCP server mode**: `aima mcp` (or `aima start --mcp`) uses stdio for communication, designed for MCP clients. Testing this requires a proper MCP client, not curl.
- **HTTP content type**: API requests may need `Content-Type: application/json` header.
- **Agent via HTTP**: The agent chat may need to be invoked via the HTTP API endpoint rather than the CLI `aima agent chat` command (which starts its own process).
- **Bug #10**: Agent requests need long timeouts. When using the HTTP API for agent chat, ensure the HTTP client timeout is sufficient (10+ minutes).
- **SQLite vs memory**: Future versions may use SQLite for persistence, making state sharing across processes possible. Current version uses in-memory stores.

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Unexpected observations
