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

Scenarios must be executed sequentially, 1 through 8. Each builds on context and state from previous ones:

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
