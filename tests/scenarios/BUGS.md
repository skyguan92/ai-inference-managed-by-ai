# AIMA Acceptance Test — Bug Tracker

## Bugs Found

### Bug #18: All device commands fail with EXECUTION_FAILED

- **Scenario**: 1 (Hello AIMA)
- **Severity**: High (P0)
- **Command**: `aima device detect`, `aima device info`, `aima device metrics`
- **Expected**: Device information for NVIDIA GB10 GPU
- **Actual**: `Error: EXECUTION_FAILED: command execution failed`
- **Root Cause**: `DeviceProvider` never wired in CLI. `pkg/cli/root.go` doesn't call `registry.WithDeviceProvider()`. HAL implementations exist at `pkg/infra/hal/nvidia/` and `pkg/infra/hal/generic/` but are never instantiated.
- **Status**: FIXED — DeviceProvider (nvidia HAL) wired in root.go via `registry.WithDeviceProvider()`

### Bug #19: `engine list` returns empty despite 3 loaded engine types

- **Scenario**: 1 (Hello AIMA)
- **Severity**: Medium (P1)
- **Command**: `aima engine list`
- **Expected**: List of available engine types (vllm, asr, tts)
- **Actual**: `items [] total 0`
- **Root Cause**: `HybridEngineProvider` loads YAML assets into memory but never populates the `EngineStore`. The `engine.list` query only checks the empty store.
- **Status**: FIXED — Added `AssetTypes()` method to HybridEngineProvider, root.go seeds EngineStore at startup

### Bug #20: Table output format renders raw JSON for nested arrays

- **Scenario**: 1 (Hello AIMA)
- **Severity**: Low (P2)
- **Command**: `aima model list --output table` (default)
- **Expected**: Formatted table with columns (ID, Name, Status, Type)
- **Actual**: `items [{"id":"model-3660ec50","name":"Qwen3-Coder-Next-FP8",...}]`
- **Root Cause**: `formatMapTable()` in `pkg/cli/output.go` JSON-marshals nested array values instead of delegating to `formatSliceTable()`.
- **Status**: FIXED — `formatMapTable()` now detects `items` key with slice value and delegates to `formatSliceTable()`

### Bug #21: `inference.chat` CLI fails — no real InferenceProvider wired

- **Scenario**: 2 (Zero to Inference)
- **Severity**: Critical (P0) — core inference path broken
- **Command**: `aima inference chat --model Qwen3-Coder-Next-FP8 --message "Write a Python hello world" --max-tokens 200`
- **Expected**: Code response from running vLLM service
- **Actual**: `Error: EXECUTION_FAILED: command execution failed`
- **Root Cause**: No real `InferenceProvider` implementation exists. Only `MockProvider` in `pkg/unit/inference/provider.go`. The CLI's `RegisterAll()` is never passed `WithInferenceProvider()`. A real provider needs to look up service endpoints and proxy OpenAI-compatible chat requests to running vLLM/Ollama.
- **Workaround**: `curl http://localhost:8000/v1/chat/completions` works directly against vLLM
- **Status**: FIXED — Created `ProxyInferenceProvider` in `pkg/infra/provider/inference_provider.go`, wired in root.go

### Bug #22: `model list` table output still shows "value" header with no data

- **Scenario**: 2 (Zero to Inference)
- **Severity**: Low (P2)
- **Command**: `aima model list` (default table output)
- **Expected**: Formatted table with columns (ID, Name, Status, Type)
- **Actual**: Shows "value" header but no model data
- **Root Cause**: Likely the Bug #20 fix wasn't synced to remote binary, or a different code path for model list response format
- **Status**: Investigating

### Bug #23: TTS engine YAML default_args incompatible with Docker image

- **Scenario**: 3 (Multimodal Stack)
- **Severity**: Low (P2) — environment config issue
- **Command**: `aima service start svc-tts-model-bb3fbcb3`
- **Expected**: TTS container starts and runs
- **Actual**: Container exits with `Error: No such option: --model`
- **Root Cause**: `catalog/engines/tts/qwen-tts-cpu.yaml` includes `--model /model` in `default_args` but the TTS Docker image's uvicorn doesn't accept `--model` flag
- **Status**: Open

### Bug #24: Service status stuck at "creating" after failed start

- **Scenario**: 3 (Multimodal Stack)
- **Severity**: Medium (P1) — state machine issue
- **Command**: `aima service list` after failed start attempts
- **Expected**: Status should be "failed" or "error" after all retries exhausted
- **Actual**: Status remains "creating" permanently
- **Root Cause**: Service lifecycle doesn't transition to a failure state when Docker start fails 3x and native fallback also fails
- **Status**: FIXED — Added `ServiceStatusFailed` status, service transitions to "failed" when retries exhausted

### Bug #25: `service stop` fails for services without running containers

- **Scenario**: 3 (Multimodal Stack)
- **Severity**: Medium (P1) — state machine issue
- **Command**: `aima service stop svc-whisper-model-1ce581fa`
- **Expected**: Gracefully transition service to "stopped" when no container exists
- **Actual**: `Error: EXECUTION_FAILED: command execution failed`
- **Root Cause**: `service.stop` handler assumes a running Docker container exists. Should handle "nothing to stop" case gracefully.
- **Status**: FIXED — `service stop` now handles non-running states (creating/failed → stopped, already stopped → success)

### Bug #27: Container not stopped when health check times out due to gateway timeout

