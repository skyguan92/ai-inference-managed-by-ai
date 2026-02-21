# Scenario 12: Skill Registry and Search

**Difficulty**: Easy
**Estimated Duration**: 3 minutes
**Prerequisites**: Scenario 1 completed (binary built), HTTP server running
**Tier**: 1 — Foundation (no Docker/GPU required)

## User Story

"I'm building an AI agent that needs domain-specific knowledge about deploying ML models. I want to register 'skills' — reusable knowledge snippets — that the agent can search and use. I need CRUD operations, keyword search, and the ability to enable/disable skills without deleting them."

## Success Criteria

1. [ ] Skill created successfully with metadata (name, description, keywords, category)
2. [ ] Keyword search returns matching skill (search for a keyword → finds the skill)
3. [ ] Non-matching keyword query returns empty result (not an error)
4. [ ] Disable skill works (skill still exists but is flagged as disabled)
5. [ ] Disabled skill excluded from search results (search should only return enabled skills)
6. [ ] List shows correct metadata for all skills (including enabled/disabled status)

## Environment Setup

```bash
# Start AIMA HTTP server
/tmp/aima start &
AIMA_PID=$!
sleep 2
```

### Test 1: Create Skills

```bash
# Skill 1: GPU optimization knowledge
curl -s -X POST http://localhost:9090/api/skill/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "gpu-memory-optimization",
    "description": "Best practices for GPU memory management in vLLM deployments",
    "keywords": ["gpu", "memory", "vllm", "optimization", "cuda"],
    "category": "deployment",
    "content": "## GPU Memory Optimization\n\n1. Set gpu_memory_utilization to 0.90 for single-model deployments\n2. Use quantization (AWQ/GPTQ) for models exceeding 70% VRAM\n3. Enable KV cache optimization for long-context models"
  }' | jq .

# Skill 2: Model selection
curl -s -X POST http://localhost:9090/api/skill/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "model-selection-guide",
    "description": "How to choose the right model for different tasks",
    "keywords": ["model", "selection", "coding", "chat", "reasoning"],
    "category": "planning",
    "content": "## Model Selection Guide\n\n- Coding tasks: Qwen3-Coder, DeepSeek-Coder\n- Chat: Llama-3, Qwen-Chat\n- Reasoning: DeepSeek-R1, Qwen3-Coder-Next"
  }' | jq .

# Skill 3: Troubleshooting
curl -s -X POST http://localhost:9090/api/skill/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "container-troubleshooting",
    "description": "Common Docker container issues and fixes for AIMA services",
    "keywords": ["docker", "container", "troubleshooting", "port", "crash"],
    "category": "operations",
    "content": "## Container Troubleshooting\n\n1. Port conflict: Use aima service cleanup\n2. OOM kill: Reduce gpu_memory_utilization\n3. Slow start: Check model download progress"
  }' | jq .
```

### Test 2: Keyword Search — Matching

```bash
curl -s "http://localhost:9090/api/skill/search?query=gpu+memory" | jq .
# Expected: returns gpu-memory-optimization skill
# Check: does it match on keywords or also on name/description/content?
```

### Test 3: Keyword Search — Non-Matching

```bash
curl -s "http://localhost:9090/api/skill/search?query=kubernetes+helm" | jq .
# Expected: empty result set (not an error)
```

### Test 4: Disable Skill

```bash
SKILL_ID="<gpu-memory-optimization skill ID from test 1>"
curl -s -X POST http://localhost:9090/api/skill/disable \
  -H "Content-Type: application/json" \
  -d "{\"skill_id\": \"$SKILL_ID\"}" | jq .
# Expected: success, skill now disabled
```

### Test 5: Search Excludes Disabled

```bash
curl -s "http://localhost:9090/api/skill/search?query=gpu" | jq .
# Expected: gpu-memory-optimization NOT in results (it's disabled)
# container-troubleshooting should still appear if "gpu" is not in its keywords
```

### Test 6: List All Skills

```bash
curl -s http://localhost:9090/api/skill/list | jq '.items[] | {name, category, enabled}'
# Expected: 3 skills listed, one with enabled=false
```

## Cleanup

```bash
kill $AIMA_PID 2>/dev/null
```

## Hints for the Operator

- Skill domain code: `pkg/unit/skill/`
- Skill registry: `pkg/agent/skill/` (SkillRegistry with MatchSkills)
- Built-in skills: embedded via `//go:embed` with YAML front-matter
- Routes: check `pkg/gateway/routes.go` for skill endpoints
- Try CLI: `aima skill list`, `aima skill search --query "gpu"`

## Known Pitfalls

- **Memory-only store**: Skill store is likely in-memory. Skills created via API will be lost on restart. Built-in skills from embedded YAML files will be re-loaded.
- **Search wiring**: The search query may not be correctly wired via InputMapper in the HTTP route definition. Check if the `query` parameter is properly extracted from the URL query string.
- **Case sensitivity**: `SkillRegistry.MatchSkills()` may do case-sensitive keyword matching. Searching for "GPU" (uppercase) might not match a skill with keyword "gpu" (lowercase).
- **Content format**: Skills use YAML front-matter format. When created via API (not embedded), the content handling may differ from the file-based skill loading path.
- **Disable vs Delete**: Disabling should be a soft operation (sets a flag). Verify that disabled skills can be re-enabled and that deletion is a separate operation.

## Key Code to Inspect

| File | What to Check |
|------|---------------|
| `pkg/unit/skill/commands.go` | Skill create, enable, disable |
| `pkg/unit/skill/queries.go` | Skill search, list |
| `pkg/agent/skill/skill.go` | SkillRegistry, MatchSkills algorithm |
| `pkg/unit/skill/store.go` | Memory vs persistent store |
| `pkg/gateway/routes.go` | Skill route definitions, InputMapper |

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Whether case sensitivity in search is an issue
- Whether InputMapper correctly wires the search query parameter
