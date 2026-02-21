# Scenario 5: Agent Self-Service -- Agent Deploys Model E2E

**Difficulty**: Hard
**Estimated Duration**: 15 minutes
**Prerequisites**: Scenario 4 completed (agent working with Kimi API); Scenario 2's service stopped (clean slate)

## User Story

"I want to tell the AI agent: 'Deploy the Qwen3-Coder model and let me chat with it.' The agent should figure out ALL the steps -- finding the model, creating a service, starting it, waiting for it to be ready, and then running inference -- without me having to hold its hand. I just want to give it one instruction and sit back."

## Success Criteria

1. [ ] Agent discovers the available model (or registers it from the known path)
2. [ ] Agent creates an inference service for the model
3. [ ] Agent starts the service and waits for it to become healthy
4. [ ] Agent runs a test inference and returns the result to the user
5. [ ] All of the above happens within ONE agent conversation (no manual CLI intervention)

## Environment Setup

- Same Kimi API configuration as Scenario 4
- Model at `/mnt/data/models/Qwen3-Coder-Next-FP8`
- Ensure Scenario 2's service is stopped: `aima service stop <service-id>` or check `docker ps`
- This is a clean-slate test: the agent should handle everything from scratch

## Hints for the Operator

- Start with: `aima agent chat`
- Give a single high-level instruction like: "Deploy the Qwen3 model at /mnt/data/models/Qwen3-Coder-Next-FP8 and test it with a simple coding question"
- The agent has access to these MCP tools: `model.create`, `model.list`, `service.create`, `service.start`, `service.list`, `inference.chat`, and others
- Watch the agent's tool calls -- it should chain multiple operations together
- Be patient: model loading takes 5-7 minutes. The agent needs to handle this async wait.

## Known Pitfalls

- **Tool schema awareness**: The agent may not know the exact parameters for each tool call. Check that MCP adapter tool definitions include all required fields (model_id, model_path, engine_type, etc.).
- **Async service start**: The agent needs to handle the fact that `service.start` returns quickly but the service takes minutes to become healthy. It should poll `service.list` or similar to check status.
- **Token budget**: Deploying a model is a multi-turn conversation with 5-10 tool calls. The Kimi API has token limits per request -- the conversation may hit them.
- **Agent timeout**: Bug #10 set the agent chat timeout to 10 minutes. A full deploy-and-infer flow may push this limit, especially with model loading.
- **Bug #13**: `engine.start` InputSchema must include `model_id` and `model_path` fields. Without these, the agent can't tell the engine where the model is.
- **Bug #17**: If `--timeout` doesn't propagate, the service start will time out at 30 seconds even though the model needs 5-7 minutes.

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Unexpected observations
