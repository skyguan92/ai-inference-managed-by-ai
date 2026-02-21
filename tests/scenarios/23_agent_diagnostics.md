# Scenario 23: Agent Diagnostics & Self-Healing

**Difficulty**: Hard
**Estimated Duration**: 20 minutes
**Prerequisites**: Scenarios 1-5 completed, Docker + GPU, LLM API access (Kimi or OpenAI-compatible)
**Tier**: 5 — Agent Intelligence

## User Story

"My inference service went down and I want the AI agent to diagnose the problem and fix it without my intervention. The agent should: check the service status, inspect logs, figure out what went wrong, and execute recovery steps (stop → cleanup → restart). I want to test if the agent is actually intelligent enough to handle multi-step troubleshooting."

## Success Criteria

1. [ ] Agent can check service status and identify the failed service
2. [ ] Agent calls `device.detect` to verify GPU availability
3. [ ] Agent proposes a diagnosis and recovery plan
4. [ ] Agent executes multi-step recovery: stop → cleanup → restart → verify
5. [ ] Conversation history is visible across messages (agent maintains context)
6. [ ] Recovery completes within the tool call round limit
7. [ ] Agent's reasoning is coherent (explains what it found and what it's doing)

## Environment Setup

```bash
# Start AIMA with LLM API configured
# Option A: Kimi API
export AIMA_LLM_API_KEY="<your-kimi-api-key>"
export OPENAI_BASE_URL="https://api.kimi.com/coding/v1"
export OPENAI_MODEL="kimi-for-coding"
export OPENAI_USER_AGENT="claude-code/1.0"

# Option B: Local LLM (if available)
# No env vars needed — AIMA auto-detects local services

/tmp/aima start &
AIMA_PID=$!
sleep 2

# Create a model and service
/tmp/aima model create --name "agent-diag-model" --source ollama --repo "tinyllama"
MODEL_ID=$(/tmp/aima model list --output json | jq -r '.items[-1].id')
/tmp/aima service create --model-id "$MODEL_ID" --engine-type vllm
SVC_ID=$(/tmp/aima service list --output json | jq -r '.items[-1].id')

# Deliberately break the service to create a diagnostic challenge
# Start the service with a very short timeout to force failure
/tmp/aima service start "$SVC_ID" --wait --timeout 5 2>/dev/null
# This should fail and leave the service in "failed" state

# Verify it's broken
/tmp/aima service list
# Expected: service shows "failed" status
```

### Test 1: Agent Diagnoses Failed Service

```bash
/tmp/aima agent chat --message "I have a service that's not working. Can you check the status of all my services and tell me what's wrong?"
# Expected: Agent calls service.list, identifies the failed service, reports status
```

### Test 2: Agent Checks Hardware

```bash
/tmp/aima agent chat --message "Is my GPU available and healthy? Check the hardware."
# Expected: Agent calls device.detect, reports GPU status (NVIDIA GB10, memory, temperature)
```

### Test 3: Agent Proposes Recovery

```bash
/tmp/aima agent chat --message "The service '$SVC_ID' is in failed state. What should we do to fix it? Don't do anything yet, just tell me the plan."
# Expected: Agent explains a recovery plan:
# 1. Stop the failed service (clean up residual state)
# 2. Clean up any orphaned containers
# 3. Restart the service with appropriate timeout
# 4. Verify it's running
```

### Test 4: Agent Executes Recovery

```bash
/tmp/aima agent chat --message "Go ahead and fix service '$SVC_ID'. Stop it, clean up, and restart it with a 600 second timeout. Then verify it's running."
# Expected: Agent executes the multi-step plan:
# - Calls service.stop
# - Calls service.start with timeout=600
# - Calls service.list to verify "running" status
# - Reports success

# BUG WATCH: Bug #29 — 10 tool call rounds may not be enough
# Stop + start + verify = 3 calls minimum, but retries may push over limit
```

### Test 5: Conversation Context