- **Scenario**: 5 (Agent Self-Service)
- **Severity**: Critical (P0) — causes cascading failures
- **Trigger**: Gateway 30s timeout kills `service.start` request while vLLM is loading (needs 7 min)
- **Expected**: Container should be stopped/cleaned up when the request context is cancelled
- **Actual**: Docker container keeps running, blocks port 8000 for all subsequent start attempts. Creates orphaned containers (10 found after one failed agent conversation)
- **Root Cause**: The health check waiter doesn't have a cleanup defer for context cancellation. When gateway timeout fires, the goroutine running the health check is cancelled but the container is never stopped.
- **Impact**: Cascading failures — every subsequent `service.start` fails with "port already allocated"
- **Status**: FIXED — `waitForHealth()` now cleans up container on context cancellation using `context.Background()` for cleanup ops

### Bug #28: Agent MCP tool doesn't expose timeout parameter for service.start

- **Scenario**: 5 (Agent Self-Service)
- **Severity**: Medium (P1)
- **Expected**: Agent should be able to pass `timeout: 600` to the service.start tool call
- **Actual**: MCP tool schema for `service.start` has no timeout parameter. Gateway uses default 30s timeout.
- **Root Cause**: Bug #17 fixed timeout propagation for CLI `--timeout` flag, but the MCP tool path doesn't expose this. The `service.start` InputSchema doesn't include a timeout field.
- **Status**: FIXED — Added `timeout` field (1-3600s) to service.start InputSchema, creates context.WithTimeout in Execute()

### Bug #29: Agent max tool call rounds (10) too low for deploy-and-test flows

- **Scenario**: 5 (Agent Self-Service)
- **Severity**: Low (P2)
- **Expected**: Agent should have enough rounds for: create + start + poll status + inference = 4+ calls
- **Actual**: With retries after failures, 10 rounds is insufficient
- **Root Cause**: Hardcoded limit in agent conversation loop
- **Status**: Open

### Bug #30: Docker retry logic doesn't clean up failed containers before retrying

- **Scenario**: 6 (Lifecycle Stress)
- **Severity**: Critical (P0) — makes restart cycles impossible
- **Trigger**: `ContainerStart` fails (e.g., port conflict, crash), retry creates NEW container without removing old one
- **Expected**: Failed containers should be removed before retry; port should be freed
- **Actual**: Each retry creates another orphaned "Created" container. Port stays blocked by Docker proxy.
- **Root Cause**: In `hybrid_engine_provider.go` retry loop: after `ContainerStart` fails, the created container is never removed. Retries compound the problem.
- **Impact**: 12 orphaned containers after 2 start attempts. Restart cycles completely broken.
- **Status**: FIXED — Pre-start cleanup of stale containers by label, container removal on start failure, ListContainers now finds all states

### Bug #31: Port not released after container stop/crash

- **Scenario**: 6 (Lifecycle Stress)
- **Severity**: Medium (P1)
- **Trigger**: Container crashes or is stopped, Docker doesn't immediately release port
- **Expected**: Wait for port availability or verify before creating new container
- **Actual**: New container creation immediately fails with "port already allocated"
- **Root Cause**: Docker proxy may hold port briefly after container stop. No wait/verify logic.
- **Status**: Open

### Bug #26: Agent config values from config.toml [agent] section not properly loaded

- **Scenario**: 4 (AI Operator)
- **Severity**: Medium (P1) — usability issue
- **Expected**: Agent reads `llm_base_url`, `llm_model`, `llm_api_key` from `[agent]` section of `~/.aima/config.toml`
- **Actual**: Binary ignores config file values, defaults to `api.openai.com` and `moonshot-v1-8k`. All settings must be overridden via env vars (`AIMA_LLM_API_KEY`, `OPENAI_MODEL`, `OPENAI_BASE_URL`, `OPENAI_USER_AGENT`).
- **Root Cause**: Likely in config loading logic — `pkg/config/config.go` or env var override precedence in `setupAgent()`
- **Status**: Open

## Scenario 7 Re-Test Findings (Post Bug #30 Fix)

Scenario 6 (Lifecycle Stress) now **PASSES all 5/5 criteria** after the Bug #30 fix — 5 consecutive start/stop cycles work cleanly with no orphan containers.

However, Scenario 7 (Recovery from Chaos) still **FAILS 0/4** — the pre-start cleanup logic only handles AIMA's own stale containers from previous sessions. It does NOT handle:

1. **Externally-created orphans**: A container created with `docker run --label aima.engine=vllm -p 8000:8000 ...` is not detected or cleaned before engine start. AIMA blindly tries to bind port 8000, fails with "port already allocated", and retries 5 times (each creating another orphan).

2. **Retry containers accumulate**: Each retry creates a new "Created" container that also fails to start. After 5 retries, there are 5+1 orphaned containers.

3. **Corrupted state after failed start**: When all retries fail, the service state is inconsistent. `service stop` and `service stop --force` both fail with `EXECUTION_FAILED`. Stop should attempt label-based Docker cleanup regardless of service state.

4. **Recovery requires manual `docker rm -f`**: No AIMA command can recover from this state without manual Docker intervention.

**Recommended fix**: Pre-start cleanup should check port availability (or do a broader label+port scan) before creating a container, not just look for AIMA-managed containers in the in-memory map.

## Observations (Not Bugs)

### Obs #1: Startup logging noise
Every CLI command prints 4-5 lines of INFO logs (SQLite init, engine provider, agent detection) to stderr before showing results. Consider suppressing unless `--verbose` is used.

### Obs #2: `aima device list` silently shows help
Running `aima device list` (nonexistent subcommand) shows the `device` help page without error. Could confuse users expecting a list of devices.

