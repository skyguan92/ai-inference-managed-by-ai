# Scenario 1: Hello AIMA -- First Contact

**Difficulty**: Easy
**Estimated Duration**: 5 minutes
**Prerequisites**: None (this is the starting point)

## User Story

"I just installed AIMA on my ARM64 Linux machine with an NVIDIA GPU. I've never used it before. I want to poke around and see what it can do -- what hardware I have, what models are available, if anything is already running. Basically just get my bearings."

## Success Criteria

1. [ ] Can build the AIMA binary from source and run it
2. [ ] Can see the CLI version and list of available commands via help
3. [ ] Can detect GPU hardware (NVIDIA GB10, ~24GB memory)
4. [ ] Can list models (may be empty or show previously imported ones)
5. [ ] Can list services and see their status (may be empty)
6. [ ] Can check device metrics (temperature, utilization, memory usage)
7. [ ] Can see what engine types are available (vllm, asr, tts)

## Environment Setup

- SSH into `qujing@100.105.58.16`
- Project source at `/home/qujing/projects/ai-inference-managed-by-ai`
- Build binary:
  ```bash
  cd /home/qujing/projects/ai-inference-managed-by-ai
  GOPROXY=https://goproxy.cn,direct /usr/local/go/bin/go build -o /tmp/aima ./cmd/aima
  ```

## Hints for the Operator

The following commands might be relevant (not a step-by-step guide):

- `aima --help` or `aima -h`
- `aima version`
- `aima device list`
- `aima device metrics`
- `aima model list`
- `aima service list`
- `aima engine list`

Explore subcommands with `aima <command> --help` to discover options.

## Known Pitfalls

- **Build failure**: If `go build` fails due to network issues, ensure `GOPROXY=https://goproxy.cn,direct` is set (direct access to golang.org times out from China)
- **Device detection**: May fail if NVIDIA drivers are not loaded. Check with `nvidia-smi` first.
- **Stale binary**: If the binary was built from older source, rebuild. The binary must match current source.
- **PATH**: The binary is at `/tmp/aima`, not in PATH. Use the full path or alias it.

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Unexpected observations
