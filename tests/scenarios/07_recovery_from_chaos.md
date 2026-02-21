# Scenario 7: Recovery from Chaos -- Orphan Container Cleanup

**Difficulty**: Medium
**Estimated Duration**: 10 minutes
**Prerequisites**: Scenario 1 completed (binary built, familiar with CLI)

## User Story

"Something went wrong last time and there are leftover Docker containers from a previous AIMA session. I don't want to manually run docker commands to clean up. I just want to use AIMA normally and have it deal with the mess."

## Success Criteria

1. [ ] Can detect orphan AIMA containers (containers with `aima.engine` label that AIMA didn't start this session)
2. [ ] AIMA service operations work even when stale containers exist from previous sessions
3. [ ] Starting a new service for the same engine type replaces the orphaned container
4. [ ] No manual `docker rm` is needed -- AIMA handles cleanup automatically
5. [ ] After cleanup, `docker ps --filter label=aima.engine` shows only actively managed containers

## Environment Setup

First, deliberately create orphan containers to simulate a crashed session:

```bash
# Create a fake orphan AIMA container (vllm type)
docker run -d --name aima-orphan-vllm --label aima.engine=vllm busybox sleep 3600

# Create another orphan (tts type)
docker run -d --name aima-orphan-tts --label aima.engine=tts busybox sleep 3600
```

Verify orphans exist:

```bash
docker ps --filter label=aima.engine
```

Now try normal AIMA operations and see if it copes.

## Hints for the Operator

- `aima service list` -- does AIMA know about the orphan containers?
- `aima service start` -- starting a new vLLM service should deal with the orphan
- `docker ps --filter label=aima.engine` -- check what containers exist at each step
- `aima service stop` -- stopping should clean up the active container

The key question: when AIMA starts a new service and finds an existing container with the same `aima.engine` label, does it clean up the old one first?

## Known Pitfalls

- **Bug #11 fix**: Added Docker label fallback in `HybridEngineProvider.Stop()`. When a container isn't found in the in-memory map (because AIMA restarted), it falls back to `ListContainers(ctx, {"aima.engine": engineType})` to find and stop orphaned containers.
- **Container in "restarting" state**: If an orphan container has a restart policy (Bug #14 was fixed to remove this), it may be in a "restarting" state that blocks port allocation. Check `docker inspect` for restart policy.
- **Context for rm**: The `docker rm` step uses `context.Background()` instead of the request context. This prevents timeout-related cleanup failures.
- **Multiple orphans**: If there are multiple orphaned containers for the same engine type, AIMA should handle all of them, not just the first one found.
- **busybox vs real image**: The orphan containers use `busybox` (a tiny image), not real engine images. This is intentional -- we're testing AIMA's cleanup logic, not the engine itself.

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Unexpected observations