### Obs #3: Stale service states in DB
2 services show "creating" status — likely stale from previous sessions. Should have been cleaned up or timed out.

### Obs #4: Version shows "dev/unknown/unknown"
`aima version` shows no build-time info. Functional but not polished.

### Obs #5: Container stop always uses label fallback path
Every `service stop` invocation finds containers via Docker label lookup (not the in-memory map), because the in-memory container map is not persisted across CLI invocations. This works correctly but is slightly inefficient — each stop requires a Docker API call. Not a bug, but worth noting for future optimization.

### Bug #32: SQLite store fails with CANTOPEN when data directory doesn't exist

- **Scenario**: 9 (Config & Persistence Resilience)
- **Severity**: Medium (P1) — prevents custom data_dir from using SQLite
- **Command**: `aima --config /tmp/aima-custom.toml model list` where config has `[general].data_dir = "/tmp/aima-test-data"`
- **Expected**: Custom directory created, SQLite database initialized there
- **Actual**: SQLite fails with "unable to open database file: out of memory (SQLITE_CANTOPEN=14)" because parent directory doesn't exist when SQLite tries to create the DB file; falls back to file store
- **Root Cause**: `NewSQLiteStore` in `pkg/infra/store/sqlite_store.go` calls `sql.Open("sqlite", dbPath)` without first creating the parent directory via `os.MkdirAll`
- **Status**: FIXED — Added `os.MkdirAll(filepath.Dir(dbPath), 0755)` before `sql.Open` in NewSQLiteStore. Commit: e91955d

### Bug #33 (Doc): Scenario 9 uses wrong TOML section for data_dir

- **Scenario**: 9 (Config & Persistence Resilience)
- **Severity**: Low (P2) — documentation bug only
- **Problem**: Test 4 in scenario file uses `[storage]\ndata_dir = ...` but the actual config struct has `data_dir` under `[general]`
- **Status**: FIXED — Updated scenario 9 test script to use `[general]` section. Commit: e91955d

### Bug #34: skill.add rejects flat JSON input — requires YAML front-matter format

- **Scenario**: 12 (Skill Registry and Search)
- **Severity**: High (P1) — user-facing API unusable for normal JSON clients
- **Command**: `POST /api/v2/execute` with `unit: "skill.add"` and flat JSON fields (name, description, keywords, category, content)
- **Expected**: Skill created from JSON fields
- **Actual**: `Error: parse skill: [01202] skill is invalid` — `AddCommand.Execute()` called `ParseSkillFile(content)` which requires YAML front-matter (`---\nid: ...\n---`) and rejects plain content
- **Root Cause**: `AddCommand.Execute()` in `pkg/unit/skill/commands.go` only accepted YAML front-matter format. Scenario test sends flat JSON fields as separate top-level keys.
- **Status**: FIXED — `AddCommand.Execute()` now supports both formats: flat JSON fields (name, description, keywords, category, content) and YAML front-matter. Also added `buildSkillFromFields()` helper. Commit: bfcc18c

### Bug #35: skill.search returns disabled skills

- **Scenario**: 12 (Skill Registry and Search)
- **Severity**: Medium (P1) — wrong search results undermine skill filtering
- **Command**: Search after disabling a skill
- **Expected**: Disabled skills excluded from search results
- **Actual**: `MemoryStore.Search()` returned all matching skills regardless of `enabled` flag
- **Root Cause**: `Search()` in `pkg/unit/skill/store.go` did not check `sk.Enabled` — unlike `List()` which has an `EnabledOnly` filter option, `Search()` always included disabled skills
- **Status**: FIXED — Added `if !sk.Enabled { continue }` check at start of `Search()` loop. Commit: bfcc18c

### Bug #36 (Arch): HTTP REST routes from routes.go not wired into HTTP server

- **Scenario**: 12 (Skill Registry and Search) — also affects all scenarios
- **Severity**: High (P1) — REST API endpoints like `/api/v2/skills` return 404
- **Command**: `GET /api/v2/skills`, `POST /api/v2/skills`, any of the 50+ routes in `pkg/gateway/routes.go`
- **Expected**: Routes registered and serving requests
- **Actual**: All return `404 page not found` — the `Router` and `defaultRoutes()` in `pkg/gateway/routes.go` are defined but never registered in the HTTP server
- **Root Cause**: `pkg/cli/start.go` `runStart()` creates `http.NewServeMux()` with only 3 handlers (`/api/v2/execute`, `/api/v2/health`, `/api/v2/metrics`). The `gateway.NewRouter()` with 50+ domain routes is never mounted.
- **Workaround**: Use `POST /api/v2/execute` with `{type, unit, input}` generic envelope — all operations work via this endpoint
- **Status**: Open — The REST routes exist but are not wired. Functionality is accessible via the generic execute endpoint.

### Bug #37: Pipeline status never resets to idle after run completes or fails

- **Scenario**: 10 (Pipeline DAG Execution)
- **Severity**: High (P1) — pipelines permanently stuck in "running" state, preventing deletion
- **Command**: `pipeline.run` followed by `pipeline.get` or `pipeline.list`
- **Expected**: After a run completes or fails, pipeline status resets to "idle"
- **Actual**: Pipeline status remains "running" indefinitely after all runs finish; `pipeline.delete` returns "pipeline is running" even after runs complete
- **Root Cause**: `executor.go` goroutine transitions the `PipelineRun` status (to completed/failed/cancelled) but never updates the parent `Pipeline.Status` back to idle. Only `CancelCommand` reset pipeline status to idle.
- **Status**: FIXED — Added `finishRun()` helper in executor that atomically updates run status AND resets pipeline to idle at all exit paths. Commit: 50b370b

