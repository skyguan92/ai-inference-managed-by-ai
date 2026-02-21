# AIMA Acceptance Test Scenarios

These are real-world acceptance test documents designed to be executed by an AI operator (Sonnet model) against a live AIMA deployment. Each scenario describes a user goal in natural language and defines observable success criteria.

## Purpose

AIMA has a mock-based E2E test suite (39 tests, all passing) but those only validate wiring -- they don't catch real integration bugs. Bugs #10 through #17 were all found through manual testing on a real ARM64 machine with real Docker containers. These scenarios aim to:

1. Exercise the full stack: Docker, inference engines, agent, all CLI commands
2. Surface integration bugs that unit/mock tests cannot detect
3. Serve as prompts for a Sonnet model to autonomously operate AIMA
4. Build confidence that the system works end-to-end before releases

## Target Environment

| Property | Value |
|----------|-------|
| Host | `qujing@100.105.58.16` (ARM64 Linux, NVIDIA GB10 GPU) |
| Project | `/home/qujing/projects/ai-inference-managed-by-ai` |
| Go | `/usr/local/go/bin/go` |
| Binary | `/tmp/aima` |
| Config | `~/.aima/config.toml` (TOML format) |
| Model Dir | `/mnt/data/models/` (Qwen3-Coder-Next-FP8 at 75GB, 40 shards) |
| Docker Image | `zhiwen-vllm:0128` (ARM64-native, ~25GB) |
| Go Proxy | `GOPROXY=https://goproxy.cn,direct` (direct access to golang.org times out) |

## Building the Binary

```bash
cd /home/qujing/projects/ai-inference-managed-by-ai
GOPROXY=https://goproxy.cn,direct /usr/local/go/bin/go build -o /tmp/aima ./cmd/aima
```

Always rebuild from source if any code has changed. The binary must match the current source.

## Kimi API Configuration (Scenarios 4, 5, 8)

Add to `~/.aima/config.toml`:

```toml
[agent]
llm_provider = "openai"
llm_model = "kimi-for-coding"
llm_base_url = "https://api.kimi.com/coding/v1"
llm_api_key = "set via AIMA_LLM_API_KEY env var"
llm_user_agent = "claude-code/1.0"
```

Set the API key as an environment variable:

```bash
export AIMA_LLM_API_KEY="<your-kimi-api-key>"
```

## Execution Order

### Phase 0 — Original Scenarios (S01-S08)

Scenarios 1 through 8 are the original acceptance tests focused on CLI operations and single-service Docker lifecycle. Execute sequentially:

| # | Scenario | Difficulty | Duration | File |
|---|----------|-----------|----------|------|
| 1 | Hello AIMA -- First Contact | Easy | 5 min | `01_hello_aima.md` |
| 2 | From Zero to Inference -- Deploy Qwen3-Coder | Hard | 15 min | `02_zero_to_inference.md` |
| 3 | Multimodal Service Stack -- ASR + TTS | Medium | 10 min | `03_multimodal_stack.md` |
| 4 | AI Operator -- Natural Language Management | Medium | 10 min | `04_ai_operator.md` |
| 5 | Agent Self-Service -- Agent Deploys Model E2E | Hard | 15 min | `05_agent_self_service.md` |
| 6 | Service Lifecycle Stress -- Container Churn | Medium | 10 min | `06_lifecycle_stress.md` |
| 7 | Recovery from Chaos -- Orphan Containers | Medium | 10 min | `07_recovery_from_chaos.md` |
| 8 | Full-Stack Integration -- All Interfaces | Hard | 20 min | `08_full_stack_integration.md` |

### Phase A — Foundation (S09-S12)

Tests domain logic, config, and persistence. Needs only the AIMA binary (no Docker/GPU for S09, HTTP server for S10-S12).

| # | Scenario | Difficulty | Duration | File |
|---|----------|-----------|----------|------|
| 9 | Config & Persistence Resilience | Easy | 5 min | `09_config_persistence.md` |
| 10 | Pipeline DAG Execution | Medium | 5 min | `10_pipeline_dag.md` |
| 11 | Catalog Recipe Matching Algorithm | Medium | 5 min | `11_catalog_matching.md` |
| 12 | Skill Registry and Search | Easy | 3 min | `12_skill_registry.md` |

