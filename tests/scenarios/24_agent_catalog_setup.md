# Scenario 24: Catalog-Driven Autonomous Setup — The Crown Jewel

**Difficulty**: Hard
**Estimated Duration**: 25 minutes
**Prerequisites**: All previous scenarios, Docker + GPU, LLM API access
**Tier**: 5 — Agent Intelligence

## User Story

"I just got a new machine with a GPU and I want AIMA's AI agent to set everything up for me — detect my hardware, find the best deployment recipe from the catalog, deploy the optimal model+engine stack, and verify it works by running an inference test. I shouldn't have to know anything about Docker, vLLM, or model formats. The agent handles everything."

## Success Criteria

1. [ ] Agent detects hardware via `device.detect` (identifies NVIDIA GPU, VRAM, architecture)
2. [ ] Agent queries catalog via `catalog.match` with detected hardware specs
3. [ ] If no matching recipe exists, agent creates one via `catalog.create`
4. [ ] Agent applies recipe and creates a service (via `service.create` + `service.start`)
5. [ ] Agent tracks deployment progress (checks service status during startup)
6. [ ] Agent tests inference after deployment succeeds (`inference.chat` with a test prompt)
7. [ ] Agent completes the entire workflow within the tool call round limit
8. [ ] Agent's reasoning chain is coherent (hardware → recipe → deploy → test)

## Environment Setup

```bash
# Start AIMA with LLM API configured
export AIMA_LLM_API_KEY="<your-kimi-api-key>"
export OPENAI_BASE_URL="https://api.kimi.com/coding/v1"
export OPENAI_MODEL="kimi-for-coding"
export OPENAI_USER_AGENT="claude-code/1.0"

/tmp/aima start &
AIMA_PID=$!
sleep 2

# Clean slate — no services running
/tmp/aima service list --output json | jq -r '.items[].id' | while read id; do
  /tmp/aima service stop "$id" 2>/dev/null
done
docker ps -a --filter label=aima.engine -q | xargs -r docker rm -f 2>/dev/null
```

### Test 1: Full Autonomous Setup

```bash
/tmp/aima agent chat --message "I just got this machine and I want to run AI inference on it. Can you:
1. Check what hardware I have
2. Find the best deployment recipe for my GPU
3. Set up a service with the optimal configuration
4. Start the service (use timeout=600 since large models take time)
5. Once it's running, test it with a simple coding prompt

Please do all of this automatically."
# Expected agent behavior:
# Round 1: device.detect → identifies NVIDIA GB10, ~24GB VRAM
# Round 2: catalog.match → queries with NVIDIA, 24GB, Linux
#   BUG WATCH: catalog.match may fail due to vram_gb string→int conversion
# Round 3: If no recipe found, catalog.create → creates recipe
# Round 4: service.create → creates service from recipe config
# Round 5: service.start → starts with timeout=600
# Round 6-8: service.list → polls status until "running"
# Round 9: inference.chat → tests with coding prompt
# Round 10: Reports success

# BUG WATCH: Bug #29 — 10 rounds may not be enough for this 7+ step chain
```

### Test 2: Hardware Detection Quality

```bash
# Separately verify what device.detect returns
/tmp/aima agent chat --message "What GPU do I have? Give me the exact model, VRAM, architecture, and driver version."
# Expected: Agent calls device.detect, parses nvidia-smi output
# BUG WATCH: device.detect may return raw nvidia-smi output that the agent must parse
```

### Test 3: Catalog Matching

```bash
# Test catalog matching with specific hardware
/tmp/aima agent chat --message "Search the catalog for the best recipe for an NVIDIA GPU with 24GB VRAM running Linux. Tell me the top 3 matches and their scores."
# Expected: Agent calls catalog.match with correct parameters
# BUG WATCH: catalog.match vram_gb string conversion may make VRAM score always 0
```

### Test 4: Recipe Creation

```bash
/tmp/aima agent chat --message "Create a new catalog recipe for vLLM on NVIDIA Blackwell GPUs with 24GB VRAM. Use these settings:
- gpu_memory_utilization: 0.90
- max_model_len: 8192
- quantization: fp16
Name it 'optimal-blackwell-24gb'."
# Expected: Agent calls catalog.create with the specified parameters
```

### Test 5: Apply Recipe