### Bug #38: CancelCommand returns error for already-failed runs

- **Scenario**: 10 (Pipeline DAG Execution)
- **Severity**: Low (P2) — cancel should be idempotent for all terminal states
- **Command**: `pipeline.cancel` on a run that has already failed
- **Expected**: Cancel returns `{"success": true}` (idempotent, already terminal)
- **Actual**: Returns `{"error": "run not cancellable"}` — `RunStatusFailed` hit the `default` branch in the switch statement
- **Root Cause**: `CancelCommand.Execute()` in `commands.go` treated only `RunStatusCompleted` and `RunStatusCancelled` as terminal no-op states; `RunStatusFailed` was sent to the `default` branch which returned `ErrRunNotCancellable`
- **Status**: FIXED — Added `RunStatusFailed` to the terminal-state case in the cancel switch. Commit: d07f5d2

### Bug #39: REST API routes never mounted — all domain endpoints return 404

- **Scenario**: 11 (Catalog Recipe Matching) — discovered root cause of Bug #36
- **Severity**: Critical (P0) — ALL REST domain endpoints broken in production server
- **Endpoints affected**: All `/api/v2/{domain}/...` routes (catalog, skill, agent, model, service, engine, etc.)
- **Expected**: POST `/api/v2/catalog/recipes` creates a recipe (200)
- **Actual**: 404 for every domain endpoint; only `/api/v2/execute`, `/api/v2/health`, `/api/v2/metrics` worked
- **Root Cause**: `pkg/cli/start.go` builds its own `http.ServeMux` with only 3 hardcoded routes, never calling `gateway.NewRouter(gw)`. The `gateway.Server` (which DOES mount the router) is never used in production. 200+ routes in `gateway/routes.go` were completely unreachable.
- **Status**: FIXED — Added `router := gateway.NewRouter(gw)` and `mux.Handle("/api/v2/", router)` in `runStart()`. Commit: 044313d

### Bug #40: `catalog.match` silently ignores `vram_gb` from GET query string

- **Scenario**: 11 (Catalog Recipe Matching) — HIGH CONFIDENCE predicted bug confirmed
- **Severity**: Medium (P1) — VRAM filtering broken for GET requests
- **Command**: `GET /api/v2/catalog/recipes/match?vram_gb=24`
- **Expected**: `vram_gb` parsed as integer 24 for VRAM scoring
- **Actual**: `vram_gb` arrives as string `"24"` from `queryInputMapper`; `toInt()` had no `string` case, silently returned `0`, VRAM score always 0 and no filtering applied
- **Root Cause**: `toInt()` in `commands.go` handled `int`, `int32`, `int64`, `float64`, `float32` but NOT `string`
- **Status**: FIXED — Added `string` case using `fmt.Sscanf(val, "%d", &n)`. Commit: 6b68a0f

### Bug #41: `scoreRecipe` scores NVIDIA recipes on AMD queries (vendor mismatch not filtered)

- **Scenario**: 11 (Catalog Recipe Matching)
- **Severity**: Medium (P1) — wrong recipes shown for vendor-specific queries
- **Command**: `GET /api/v2/catalog/recipes/match?gpu_vendor=AMD`
- **Expected**: NVIDIA-specific recipes excluded (score=0)
- **Actual**: NVIDIA recipes appeared with OS+VRAM scores (e.g. score=15) despite vendor mismatch
- **Root Cause**: `scoreRecipe()` only skipped the vendor bonus if vendor didn't match, but didn't exclude the recipe entirely
- **Status**: FIXED — Added hard filter: if both recipe and query specify `gpu_vendor` and they differ, return 0. Commit: 6b68a0f

### Bug #42: `scoreRecipe` includes 48GB-min recipes in 24GB queries (VRAM boundary not filtered)

- **Scenario**: 11 (Catalog Recipe Matching)
- **Severity**: Medium (P1) — incompatible recipes shown to users
- **Command**: `GET /api/v2/catalog/recipes/match?vram_gb=24`
- **Expected**: Recipe requiring `min_vram_gb=48` excluded from 24GB query
- **Actual**: Recipe appeared in results with reduced score (missing +10 bonus but still score=45 from vendor+OS)
- **Root Cause**: `scoreRecipe()` skipped the VRAM bonus but did not exclude the recipe when query VRAM < recipe minimum
- **Status**: FIXED — Added hard filter: if `recipe.profile.vram_min_gb > 0 && query.vram_gb > 0 && query.vram_gb < recipe.vram_min_gb`, return 0. Commit: 6b68a0f

---

## Scenario 13: HTTP REST API Completeness

### Bug #43: `bodyInputMapper` silently drops JSON decode errors — returns `{}` instead of 400

- **Scenario**: 13 (HTTP REST API Completeness)
- **Severity**: High (P1) — malformed JSON gets silently accepted; clients get confusing 500 errors instead of 400
- **Command**: `POST /api/v2/models/create` with `Content-Type: application/json` and malformed body `{"this is not valid json`
- **Expected**: HTTP 400 with `{"error": {"code": "invalid_request", "message": "..."}}`
- **Actual**: HTTP 500 with `EXECUTION_FAILED: name is required` — the bad JSON was silently swallowed as empty `{}` and the downstream validator raised a different error
- **Root Cause**: `bodyInputMapper` in `pkg/gateway/routes.go:219` returns empty `map[string]any{}` on JSON decode error (`err != nil`), never signals the error to the caller. The router's `handleRoute` then calls the unit with an empty input map.
- **Evidence**: `{"this is not valid json` → HTTP 500 `EXECUTION_FAILED: name is required` (not 400 invalid JSON)
- **Status**: Open — fix requires `bodyInputMapper` to return an error (change signature) or the router to detect and reject decode failures with HTTP 400

