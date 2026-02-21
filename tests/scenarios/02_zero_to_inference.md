# Scenario 2: From Zero to Inference -- Deploy Qwen3-Coder

**Difficulty**: Hard
**Estimated Duration**: 15 minutes
**Prerequisites**: Scenario 1 completed (binary built, device detected)

## User Story

"I have a Qwen3-Coder-Next-FP8 model sitting in `/mnt/data/models/Qwen3-Coder-Next-FP8`. It's about 75GB across 40 shards. I want to start chatting with it. Just make it work -- I don't care about the details, I just want to ask it to write some code and get a response."

## Success Criteria

1. [ ] Model is registered in AIMA (appears in `model list`)
2. [ ] A vLLM inference service is running in a Docker container
3. [ ] Service reaches "healthy" status (vLLM `/health` endpoint responds)
4. [ ] Can send a chat message (e.g., "Write a Python hello world") and get a code response
5. [ ] Can stop the service cleanly when done (container removed)

## Environment Setup

- Model path: `/mnt/data/models/Qwen3-Coder-Next-FP8` (75GB, 40 shards, FP8 quantization)
- Docker image: `zhiwen-vllm:0128` (must already be available locally -- no pull needed)
- GPU: NVIDIA GB10 with ~24GB memory
- **Important**: This model takes 5-7 minutes to load into GPU memory. Timeouts must be generous.
- Ensure no leftover services from previous runs: check `aima service list` and `docker ps`

## Hints for the Operator

These commands might be useful (in no particular order):

- `aima model create` -- register a model from a local path
- `aima service create` -- create a service for a registered model
- `aima service start --wait --timeout 600` -- start and wait for healthy status
- `aima service logs <service-id>` -- watch Docker output for progress
- `aima inference chat` -- send a chat message to a running service
- `aima service stop` -- stop the service and remove the container
- `docker ps` -- verify container state directly

The `--timeout 600` flag gives 10 minutes for the model to load. Without it, the default 30-second timeout will expire long before vLLM finishes loading.

## Known Pitfalls

- **Bug #15**: Engine YAML needs `startup.command` field for vLLM entrypoint. Without it, the container tries to execute raw arguments as a command (e.g., `exec --gpu-memory-utilization 0.95` instead of `python -m vllm.entrypoints.openai.api_server --gpu-memory-utilization 0.95`).
- **Bug #16**: YAML assets must be embedded via `go:embed`. If the binary is run outside the project directory, it won't find `catalog/engines/*.yaml` unless they are embedded.
- **Bug #17**: The `--timeout` flag on `service start` must propagate all the way to the gateway HTTP request. In older versions, the CLI waited for the timeout but the gateway request used the 30-second default.
- **Service appears to hang**: Use `aima service logs <id>` to see Docker output. vLLM prints loading progress that confirms it's working.
- **Health check timing**: vLLM's `/health` endpoint may take 5+ minutes to return 200 for large models. AIMA polls this endpoint during `--wait`.
- **GPU memory**: With `--gpu-memory-utilization 0.95`, vLLM uses nearly all 24GB. If another process holds GPU memory, loading will fail with CUDA OOM.

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Unexpected observations
