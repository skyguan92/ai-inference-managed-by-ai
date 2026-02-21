# Scenario 3: Multimodal Service Stack -- ASR + TTS

**Difficulty**: Medium
**Estimated Duration**: 10 minutes
**Prerequisites**: Scenario 1 completed (binary built, familiar with CLI)

## User Story

"I want to set up a speech pipeline -- speech-to-text and text-to-speech. I don't need a GPU for these, CPU should be fine. I want both running at the same time so I can eventually pipe audio through them."

## Success Criteria

1. [ ] Two models registered: one with ASR type, one with TTS type
2. [ ] Two services created: one for ASR, one for TTS (different engine types)
3. [ ] Both services can be started (or fail gracefully with a clear error if Docker images are missing)
4. [ ] Services use different ports (no port conflict)
5. [ ] Can list both services and see their individual status
6. [ ] Can stop both services cleanly

## Environment Setup

- No real ASR/TTS models or Docker images are required on the test machine
- Engine types `asr` and `tts` are defined in `catalog/engines/` YAML files
- Port allocation is automatic based on service ID
- If Docker images for ASR/TTS don't exist locally, AIMA should fail with a clear error (not a cryptic crash)
- Stop any running services from Scenario 2 first to free resources

## Hints for the Operator

Commands that might help:

- `aima model create --type asr` -- register an ASR-type model
- `aima model create --type tts` -- register a TTS-type model
- `aima service create` -- create services for each model
- `aima service start` -- start each service
- `aima service list` -- view all services and their status
- `aima service stop` -- stop services when done

This scenario is partly about verifying that AIMA handles multi-engine scenarios correctly -- different engine types, concurrent containers, separate port allocation.

## Known Pitfalls

- **Bug #11**: ServiceID parsing uses the format `svc-{engineType}-{hash}`. The stop command must parse the engine type correctly from this ID. An older bug caused `service.stop` to pass the full service ID where only the engine type was expected.
- **Port conflicts**: If services from previous runs weren't cleaned up, ports may be occupied. Check `docker ps` and clean up stale containers.
- **Missing Docker images**: The test machine may not have ASR/TTS Docker images. This is expected -- the goal is to verify that AIMA handles this gracefully rather than crashing.
- **Engine YAML**: ASR and TTS engine configurations come from `catalog/engines/asr/*.yaml` and `catalog/engines/tts/*.yaml`. These define default images, ports, and arguments.

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Unexpected observations