### Bug #44: Domain unit errors always return HTTP 500 — no semantic HTTP status mapping

- **Scenario**: 13 (HTTP REST API Completeness)
- **Severity**: Medium (P1) — API clients can't distinguish validation errors from server errors
- **Commands affected**: Any domain endpoint with validation errors (missing required fields, invalid input)
- **Expected**: `inference.chat` with empty body → HTTP 400 (bad request); `resource.status` with no provider → HTTP 503 (service unavailable)
- **Actual**: All unit errors return HTTP 500 regardless of error type
- **Root Cause**: `handleRoute` in `pkg/gateway/routes.go` always calls `w.WriteHeader(http.StatusInternalServerError)` when `!resp.Success`. Error codes like `[00009] invalid input` or `EXECUTION_FAILED` are not mapped to appropriate HTTP status codes (400, 404, 503, etc.)
- **Evidence**:
  - `inference.chat {}` → HTTP 500 `model not specified` (should be 400)
  - `resource.status` no provider → HTTP 500 (should be 503)
  - `model.create` with bad JSON → HTTP 500 (should be 400)
- **Status**: Open

### Bug #45: No MaxBytesReader on REST domain routes — 10MB+ payloads accepted

- **Scenario**: 13 (HTTP REST API Completeness)
- **Severity**: Medium (P1) — potential OOM/DoS vector
- **Command**: `POST /api/v2/models/create` with 10MB body
- **Expected**: HTTP 413 (Payload Too Large)
- **Actual**: Request accepted; returns 500 with validation error (body was read into memory)
- **Root Cause**: `pkg/cli/start.go` applies `http.MaxBytesReader(w, r.Body, 10<<20)` only inside `handleExecute()` for `/api/v2/execute`. The `gateway.NewRouter(gw)` handler (`ServeHTTP` in `routes.go`) has no MaxBytesReader — every domain REST route is vulnerable.
- **Status**: Open — fix: add `r.Body = http.MaxBytesReader(w, r.Body, 10<<20)` at the start of `handleRoute()` or in a middleware wrapping the router

### Bug #46: CORS not wired — OPTIONS requests return 404

- **Scenario**: 13 (HTTP REST API Completeness)
- **Severity**: Medium (P1) — web dashboard clients cannot make cross-origin requests
- **Command**: `OPTIONS /api/v2/models` with Origin header
- **Expected**: HTTP 200 with `Access-Control-Allow-Origin`, `Access-Control-Allow-Methods` headers
- **Actual**: HTTP 404 `{"error": {"code": "UNIT_NOT_FOUND", "message": "route not found: OPTIONS /api/v2/models"}}`
- **Root Cause**: No CORS middleware is registered. The `Router.ServeHTTP` treats OPTIONS as a regular method and fails to find a matching route. No CORS middleware file was found in `pkg/gateway/middleware/`.
- **Status**: Open

### Bug #47: HEAD requests return 404 on REST routes

- **Scenario**: 13 (HTTP REST API Completeness)
- **Severity**: Low (P2) — HEAD should work on any GET endpoint per HTTP spec
- **Command**: `HEAD /api/v2/models`
- **Expected**: HTTP 200 with headers, no body (mirrors GET /api/v2/models)
- **Actual**: HTTP 404 — router only finds exact method matches, no HEAD→GET fallback
- **Root Cause**: `Router.ServeHTTP` in `pkg/gateway/routes.go` does exact method matching; no fallback to respond to HEAD with a GET handler
- **Status**: Open

---

## Scenario 14: MCP Protocol Compliance

### Bug #48: MCP stdio goroutines can exit before writing response (race at stdin EOF)

- **Scenario**: 14 (MCP Protocol Compliance)
- **Severity**: High (P1) — parse errors and other single-message responses silently dropped
- **Command**: `echo 'this is not json' | aima mcp serve`
- **Expected**: JSONRPC parse error response `{"jsonrpc":"2.0","error":{"code":-32700,"message":"parse error:..."}}`
- **Actual**: No output — the `Serve()` loop returns (stdin EOF) before the goroutine for the malformed message writes to stdout
- **Root Cause**: `MCPServer.Serve()` in `pkg/gateway/mcp_server.go` spawns goroutines via `go func() { s.handleLine(...) }()` then returns `nil` when scanner reaches EOF. The goroutines may not have finished writing yet. `s.wg.Wait()` was only called in `Shutdown()`, not at the end of `Serve()`.
- **Status**: FIXED — Added `s.wg.Wait()` at all three return paths in `Serve()` (context cancelled, scanner error, EOF). Commit: 9d01a11

### Bug #49: Tool not found returns wrong JSONRPC error code (-32002 instead of -32601)

