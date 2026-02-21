# Scenario 6: Service Lifecycle Stress -- Container Churn

**Difficulty**: Medium
**Estimated Duration**: 10 minutes
**Prerequisites**: Scenario 2 completed (model registered, service creation understood)

## User Story

"I want to make sure AIMA can handle rapid service restarts without getting confused. Start a service, stop it, start it again -- do this 5 times in a row. I need to know that every restart gives me a clean container and that the state tracking doesn't get corrupted."

## Success Criteria

1. [ ] Service can be started and stopped 5 times in sequence without errors
2. [ ] Each restart creates a clean Docker container (no reuse of stale state)
3. [ ] Service status is correct after each operation (running after start, stopped after stop)
4. [ ] No orphan Docker containers left after the final stop
5. [ ] Port allocation is consistent across restarts (same service gets same port)

## Environment Setup

- A model must already be registered (from Scenario 2, or create a new one)
- Ensure a clean starting state: no running AIMA services, no orphan containers
- Use `docker ps --filter label=aima.engine` to verify no AIMA containers exist before starting

## Hints for the Operator

This is a stress test. The basic loop is:

1. Start the service
2. Verify it reaches running/healthy status
3. Stop the service
4. Verify it is fully stopped (container removed)
5. Repeat 5 times

Relevant commands:
- `aima service start --wait --timeout 600`
- `aima service list`
- `aima service stop`
- `docker ps --filter label=aima.engine` (direct Docker check)
- `docker ps -a --filter label=aima.engine` (include stopped containers)

Note: Each start-wait cycle takes 5-7 minutes for vLLM model loading. To speed up, you could use a smaller timeout and not wait for full health -- just verify the container starts.

## Known Pitfalls

- **Bug #11**: `service.stop` had a key mismatch -- it passed the full serviceID where only the engine type was expected. The fix parses `svc-{engineType}-{hash}` correctly.
- **Bug #14**: Docker restart policy (`unless-stopped`) caused containers to auto-restart after being stopped. The fix removed the restart policy -- AIMA manages retries, not Docker.
- **Container name conflicts**: Docker may reject creating a new container if an old one with the same name still exists. AIMA should handle this by removing the old container first.
- **Docker rm timing**: After `docker stop`, the container may need a moment before `docker rm` succeeds. AIMA uses `context.Background()` for the rm step to avoid timeout issues.
- **Port reuse**: If a stopped container hasn't fully released its port, the next start may fail with "port already in use". A brief pause between stop and start helps.

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Unexpected observations
