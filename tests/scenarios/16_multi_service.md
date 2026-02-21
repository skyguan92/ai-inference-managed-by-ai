# Scenario 16: Multi-Model Concurrent Deployment

**Difficulty**: Hard
**Estimated Duration**: 20 minutes
**Prerequisites**: Scenarios 1-2 completed, Docker available, GPU available
**Tier**: 3 — Production

## User Story

"I need to run multiple inference services simultaneously — a code completion model and a chat model, both on the same GPU. I want to verify that AIMA correctly allocates different ports, shares GPU memory, keeps services isolated, and cleans up properly when I stop one without affecting the other."

## Success Criteria

1. [ ] Two services created for different models with unique auto-assigned ports
2. [ ] Both services appear in `service list` with correct statuses
3. [ ] Stopping one service does NOT affect the other running service
4. [ ] After stopping a service, its port is released and reusable by a new service
5. [ ] ServiceID format is correct and parseable (e.g., `svc-{engine}-{model-hash}`)
6. [ ] `service.recommend` returns appropriate suggestions based on available resources
7. [ ] GPU memory is shared (both services run without OOM, assuming models fit)

## Environment Setup

```bash
# Ensure AIMA server is running with Docker access
/tmp/aima start &
AIMA_PID=$!
sleep 2

# Verify GPU is available
/tmp/aima device detect
nvidia-smi --query-gpu=memory.total,memory.free --format=csv
```

### Test 1: Create Two Services

```bash
# Create first model (small, for quick testing)
/tmp/aima model create --name "model-alpha" --source ollama --repo "tinyllama"
MODEL_A=$(/tmp/aima model list --output json | jq -r '.items[0].id')

# Create second model
/tmp/aima model create --name "model-beta" --source ollama --repo "phi"
MODEL_B=$(/tmp/aima model list --output json | jq -r '.items[1].id')

# Create service for model A
/tmp/aima service create --model-id "$MODEL_A" --engine-type vllm
SVC_A=$(/tmp/aima service list --output json | jq -r '.items[0].id')

# Create service for model B
/tmp/aima service create --model-id "$MODEL_B" --engine-type vllm
SVC_B=$(/tmp/aima service list --output json | jq -r '.items[1].id')

echo "Service A: $SVC_A"
echo "Service B: $SVC_B"
# Expected: two different service IDs with different auto-assigned ports
```

### Test 2: Verify Service List

```bash
/tmp/aima service list
# Expected: both services listed with status "creating" and unique ports
# Check that ports are different (e.g., 8000 and 8001)
```

### Test 3: Start Both Services

```bash
# Start service A
/tmp/aima service start "$SVC_A" --wait --timeout 300 &
START_A_PID=$!

# Start service B (may need to wait for A to claim GPU memory first)
sleep 5
/tmp/aima service start "$SVC_B" --wait --timeout 300 &
START_B_PID=$!

# Wait for both
wait $START_A_PID
echo "Service A start result: $?"
wait $START_B_PID
echo "Service B start result: $?"

# Check statuses
/tmp/aima service list
# Expected: both "running" (or one may fail if GPU memory insufficient)
```

### Test 4: Stop One, Other Survives

```bash
# Stop service A
/tmp/aima service stop "$SVC_A"
echo "Stop A result: $?"

# Verify B is still running
/tmp/aima service list
docker ps --filter label=aima.engine
# Expected: service A stopped, service B still running
# B's container should be unaffected
```

### Test 5: Port Reuse After Stop

```bash
# Get the port that service A was using
PORT_A=$(/tmp/aima service list --output json | jq -r ".items[] | select(.id==\"$SVC_A\") | .port")

# Create a new service — it should be able to use the freed port
/tmp/aima model create --name "model-gamma" --source ollama --repo "gemma:2b"
MODEL_C=$(/tmp/aima model list --output json | jq -r '.items[-1].id')
/tmp/aima service create --model-id "$MODEL_C" --engine-type vllm
SVC_C=$(/tmp/aima service list --output json | jq -r '.items[-1].id')

# BUG WATCH: Bug #31 — port may not be immediately released after container stop
# Docker proxy may hold the port briefly
sleep 5

/tmp/aima service start "$SVC_C" --wait --timeout 300
# Expected: starts successfully, possibly on the same port as the stopped service A
```

### Test 6: ServiceID Format

```bash
# Verify ServiceID format
/tmp/aima service list --output json | jq -r '.items[].id'
# Expected format: svc-{engine}-{model-hash} (e.g., svc-vllm-a1b2c3d4)
# Verify with ServiceID parser
```

### Test 7: Service Recommend

```bash
curl -s http://localhost:9090/api/service/recommend | jq .
# Expected: recommendations based on available GPU memory and running services
```

## Cleanup

```bash
# Stop all services
/tmp/aima service stop "$SVC_B" 2>/dev/null
/tmp/aima service stop "$SVC_C" 2>/dev/null

# Verify no orphan containers
docker ps --filter label=aima.engine

kill $AIMA_PID 2>/dev/null
```

## Hints for the Operator

- Port allocation: `pkg/infra/provider/hybrid_engine_provider.go` (portCounter)
- ServiceID: `pkg/unit/service/service_id.go`
- Service lifecycle: `pkg/unit/service/commands.go`
- GPU memory: vLLM's `--gpu-memory-utilization` flag controls per-service allocation
- Docker labels: `aima.engine` and `aima.service` for container identification

## Known Pitfalls

- **Port counter not persisted**: The `portCounter` starts at the base port (8000) on every restart. If containers from a previous session are still running, port conflicts occur. The counter should be recovered from the ServiceStore.
- **Bug #31 — Port release delay**: Docker's proxy process may hold a port for a few seconds after container stop. Starting a new service immediately after stopping one may hit "port already allocated". A brief sleep or retry may be needed.
- **GPU memory exhaustion**: Running two vLLM instances on the same GPU requires careful memory allocation. If both try to use 90% of VRAM, the second will OOM. Check `gpu_memory_utilization` settings.
- **Service isolation**: Stopping a Docker container should only affect its service. Verify that the stop command targets the correct container (by label and service ID) and doesn't accidentally affect siblings.
- **Concurrent start race**: Starting two services simultaneously may race on port allocation. The `sync.Mutex` on `portCounter` should prevent this, but verify.

## Key Code to Inspect

| File | What to Check |
|------|---------------|
| `pkg/infra/provider/hybrid_engine_provider.go` | Port allocation, container creation |
| `pkg/unit/service/service_id.go` | ServiceID parsing and formatting |
| `pkg/unit/service/commands.go` | Create, start, stop lifecycle |
| `pkg/unit/service/store.go` | Service persistence |
| `pkg/config/config.go` | Port range configuration |

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Ports assigned to each service
- GPU memory usage after both services running
- Whether port reuse works after service stop