- **Scenario**: 14 (MCP Protocol Compliance)
- **Severity**: Medium (P1) — MCP protocol violation; clients expecting -32601 for unknown tools
- **Command**: `tools/call` with `name: "nonexistent.tool"`
- **Expected**: JSONRPC error with code `-32601` (Method not found) per scenario spec
- **Actual**: Error code `-32002` (MCPErrorCodeToolExecution) — `handleToolsCall` used the tool-execution error code for a tool-lookup failure
- **Root Cause**: `ExecuteTool()` in `pkg/gateway/mcp_tools.go` returned a plain `fmt.Errorf("tool not found: %s", name)`; `handleToolsCall` wrapped it with `MCPErrorCodeToolExecution` (-32002). The tool-not-found case should return `MCPErrorCodeMethodNotFound` (-32601) per JSONRPC spec.
- **Status**: FIXED — `ExecuteTool()` now returns `&MCPError{Code: MCPErrorCodeMethodNotFound, ...}`; `handleToolsCall` uses `errors.As` to detect `*MCPError` and use its code directly. Commit: 9d01a11

### Bug #48: Method Not Allowed returns 404 instead of 405

- **Scenario**: 13 (HTTP REST API Completeness)
- **Severity**: Low (P2) — incorrect HTTP semantics
- **Command**: `DELETE /api/v2/models` (only GET is defined for this path)
- **Expected**: HTTP 405 Method Not Allowed with `Allow: GET` header
- **Actual**: HTTP 404 `route not found: DELETE /api/v2/models`
- **Root Cause**: `Router.ServeHTTP` does not distinguish "path exists but wrong method" from "path not found"; both return 404
- **Status**: Open

### Bug #49: Resource domain has no ResourceProvider wired — resource.status always fails

- **Scenario**: 13 (HTTP REST API Completeness)
- **Severity**: Medium (P1) — resource endpoint unusable
- **Command**: `GET /api/v2/resource/status`
- **Expected**: Resource usage data (CPU, GPU, memory)
- **Actual**: HTTP 500 `{"error": {"code": "EXECUTION_FAILED", "details": "[00008] resource provider not set"}}`
- **Root Cause**: `RegisterAll()` does not wire a `ResourceProvider`. The resource domain exists but no provider implementation is registered.
- **Status**: Open

---

## Scenario 15: Auth Middleware Wiring

### Bug #50 (Arch): Auth middleware not wired into production HTTP server (start.go uses plain mux)

- **Scenario**: 15 (Auth Middleware Wiring)
- **Severity**: Critical (P0) — all auth enforcement is bypassed in production
- **Evidence**: `POST /api/v2/models/create` without any token → HTTP 200 (should require auth for commands)
- **Root Cause**: `pkg/cli/start.go` `runStart()` builds a plain `http.NewServeMux()` and wraps handlers with only `instrumentHandler`. The `gateway.Server` (`pkg/gateway/server.go`) has complete auth middleware in `buildHandler()` (lines 120–124), but `gateway.Server` is never instantiated in production — only in tests. `start.go` never calls `middleware.Auth()`.
- **Impact**: ALL endpoints bypass auth. `Forced` routes (remote.exec, model.delete) that should ALWAYS require a token are completely unprotected.
- **Code to fix**: `pkg/cli/start.go` — wrap the router with `middleware.Auth(authCfg)` after constructing the mux.
- **Status**: Open

### Bug #51: No TOML or env-var mechanism to enable auth or supply API keys

- **Scenario**: 15 (Auth Middleware Wiring)
- **Severity**: High (P1) — even if auth middleware were wired, there is no way for operators to enable it
- **Evidence**:
  - `AIMA_CONFIG=/tmp/aima-auth.toml /tmp/aima start` with `[auth] enabled=true api_keys=[...]` → server ignores the section, all requests still 200
  - `AIMA_AUTH_ENABLED=true AIMA_API_KEYS="..."` → no such env vars processed in `ApplyEnvOverrides()`
- **Root Cause**: `pkg/config/config.go` has `SecurityConfig` with only a single `api_key` string and `rate_limit_per_min` integer. There is no `AuthConfig` struct with `enabled bool` and `api_keys []string`. `ApplyEnvOverrides()` has no `AIMA_AUTH_ENABLED` or `AIMA_API_KEYS` handling. The `[auth]` TOML section is silently ignored.
- **Code to fix**: Add `AuthConfig` struct to `config.go` with `enabled`, `api_keys` fields; add `[auth]` TOML tag; add env var handling; wire it into the auth middleware in `start.go`.
- **Status**: Open

### Bug #52: `remote.exec` (Forced auth level) not protected — returns 500 instead of 401

