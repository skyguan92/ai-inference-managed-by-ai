# Scenario 17: Inference Proxy Routing

**Difficulty**: Medium
**Estimated Duration**: 15 minutes
**Prerequisites**: Scenarios 1-2 completed, Docker + GPU, a running vLLM service
**Tier**: 3 — Production

## User Story

"I have a vLLM service running with a code model. I want to test the inference proxy from multiple angles — CLI, HTTP API, and directly. I need to verify that the proxy correctly resolves which service handles which model, that parameters like max_tokens and temperature are passed through, and that errors are clear when things go wrong."

## Environment Setup

```bash
# Ensure a model and vLLM service are running
/tmp/aima start &
AIMA_PID=$!
sleep 2

# Verify a service is running (from previous scenarios or start one)
/tmp/aima service list
# If no running service, start one:
# /tmp/aima service start <service-id> --wait --timeout 600

# Get the running service's model name and port
MODEL_NAME=$(/tmp/aima model list --output json | jq -r '.items[0].name')
SERVICE_PORT=$(/tmp/aima service list --output json | jq -r '.items[] | select(.status=="running") | .port')
echo "Model: $MODEL_NAME, Port: $SERVICE_PORT"
```

## Success Criteria

1. [ ] CLI `inference chat` returns a code response from vLLM
2. [ ] HTTP API `inference.chat` returns the same format as CLI
3. [ ] Non-existent model name returns clear error (not a timeout or 500)
4. [ ] No running services → clear error message (not "connection refused")
5. [ ] `max_tokens` parameter is respected (shorter response with low value)
6. [ ] `temperature=0` gives deterministic output (same response twice)
7. [ ] Proxy correctly routes request to the right service endpoint

## Test Execution

### Test 1: CLI Inference Chat

```bash
/tmp/aima inference chat \
  --model "$MODEL_NAME" \
  --message "Write a Python function that returns the fibonacci sequence up to n terms" \
  --max-tokens 200
# Expected: code response from vLLM service
# Record: response time, token count, response quality
```

### Test 2: HTTP API Inference Chat

```bash
curl -s -X POST http://localhost:9090/api/inference/chat \
  -H "Content-Type: application/json" \
  -d "{
    \"model\": \"$MODEL_NAME\",
    \"message\": \"Write a Python function that returns the fibonacci sequence up to n terms\",
    \"max_tokens\": 200
  }" | jq .
# Expected: same format as CLI output
# Check: does the response include usage stats (prompt_tokens, completion_tokens)?
```

### Test 3: Non-Existent Model

```bash
/tmp/aima inference chat \
  --model "nonexistent-model-xyz" \
  --message "hello" \
  --max-tokens 50
# Expected: clear error like "model not found" or "no service running for model"
# Not: timeout, connection refused, or cryptic proxy error
```

```bash
curl -s -w "\nHTTP %{http_code}\n" -X POST http://localhost:9090/api/inference/chat \
  -H "Content-Type: application/json" \
  -d '{"model": "nonexistent-model-xyz", "message": "hello", "max_tokens": 50}'
# Expected: structured error response with appropriate HTTP status
```

### Test 4: No Running Services

```bash
# Stop all services first
/tmp/aima service list --output json | jq -r '.items[] | select(.status=="running") | .id' | while read id; do
  /tmp/aima service stop "$id"
done

# Now try inference
/tmp/aima inference chat --model "$MODEL_NAME" --message "hello"
# Expected: clear error like "no running service for model"
# Not: "connection refused to localhost:8000"

# Restart the service for remaining tests
# /tmp/aima service start <service-id> --wait --timeout 600
```

### Test 5: Max Tokens Respected

