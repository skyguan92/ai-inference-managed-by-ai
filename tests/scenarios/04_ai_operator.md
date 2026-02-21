# Scenario 4: AI Operator -- Natural Language Management

**Difficulty**: Medium
**Estimated Duration**: 10 minutes
**Prerequisites**: Scenario 1 completed; Kimi API key configured

## User Story

"I don't want to memorize CLI commands. I just want to type things like 'show me my GPU' or 'list all models' in natural language, and AIMA should figure out what to do. I heard there's an AI agent built in -- let me try talking to it."

## Success Criteria

1. [ ] Agent is configured with a working LLM backend (Kimi API)
2. [ ] Can start an agent chat session without errors
3. [ ] Can ask "What devices do I have?" and get a real answer showing GPU info
4. [ ] Can ask "Show me all models" and see the model list
5. [ ] Agent translates natural language to AIMA tool calls automatically (visible in output)
6. [ ] Conversation context is maintained across turns (agent remembers what was discussed)

## Environment Setup

Configure the Kimi API in `~/.aima/config.toml`:

```toml
[agent]
llm_provider = "openai"
llm_model = "kimi-for-coding"
llm_base_url = "https://api.kimi.com/coding/v1"
llm_api_key = "set via AIMA_LLM_API_KEY env var"
llm_user_agent = "claude-code/1.0"
```

Set the API key:

```bash
export AIMA_LLM_API_KEY="<your-kimi-api-key>"
```

The `llm_user_agent` field is critical -- the Kimi coding API blocks requests without the `claude-code/1.0` User-Agent header.

## Hints for the Operator

- `aima agent chat` -- start an interactive agent conversation
- Type natural language queries into the chat REPL
- The agent has access to all AIMA tools via MCP (model.list, device.list, service.list, etc.)
- Try multi-turn conversations: ask about devices, then ask about models, then ask to compare them

## Known Pitfalls

- **Bug #5**: CLI may swallow error details. If the agent fails, check `resp.Error.Details` for the real error message.
- **Bug #7**: Kimi returns `reasoning_content` in tool call messages. AIMA's OpenAI client must handle this extra field without crashing.
- **Bug #8**: Kimi requires the `User-Agent: claude-code/1.0` header. Without `llm_user_agent` in config, the API returns a 403 or similar error.
- **Bug #10**: Agent chat needs a long timeout (10+ minutes for multi-turn). The gateway timeout for agent requests is set to 10 minutes, but if the binary is old, it may still use the 30-second default.
- **No API key**: If `AIMA_LLM_API_KEY` is not set and no API key is in the config file, AIMA tries local LLM auto-detection. This probes for running OpenAI-compatible services and Ollama. It may or may not find a working backend.
- **Network issues**: The Kimi API is a cloud service. Network connectivity from the ARM64 machine to `api.kimi.com` is required.

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Unexpected observations