- **Scenario**: 15 (Auth Middleware Wiring)
- **Severity**: Critical (P0) — highest-risk endpoint is completely unprotected
- **Evidence**: `POST /api/v2/remote/exec -d '{"command":"echo test"}'` without any token → HTTP 500 (remote provider not set), not 401
- **Root Cause**: Consequence of Bug #50 — auth middleware never runs, so `AuthLevelForced` for `remote.exec` is never evaluated. If auth middleware were wired, `remote.exec` would still fail because no API keys are configured (Bug #51), making it permanently inaccessible.
- **Note**: The 500 error is from `remote provider not set` (RemoteProvider not wired), not from auth rejection. Both bugs compound.
- **Status**: Open (blocked by Bug #50 and Bug #51)

### Bug #53: TokenBucketLimiter code exists but is never instantiated as middleware

- **Scenario**: 15 (Auth Middleware Wiring)
- **Severity**: Low (P2) — rate limiting is effectively disabled
- **Evidence**: 50 rapid requests to `GET /api/v2/models` all return HTTP 200 with no throttling
- **Root Cause**: `pkg/infra/ratelimit/ratelimit.go` implements `TokenBucketLimiter` with `Allow()` and `Reset()` methods, and tests exist in `ratelimit_test.go`. However, it is never instantiated or registered as HTTP middleware anywhere. `SecurityConfig.RateLimitPerMin` (config field) is read but never used to create a limiter.
- **Code to fix**: Create a rate limiter middleware in `pkg/gateway/middleware/`, instantiate `TokenBucketLimiter` in `start.go` using `cfg.Security.RateLimitPerMin`, and register it in the handler chain.
- **Status**: Open

### Bug #54: `service logs` fails — CLI sends "service.logs" but query is registered as "app.logs"

- **Scenario**: 18 (Service Monitoring & Events)
- **Severity**: Medium (P1) — `aima service logs` completely non-functional
- **Command**: `aima service logs svc-vllm-model-918aaddf`
- **Expected**: Container log output
- **Actual**: `Error: UNIT_NOT_FOUND: query not found: service.logs`
- **Root Cause**: In `pkg/cli/service.go:444`, `runServiceLogs()` dispatches `Unit: "service.logs"`. However, the actual `LogsQuery` in `pkg/unit/app/queries.go:311` has `Name() = "app.logs"`. The CLI unit name is mismatched with the registered query name.
- **Code to fix**: Added `service.LogsQuery` to `pkg/unit/service/queries.go` with `service_id` input. Added `GetLogs()` to `ServiceProvider` interface with `HybridServiceProvider` implementation via Docker label lookup (`aima.engine` label). Registered in registry. CLI already dispatches `"service.logs"` correctly.
- **Status**: FIXED — commit b8fda4c + 7789c99. Verified: `aima service logs svc-vllm-*` returns actual container output.

### Bug #55: EventBus not passed to RegisterAll — domain event delivery is silently skipped

- **Scenario**: 18 (Service Monitoring & Events)
- **Severity**: Medium (P1) — events from commands never delivered to subscribers
- **Evidence**: `pkg/cli/root.go:150` creates `eventBus` and sets it on `hep` (HybridEngineProvider) via `hep.SetEventBus(bus)`. However, the `RegisterAll()` call at line 188 does NOT include `registry.WithEventBus(bus)`. Thus all domain command handlers receive `nil` for `events`, and every `events.Publish(...)` call is a no-op.
- **Root Cause**: `WithEventBus()` option exists in `pkg/registry/register.go:69` but is never passed in the CLI invocation of `RegisterAll()`.
- **Code to fix**: Added `eventbus.EventPublisherAdapter` in `pkg/infra/eventbus/eventbus.go` bridging `eventbus.EventBus` → `unit.EventPublisher`. Passed via `registry.WithEventBus(eventbus.NewEventPublisherAdapter(r.eventBus))` in `pkg/cli/root.go`.
- **Note**: `StartProgressEvent` (Phase 9.3) still works via `hep.SetEventBus(bus)` direct path. Now all 10 domain registries also get EventBus.
- **Status**: FIXED — commit b8fda4c

### Bug #56: HTTP `GET /api/v2/services/status?service_id=X` returns "service not found"

- **Scenario**: 18 (Service Monitoring & Events)
- **Severity**: Low (P2) — query-param based status endpoint broken
- **Evidence**: `curl "http://localhost:9090/api/v2/services/status?service_id=svc-vllm-model-918aaddf"` returns `{"error":{"code":"EXECUTION_FAILED","details":"get service status: [00600] service not found"}}`. But `GET /api/v2/services/svc-vllm-model-918aaddf` works correctly.
- **Root Cause**: The `?service_id=` query parameter routing is likely mapping to a different query handler or input mapper than the path-param route.
- **Code to fix**: Investigate the HTTP route definition for `service.status` query and ensure the `service_id` input mapper reads from the correct source.
- **Status**: Open

### Bug #57: Container name conflict across retry attempts causes silent cascading failures

- **Scenario**: 18 (Service Monitoring & Events)
- **Severity**: Medium (P1) — retry loop degrades instead of recovering
- **Evidence**: During `service start`, retry attempts 4 and 5 fail with "Conflict. The container name /aima-vllm-XXXXXXXXXX is already in use". The orphan-cleanup in `FindContainersByPort` only finds containers by port binding, but stopping a container takes time and leaves it in "removing" state. The next attempt starts a new container (different random suffix) while the old one is still being removed, causing Docker to report a different name conflict.
- **Root Cause**: Container removal is async; the code calls `docker rm` but doesn't wait for full removal before starting a new container. Also, different retry timestamps generate different container names (random suffix based on timestamp), so the port-based cleanup removes one name but a different named container appears.
- **Code to fix**: After stopping a container, wait for its removal before proceeding to start a new one. Consider using `docker wait` or polling container state.
- **Status**: Open

### Bug #58: inference.chat CLI fails with "messages are required" when called from CLI

- **Scenario**: 17 (Inference Proxy Routing)
- **Severity**: High (P0) — `aima inference chat` completely non-functional via CLI
- **Command**: `aima inference chat --model smollm2:1.7b --message "hello"`
- **Expected**: Chat response from the model
- **Actual**: `Error: EXECUTION_FAILED: command execution failed` / `Error: chat completion failed: command execution failed`
- **Root Cause**: `pkg/cli/inference.go` passes `messages` as `[]map[string]string` (Go typed slice) to the gateway. But `ChatCommand.Execute()` in `pkg/unit/inference/commands.go` does `inputMap["messages"].([]any)` — this type assertion always fails because `[]map[string]string` is not `[]any`. The HTTP path works because JSON unmarshaling always produces `[]any` for arrays.
- **Status**: FIXED — Added `parseMessages()` helper that handles `[]any`, `[]map[string]string`, and `[]map[string]any` via type switch. Commit: ccf2abf

### Bug #59: inference.chat CLI shows generic error, not specific failure reason

- **Scenario**: 17 (Inference Proxy Routing)
- **Severity**: Medium (P1) — operator cannot diagnose why inference failed
- **Command**: `aima inference chat --model nonexistent --message "hello"`
- **Expected**: Clear error like "no running services found for model nonexistent"
- **Actual**: `Error: EXECUTION_FAILED: command execution failed` (no detail on what failed)
- **Root Cause**: `runInferenceChat()` in `pkg/cli/inference.go` printed only `resp.Error.Message` ("command execution failed"), not `resp.Error.Details` which contains the actual cause.
- **Status**: FIXED — Now shows `resp.Error.Details` when present. Commit: ccf2abf

### Bug #60 (Doc): HTTP inference/chat API uses `messages` array, not `message` string

- **Scenario**: 17 (Inference Proxy Routing)
- **Severity**: Low (P2) — documentation/test expectation mismatch
- **Problem**: S17 test script used `"message": "..."` (singular string) in the HTTP body, but `inference.chat` command handler requires `"messages": [...]` (array of message objects). The test script sends the wrong format.
- **Status**: Known — the HTTP API requires the messages array format. The CLI `--message` flag is automatically converted to the array format before calling the gateway.

---

## Scenario 16: Multi-Model Concurrent Deployment

### Bug #61: Both services default to port 8000 — second service kills first

- **Scenario**: 16 (Multi-Model Concurrent Deployment)
- **Severity**: Critical (P0) — multi-service deployment completely broken
- **Command**: Create service A (assigned port 8000), then start service B → service B also gets port 8000 → Phase 2 stale cleanup removes service A's running container
- **Expected**: Each service uses its uniquely-assigned port from creation time
- **Root Cause (2-part)**:
  1. `StartAsync()` in `hybrid_engine_provider.go` calls `hybridProvider.Start(engineType, ...)` without reading the service's stored port from the SQLite config. `getDefaultPort("vllm")` always returns 8000.
  2. `HybridEngineProvider.Start()` did not accept a port override from the config map.
- **Fix**: Added port override in `HybridEngineProvider.Start()` reading `config["port"]`; `StartAsync()` reads stored port from `serviceStore.Get()` and passes via config. Commit: 3201d56
- **Status**: FIXED

### Bug #62: Service Config (port + engine settings) not persisted to SQLite on create

- **Scenario**: 16 (Multi-Model Concurrent Deployment)
- **Severity**: Critical (P0) — blocks Bug #61 fix; port assignment lost between sessions
- **Root Cause (2-part)**:
  1. `service.create` command in `commands.go` did not copy `result.Config` and `result.Endpoints` from the provider's `CreateResult` to the `ModelService` saved to the store.
  2. `ServiceSQLiteStore.Create()` hardcoded `""` for the config column instead of JSON-serializing `svc.Config`. Also `Get()`, `GetByName()`, `List()`, `Update()` did not read/write the config column.
- **Fix**: Fixed `CreateCommand.Execute()` to copy `Config` and `Endpoints`; fixed all SQLite store CRUD methods to properly serialize/deserialize Config as JSON. Commits: 6e97d31, f976f3f
- **Status**: FIXED

### Bug #63: `ListContainers` returns running containers — Phase 2 cleanup kills active services

- **Scenario**: 16 (Multi-Model Concurrent Deployment)
- **Severity**: Critical (P0) — any service start attempt removes other running services
- **Root Cause**: `SDKClient.ListContainers()` was called in Phase 2 pre-start cleanup to find stale containers for the same engine type. It returned ALL aima-labeled containers including running ones. This caused Service A's running container to be removed when Service B attempted to start.
- **Fix**: Added state filter in `ListContainers()` — skip containers with `ct.State == "running"` or `ct.State == "restarting"`. Commit: de791ee
- **Status**: FIXED

### Bug #64: Stop uses default port (8000) in fallback — kills wrong service

- **Scenario**: 16 (Multi-Model Concurrent Deployment)
- **Severity**: Critical (P0) — stopping any service kills Service A (port 8000)
- **Root Cause**: `HybridEngineProvider.Stop()` had a port-based fallback that called `p.getDefaultPort(engineType)` (always 8000 for vLLM) when the label-based container search returned empty. When stopping Service B (port 8001, no running container), the fallback found and removed Service A's container at port 8000.
- **Fix**: Removed the default-port fallback from `HybridEngineProvider.Stop()`. Moved port-based cleanup to `HybridServiceProvider.Stop()`, which reads the service's actual port from the store and targets only the correct container. Commit: 0a85c5d
- **Status**: FIXED

### Observation: GPU OOM with two vLLM services on NVIDIA GB10

- **Scenario**: 16 (Multi-Model Concurrent Deployment)
- **Hardware**: NVIDIA GB10 (ARM64, unified LPDDR memory ~136GB shared)
- **Observation**: Service A (GLM-4.7-Flash, ~75GB model weights + KV cache) uses most of the unified memory (~110GB with `--gpu-memory-utilization 0.9`). Service B (Qwen2.5-Coder-3B-Instruct) fails with `RuntimeError: Engine core initialization failed` due to insufficient remaining memory.
- **Expected per Criterion 7**: "Document as expected hardware behavior if GPU memory insufficient for two models simultaneously"
- **Status**: Expected behavior — not a bug. GB10 unified memory cannot run two large vLLM services simultaneously. Criterion 7 documented as PASS (OOM correctly reported).