```bash
# Short response
curl -s -X POST http://localhost:9090/api/inference/chat \
  -H "Content-Type: application/json" \
  -d "{\"model\": \"$MODEL_NAME\", \"message\": \"Write a very long essay about AI\", \"max_tokens\": 20}" | jq '.response | length'
# Expected: short response (roughly 20 tokens worth of text)

# Longer response
curl -s -X POST http://localhost:9090/api/inference/chat \
  -H "Content-Type: application/json" \
  -d "{\"model\": \"$MODEL_NAME\", \"message\": \"Write a very long essay about AI\", \"max_tokens\": 500}" | jq '.response | length'
# Expected: longer response
```

### Test 6: Temperature Determinism

```bash
# temperature=0 should give deterministic output
RESP1=$(curl -s -X POST http://localhost:9090/api/inference/chat \
  -H "Content-Type: application/json" \
  -d "{\"model\": \"$MODEL_NAME\", \"message\": \"What is 2+2?\", \"max_tokens\": 10, \"temperature\": 0}" | jq -r '.response')

RESP2=$(curl -s -X POST http://localhost:9090/api/inference/chat \
  -H "Content-Type: application/json" \
  -d "{\"model\": \"$MODEL_NAME\", \"message\": \"What is 2+2?\", \"max_tokens\": 10, \"temperature\": 0}" | jq -r '.response')

echo "Response 1: $RESP1"
echo "Response 2: $RESP2"
[ "$RESP1" = "$RESP2" ] && echo "DETERMINISTIC: PASS" || echo "DETERMINISTIC: FAIL"
```

### Test 7: Direct vs Proxy Comparison

```bash
# Direct vLLM call (bypass AIMA proxy)
DIRECT=$(curl -s http://localhost:$SERVICE_PORT/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d "{\"model\": \"$MODEL_NAME\", \"messages\": [{\"role\": \"user\", \"content\": \"Say hello\"}], \"max_tokens\": 20}" | jq -r '.choices[0].message.content')

# Via AIMA proxy
PROXY=$(curl -s -X POST http://localhost:9090/api/inference/chat \
  -H "Content-Type: application/json" \
  -d "{\"model\": \"$MODEL_NAME\", \"message\": \"Say hello\", \"max_tokens\": 20}" | jq -r '.response')

echo "Direct: $DIRECT"
echo "Proxy: $PROXY"
# Both should return meaningful responses (content may differ due to temperature)
```

## Cleanup

```bash
kill $AIMA_PID 2>/dev/null
```

## Hints for the Operator

- Inference proxy: `pkg/infra/provider/inference_provider.go` (ProxyInferenceProvider)
- Endpoint resolution: `resolveEndpoint()` method
- CLI inference: `pkg/cli/inference.go`
- HTTP route: `pkg/gateway/routes.go` (inference.chat)

## Known Pitfalls

- **`resolveEndpoint()` case-insensitive matching**: The endpoint resolver scans service names case-insensitively. If two models have similar names (e.g., "Qwen3" and "qwen3-chat"), the wrong service may be matched.
- **Gateway WriteTimeout kills long inference**: The HTTP `WriteTimeout=30s` in `pkg/gateway/server.go:87` may terminate long inference requests. Large models with many output tokens may exceed this.
- **CLI parameter passthrough**: The CLI may not correctly pass all parameters (temperature, top_p, etc.) to the gateway API.
- **Empty response handling**: If vLLM returns an empty choices array, the proxy should return a clear error, not an index-out-of-range panic.
- **OpenAI format translation**: AIMA uses its own inference.chat format; the proxy must translate to/from OpenAI's chat completions format when talking to vLLM.

## Key Code to Inspect

| File | What to Check |
|------|---------------|
| `pkg/infra/provider/inference_provider.go` | ProxyInferenceProvider, resolveEndpoint |
| `pkg/unit/inference/commands.go` | Chat command implementation |
| `pkg/cli/inference.go` | CLI parameter handling |
| `pkg/gateway/server.go` | WriteTimeout setting |
| `pkg/gateway/routes.go` | inference.chat route |

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Response latency for CLI vs HTTP vs direct vLLM
- Whether temperature=0 is actually deterministic
- Any parameter passthrough issues found
