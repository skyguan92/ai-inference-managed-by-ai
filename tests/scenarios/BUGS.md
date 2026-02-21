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