```bash
# Test that agent remembers previous context
/tmp/aima agent chat --message "What was the original problem we just fixed?"
# Expected: Agent references the failed service from earlier messages
# BUG WATCH: Agent conversation history may be lost between CLI invocations
# Each `aima agent chat` call may start a fresh conversation
```

### Test 6: Round Limit

```bash
# Check if the agent completed recovery within the round limit
# Look at the output from Test 4 — if it says "max rounds reached" or similar,
# the round limit is too low

# Also test with a complex request that requires many steps
/tmp/aima agent chat --message "Check all services, stop any failed ones, clean up orphaned Docker containers using service cleanup, restart the services with a 300s timeout, and then verify everything is running. Give me a status report."
# Expected: completes within 10 rounds (or fails with round limit error)
# Bug #29 prediction: 10 rounds may not be enough
```

### Test 7: Agent Reasoning Quality

```bash
# Test coherence with a diagnostic puzzle
# First, create an ambiguous situation
/tmp/aima model create --name "mystery-model" --source ollama --repo "phi"
/tmp/aima service create --model-id "$(/tmp/aima model list --output json | jq -r '.items[-1].id')" --engine-type vllm

/tmp/aima agent chat --message "I created a new service but I'm not sure if it will work on my hardware. Can you check if my GPU has enough memory for it and recommend whether to start it?"
# Expected: Agent calls device.detect, checks VRAM, maybe calls catalog.match,
# and gives a reasoned recommendation
```

## Cleanup

```bash
# Stop all services
/tmp/aima service list --output json | jq -r '.items[] | select(.status=="running") | .id' | while read id; do
  /tmp/aima service stop "$id" 2>/dev/null
done

docker ps -a --filter label=aima.engine -q | xargs -r docker rm -f

kill $AIMA_PID 2>/dev/null
```

## Hints for the Operator

- Agent implementation: `pkg/agent/` (conversation loop, LLM clients)
- Tool execution: `pkg/gateway/agent_executor.go` (AgentExecutor adapter)
- Agent config: `pkg/config/config.go` (AgentConfig), env var overrides
- MCP tools available to agent: all registered units from 13 domains
- Round limit: hardcoded in conversation loop

## Known Pitfalls

- **Bug #29 — Tool call round limit (10)**: The agent's conversation loop allows only 10 tool call rounds. Complex diagnostic scenarios (status check → log inspection → stop → cleanup → restart → verify) may require 6+ calls. With retries, 10 may not be enough.
- **Bug #26 — Config not loaded**: Agent config from `config.toml` may not be properly loaded. All LLM settings must be passed via env vars (AIMA_LLM_API_KEY, OPENAI_MODEL, OPENAI_BASE_URL, OPENAI_USER_AGENT).
- **Conversation history not persisted**: Each `aima agent chat` CLI invocation may start a fresh conversation. The agent loses context between calls. This means Test 5 may fail — the agent won't remember previous messages.
- **Agent lacks Docker diagnostics**: The agent can call service.stop and service.start, but may not have tools for direct Docker inspection (container logs, docker inspect). It relies on AIMA's abstractions.
- **LLM API latency**: Kimi API calls can take 5-30 seconds per round. A 10-round conversation may take 1-5 minutes total. Set appropriate expectations.

## Key Code to Inspect

| File | What to Check |
|------|---------------|
| `pkg/agent/llm/openai.go` | OpenAI-compatible client (used for Kimi) |
| `pkg/agent/conversation.go` | Conversation loop, round limit |
| `pkg/gateway/agent_executor.go` | AgentExecutor tool bridge |
| `pkg/cli/agent.go` | Agent CLI, config loading |
| `pkg/config/config.go` | AgentConfig struct |

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Number of tool call rounds used for recovery (Test 4)
- Whether conversation context persists between CLI invocations
- Quality of agent reasoning (coherent plan vs random tool calls)
- LLM API used and per-round latency
