# Scenario 9: Config & Persistence Resilience

**Difficulty**: Easy
**Estimated Duration**: 5 minutes
**Prerequisites**: Scenario 1 completed (binary built)
**Tier**: 1 — Foundation (no Docker/GPU required)

## User Story

"I want to make sure AIMA handles configuration edge cases gracefully — what happens if I give it a malformed config file? What if I set conflicting resource pool percentages? I also want to understand which data survives a restart and which doesn't, so I know what I can rely on in production."

## Success Criteria

1. [ ] Malformed TOML config file produces a clear, actionable error message (not a panic or cryptic Go error)
2. [ ] Pool overflow (`inference_pool_pct + container_pool_pct > 1.0`) is rejected at startup with a descriptive error
3. [ ] Invalid log level (e.g., `log_level = "TRACE"`) produces a clear error or falls back to a default with a warning
4. [ ] Custom data directory (`data_dir = "/tmp/aima-test-data"`) is created and used for SQLite storage
5. [ ] SQLite fallback behavior verified: models use FileStore (persisted), services use appropriate store
6. [ ] Env var precedence is correct: `AIMA_LLM_API_KEY` overrides `[agent].llm_api_key` in config.toml
7. [ ] Gateway timeout gap confirmed: `AIMA_GATEWAY_TIMEOUT` env var does NOT exist in `ApplyEnvOverrides` (expected gap)

## Environment Setup

```bash
# SSH into remote machine
ssh qujing@100.105.58.16

# Ensure binary is built
cd /home/qujing/projects/ai-inference-managed-by-ai
GOPROXY=https://goproxy.cn,direct /usr/local/go/bin/go build -o /tmp/aima ./cmd/aima

# Backup existing config
cp ~/.aima/config.toml ~/.aima/config.toml.bak 2>/dev/null || true
```

### Test 1: Malformed TOML

```bash
cat > /tmp/aima-bad-config.toml << 'EOF'
[gateway
port = "not-a-number"
this is not valid toml
EOF

AIMA_CONFIG=/tmp/aima-bad-config.toml /tmp/aima model list
# Expected: clear error about TOML parse failure
```

### Test 2: Pool Overflow

```bash
cat > /tmp/aima-pool-overflow.toml << 'EOF'
[resource]
inference_pool_pct = 0.8
container_pool_pct = 0.5
EOF

AIMA_CONFIG=/tmp/aima-pool-overflow.toml /tmp/aima start
# Expected: error about pool percentages exceeding 1.0
```

### Test 3: Invalid Log Level

```bash
cat > /tmp/aima-bad-log.toml << 'EOF'
[logging]
level = "TRACE"
EOF

AIMA_CONFIG=/tmp/aima-bad-log.toml /tmp/aima model list
# Expected: error or warning about invalid log level, fallback to default
```

### Test 4: Custom Data Directory

```bash
rm -rf /tmp/aima-test-data
cat > /tmp/aima-custom-dir.toml << 'EOF'
[general]
data_dir = "/tmp/aima-test-data"
EOF

/tmp/aima --config /tmp/aima-custom-dir.toml model list
ls -la /tmp/aima-test-data/
# Expected: directory created with SQLite files (aima.db)
```

### Test 5: Env Var Precedence

```bash
# Set conflicting values
cat > /tmp/aima-agent.toml << 'EOF'
[agent]
llm_api_key = "from-config-file"
llm_model = "from-config-file"
EOF

# Env var should win
AIMA_CONFIG=/tmp/aima-agent.toml \
AIMA_LLM_API_KEY="from-env-var" \
OPENAI_MODEL="from-openai-env" \
/tmp/aima agent chat --message "test" 2>&1 | head -5
# Expected: uses "from-env-var" key (will fail auth, but the attempt URL/model should reflect env vars)
```

### Test 6: Gateway Timeout Gap

```bash
# Check if AIMA_GATEWAY_TIMEOUT is recognized
grep -r "AIMA_GATEWAY_TIMEOUT" /home/qujing/projects/ai-inference-managed-by-ai/pkg/
# Expected: no results — this env var is NOT implemented in ApplyEnvOverrides
```

## Hints for the Operator

- Config loading code is in `pkg/config/config.go`
- `ApplyEnvOverrides()` is where env vars override config file values
- Store implementations: `pkg/unit/model/store.go`, `pkg/unit/service/store.go`
- Check what happens when `AIMA_CONFIG` points to a nonexistent file
- Try `--config` flag if it exists as a CLI override

## Known Pitfalls

- **Bug #26 verification**: Agent config from `config.toml` [agent] section may not be loaded at all. This scenario specifically tests that chain.
- **Config validation gap**: The config struct may be parsed successfully but individual fields aren't validated (e.g., pool percentages aren't checked for sum > 1.0).
- **SQLite vs Memory**: Models use `FileModelStore` (SQLite-backed), but other domains (pipeline, catalog, skill, alert, resource) may use in-memory stores. This means those domains lose ALL data on restart.
- **AIMA_CONFIG env var**: May not be implemented. The binary may only look at `~/.aima/config.toml` hardcoded path.
- **TOML parsing library**: Go TOML libraries differ in error message quality. Check if the error includes line numbers and field names.

## Key Code to Inspect

| File | What to Check |
|------|---------------|
| `pkg/config/config.go` | Config struct, validation, `ApplyEnvOverrides()` |
| `pkg/cli/root.go` | How config is loaded and passed to subsystems |
| `pkg/unit/model/store.go` | FileModelStore (SQLite) implementation |
| `pkg/unit/service/store.go` | Service store implementation (memory or SQLite?) |
| `pkg/registry/register.go` | Which stores are memory-only vs persistent |

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Config edge cases that weren't covered by the tests above