```bash
/tmp/aima agent chat --message "Apply the recipe 'optimal-blackwell-24gb' for my existing model. What deployment plan does it generate?"
# Expected: Agent calls catalog.apply, interprets the deployment plan
# BUG WATCH: catalog.apply_recipe returns a plan but doesn't deploy
# The agent must understand it needs to call service.create separately
```

### Test 6: Deployment Tracking

```bash
# If the agent started a service in Test 1, check if it tracked progress
/tmp/aima agent chat --message "What's the current status of all my services? Are any still starting up?"
# Expected: Agent calls service.list, reports current statuses
```

### Test 7: End-to-End Inference Test

```bash
# Final validation — does the deployed service actually work?
/tmp/aima agent chat --message "Run an inference test against the running service. Ask it to write a Python function that calculates factorial, and show me the response."
# Expected: Agent calls inference.chat, gets a code response, displays it
```

### Test 8: Reasoning Chain Quality

```bash
# Meta-test: ask the agent to explain its reasoning
/tmp/aima agent chat --message "Summarize everything you've done in this session. What hardware did you detect? What recipe did you use? What service did you deploy? How did the inference test go?"
# Expected: coherent summary covering all steps
# BUG WATCH: If each `agent chat` starts a fresh conversation, the agent won't remember
```

## Advanced Test: Full Pipeline Without Prompting

```bash
# The ultimate test — one prompt, agent does everything
/tmp/aima agent chat --message "Set up this machine for AI inference from scratch. Detect hardware, find or create a recipe, deploy a service, and test it. I'll wait."
# This is the crown jewel test — does the agent have the autonomy and capability
# to execute a full 7+ step workflow from a single natural language request?
```

## Cleanup

```bash
# Stop all services
/tmp/aima service list --output json | jq -r '.items[] | select(.status=="running") | .id' | while read id; do
  /tmp/aima service stop "$id"
done

docker ps -a --filter label=aima.engine -q | xargs -r docker rm -f

kill $AIMA_PID 2>/dev/null
```

## Hints for the Operator

- Agent conversation: `pkg/agent/conversation.go` (loop and round limit)
- Available tools: all registered units from 13 domains
- Catalog match scoring: GPU vendor (+40), model (+30), arch (+15), VRAM (+10), OS (+5)
- Device detect: returns nvidia-smi output; agent must parse VRAM, model, etc.
- Service start: requires `timeout` parameter for large model loading

## Known Pitfalls

- **Bug #29 — Round limit (10) too low**: This scenario requires 7+ tool calls minimum: device.detect → catalog.match → (optional catalog.create) → service.create → service.start → (service.list × N for status polling) → inference.chat. With any retries, 10 rounds is insufficient.
- **`catalog.match` string-to-int bug (same as S11)**: GET request passes `vram_gb` as string `"24"`. The `toInt()` conversion in `pkg/unit/catalog/queries.go:143-144` fails silently, making VRAM score always 0. This means recipes won't be ranked correctly by VRAM.
- **`catalog.apply_recipe` doesn't deploy**: This command returns a deployment plan (service configuration) but does NOT call `service.create`. The agent must interpret the plan and make a separate service.create call. If the agent expects `apply_recipe` to handle deployment end-to-end, it will get stuck.
- **`device.detect` raw output**: The device detection returns raw nvidia-smi output, not structured JSON. The agent must parse this text to extract VRAM, model name, and architecture. Different LLMs have varying ability to parse this correctly.
- **Conversation context loss**: Each `aima agent chat` CLI invocation starts a fresh conversation. Tests 1-8 each start with no memory of previous interactions. The "Full Pipeline" advanced test is the only one that tests multi-step reasoning within a single conversation.
- **LLM quality matters**: The quality of the agent's reasoning depends heavily on the LLM used. Kimi-for-coding is optimized for tool use but may not handle complex multi-step planning as well as Claude or GPT-4.

## Key Code to Inspect

| File | What to Check |
|------|---------------|
| `pkg/agent/conversation.go` | Round limit, tool call loop |
| `pkg/unit/catalog/queries.go` | catalog.match scoring, toInt() bug |
| `pkg/unit/catalog/commands.go` | catalog.apply implementation |
| `pkg/unit/device/queries.go` | device.detect output format |
| `pkg/gateway/agent_executor.go` | Tool call bridge |

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken (total and per-agent-round)
- Number of tool call rounds used for the full pipeline
- Quality assessment: did the agent's reasoning chain make sense?
- Whether the agent could complete the full pipeline in one conversation
- Specific tool calls the agent made (in order)
- LLM used and its effectiveness for this task