### Phase B — API Surface (S13-S15)

Tests HTTP REST API, MCP protocol, and auth middleware. Needs the HTTP server running.

| # | Scenario | Difficulty | Duration | File |
|---|----------|-----------|----------|------|
| 13 | HTTP REST API Completeness | Medium | 8 min | `13_http_api_surface.md` |
| 14 | MCP Protocol Compliance | Medium | 8 min | `14_mcp_protocol.md` |
| 15 | Auth Middleware Wiring | Easy | 5 min | `15_auth_security.md` |

### Phase C — Production (S16-S18)

Tests multi-service deployment, inference proxy, and monitoring. Needs Docker + GPU.

| # | Scenario | Difficulty | Duration | File |
|---|----------|-----------|----------|------|
| 16 | Multi-Model Concurrent Deployment | Hard | 20 min | `16_multi_service.md` |
| 17 | Inference Proxy Routing | Medium | 15 min | `17_inference_quality.md` |
| 18 | Service Monitoring & Events | Medium | 10 min | `18_service_monitoring.md` |

### Phase D — Resilience (S19-S22)

Tests failure modes, recovery, and shutdown behavior. Needs Docker (GPU for some).

| # | Scenario | Difficulty | Duration | File |
|---|----------|-----------|----------|------|
| 19 | Process Restart Data Survival | Medium | 10 min | `19_restart_survival.md` |
| 20 | Port Contention & Resource Exhaustion | Medium | 10 min | `20_port_contention.md` |
| 21 | Timeout Cascade | Medium | 10 min | `21_timeout_cascade.md` |
| 22 | Graceful Shutdown Under Load | Medium | 10 min | `22_graceful_shutdown.md` |

### Phase E — Agent Intelligence (S23-S24)

Tests AI agent autonomy. Needs Docker + GPU + LLM API access.

| # | Scenario | Difficulty | Duration | File |
|---|----------|-----------|----------|------|
| 23 | Agent Diagnostics & Self-Healing | Hard | 20 min | `23_agent_diagnostics.md` |
| 24 | Catalog-Driven Autonomous Setup | Hard | 25 min | `24_agent_catalog_setup.md` |

### High-Confidence Bug Predictions (S09-S24)

These bugs have >80% probability of being found during testing:

| # | Scenario | Bug | Severity |
|---|----------|-----|----------|
| 1 | S11/S24 | `catalog.match` GET passes vram_gb as string → `toInt()` silent fail | P1 |
| 2 | S13 | `bodyInputMapper` returns `{}` on JSON error (not 400) | P1 |
| 3 | S15 | No TOML field to enable auth — `EnableAuth` only in code | P0 |
| 4 | S18 | EventBus never passed to RegisterAll — events dropped | P1 |
| 5 | S19 | 5/13 domains memory-only — data lost on restart | P1 |
| 6 | S21 | WriteTimeout=30s kills agent chat (10min timeout) | P0 |
| 7 | S15 | Forced auth routes permanently locked with no API keys | P1 |
| 8 | S10 | pipeline.cancel URL maps {id} to run_id (user passes pipeline_id) | P2 |

## Bug Report Format

When a scenario uncovers a bug, document it using this format:

```markdown
### Bug #N: Short Description

- **Scenario**: Which scenario triggered it
- **Command**: What was run
- **Expected**: What should have happened
- **Actual**: What actually happened (include error output)
- **Root Cause**: (if identifiable from logs/source)
- **Suggested Fix**: (if obvious)
- **Severity**: Blocker / Major / Minor
```

Bugs are tracked in `tests/scenarios/BUGS.md` (created during execution).

## Reporting

After each scenario, the operator should produce a report covering:

1. **Result per criterion**: PASS or FAIL with evidence (command output snippets)
2. **Bugs discovered**: Using the bug report format above
3. **Time taken**: Wall clock time for the scenario
4. **Unexpected observations**: Anything surprising, even if not a bug
