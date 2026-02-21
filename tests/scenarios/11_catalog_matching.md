# Scenario 11: Catalog Recipe Matching Algorithm

**Difficulty**: Medium
**Estimated Duration**: 5 minutes
**Prerequisites**: Scenario 1 completed (binary built), HTTP server running
**Tier**: 1 — Foundation (no Docker/GPU required)

## User Story

"I manage a fleet of machines with different GPUs — some NVIDIA, some AMD, different VRAM sizes. I want AIMA's catalog to recommend the best deployment recipe for each machine based on its hardware. I need to verify the scoring algorithm actually works: does an NVIDIA-specific recipe score higher on an NVIDIA machine? Does VRAM filtering work correctly?"

## Success Criteria

1. [ ] 4 recipes created successfully: NVIDIA-optimized, AMD-optimized, CPU-only, and generic GPU
2. [ ] Query with NVIDIA GPU + 24GB VRAM correctly ranks the NVIDIA recipe highest
3. [ ] Exact model match (e.g., "GB10") outranks vendor-only match (e.g., "NVIDIA" without model)
4. [ ] VRAM boundary enforced: recipe requiring 48GB VRAM excluded from 24GB query
5. [ ] AMD query gives 0 score for NVIDIA-only recipe
6. [ ] `catalog.validate_recipe` rejects recipe with missing required fields
7. [ ] `catalog.apply_recipe` returns a deployment plan (may not actually deploy)

## Environment Setup

```bash
# Start AIMA HTTP server
/tmp/aima start &
AIMA_PID=$!
sleep 2
```

### Test 1: Create 4 Recipes

```bash
# Recipe 1: NVIDIA-optimized for 24GB
curl -s -X POST http://localhost:9090/api/catalog/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "vllm-nvidia-24gb",
    "description": "vLLM optimized for NVIDIA 24GB GPUs",
    "engine_type": "vllm",
    "hardware": {
      "gpu_vendor": "NVIDIA",
      "gpu_model": "GB10",
      "gpu_arch": "Blackwell",
      "min_vram_gb": 16,
      "max_vram_gb": 24,
      "os": "linux"
    },
    "config": {
      "gpu_memory_utilization": 0.90,
      "max_model_len": 8192,
      "quantization": "fp16"
    }
  }' | jq .

# Recipe 2: AMD-optimized
curl -s -X POST http://localhost:9090/api/catalog/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "vllm-amd-rocm",
    "description": "vLLM with ROCm for AMD GPUs",
    "engine_type": "vllm",
    "hardware": {
      "gpu_vendor": "AMD",
      "gpu_arch": "RDNA3",
      "min_vram_gb": 16,
      "os": "linux"
    },
    "config": {
      "gpu_memory_utilization": 0.85,
      "quantization": "fp16"
    }
  }' | jq .

# Recipe 3: CPU-only (no GPU requirement)
curl -s -X POST http://localhost:9090/api/catalog/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ollama-cpu",
    "description": "Ollama for CPU-only inference",
    "engine_type": "ollama",
    "hardware": {
      "os": "linux"
    },
    "config": {
      "num_threads": 8
    }
  }' | jq .

# Recipe 4: Generic GPU (high VRAM requirement)
curl -s -X POST http://localhost:9090/api/catalog/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "vllm-large-model",
    "description": "vLLM for 48GB+ GPUs running large models",
    "engine_type": "vllm",
    "hardware": {
      "gpu_vendor": "NVIDIA",
      "min_vram_gb": 48,
      "os": "linux"
    },
    "config": {
      "gpu_memory_utilization": 0.95,
      "max_model_len": 32768,
      "quantization": "awq"
    }
  }' | jq .
```

### Test 2: Match NVIDIA 24GB Query

```bash
curl -s "http://localhost:9090/api/catalog/match?gpu_vendor=NVIDIA&gpu_model=GB10&gpu_arch=Blackwell&vram_gb=24&os=linux" | jq .
# Expected: vllm-nvidia-24gb ranked first (score: vendor 40 + model 30 + arch 15 + VRAM 10 + OS 5 = 100)
# BUG WATCH: vram_gb is passed as string "24" in GET query params — toInt() may fail silently
```

### Test 3: Model Match vs Vendor-Only Match

```bash
# Create a vendor-only NVIDIA recipe (no model specified)
curl -s -X POST http://localhost:9090/api/catalog/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "vllm-nvidia-generic",
    "description": "Generic NVIDIA recipe",
    "engine_type": "vllm",
    "hardware": {
      "gpu_vendor": "NVIDIA",
      "os": "linux"
    },
    "config": {}
  }' | jq .

# Now match — the GB10-specific recipe should score higher
curl -s "http://localhost:9090/api/catalog/match?gpu_vendor=NVIDIA&gpu_model=GB10&vram_gb=24&os=linux" | jq '.items[0].name, .items[1].name'
# Expected: vllm-nvidia-24gb first (model match +30), vllm-nvidia-generic second (no model match)
```

### Test 4: VRAM Boundary — Exclude Recipe Requiring 48GB

```bash
curl -s "http://localhost:9090/api/catalog/match?gpu_vendor=NVIDIA&vram_gb=24&os=linux" | jq '.items[] | select(.name == "vllm-large-model")'
# Expected: empty — vllm-large-model requires min_vram_gb=48, should be excluded from 24GB query
```

### Test 5: AMD Query — NVIDIA Recipe Gets 0

```bash
curl -s "http://localhost:9090/api/catalog/match?gpu_vendor=AMD&gpu_arch=RDNA3&vram_gb=24&os=linux" | jq '.items[] | .name, .score'
# Expected: vllm-amd-rocm ranked first; NVIDIA-specific recipes should have 0 vendor score
```

### Test 6: Validate Recipe — Missing Required Fields

```bash
curl -s -X POST http://localhost:9090/api/catalog/validate \
  -H "Content-Type: application/json" \
  -d '{
    "name": "",
    "engine_type": ""
  }' | jq .
# Expected: validation error about missing required fields (name, engine_type)
```

### Test 7: Apply Recipe

```bash
RECIPE_ID="<vllm-nvidia-24gb recipe ID from test 1>"
curl -s -X POST http://localhost:9090/api/catalog/apply \
  -H "Content-Type: application/json" \
  -d "{\"recipe_id\": \"$RECIPE_ID\", \"model_id\": \"test-model\"}" | jq .
# Expected: returns a deployment plan (service creation parameters)
# Note: This likely does NOT trigger actual deployment — agent must call service.create manually
```

## Cleanup

```bash
kill $AIMA_PID 2>/dev/null
```

## Hints for the Operator

- Catalog matching code: `pkg/unit/catalog/queries.go`
- Scoring weights: GPU vendor (+40), GPU model (+30), GPU arch (+15), VRAM (+10), OS (+5)
- Recipe CRUD: `pkg/unit/catalog/commands.go`
- Built-in recipes: `catalog/engines/` (embedded via `catalog/embed.go`)
- HTTP routes: check `pkg/gateway/routes.go` for catalog endpoints

## Known Pitfalls

- **HIGH CONFIDENCE BUG — `catalog.match` GET string conversion**: When `vram_gb=24` is passed as a GET query parameter, it arrives as string `"24"`. The `toInt()` conversion in `pkg/unit/catalog/queries.go:143-144` may fail silently, causing the VRAM score to always be 0. This is the #1 predicted bug for this scenario.
- **Memory-only store**: Catalog store is in-memory. All recipes will be lost on restart. The embedded YAML recipes may be re-loaded but custom recipes will be gone.
- **`catalog.apply_recipe` scope**: This command returns a deployment plan but does NOT actually call `service.create`. The agent must interpret the plan and make separate service calls. This is by design but may surprise users.
- **Scoring edge case**: If two recipes tie on score, the ordering is undefined. Check if there's a tiebreaker (e.g., creation time, name).
- **Empty match**: If no recipes match the query, the response should be an empty list (not an error).

## Key Code to Inspect

| File | What to Check |
|------|---------------|
| `pkg/unit/catalog/queries.go` | Match scoring algorithm, `toInt()` conversion |
| `pkg/unit/catalog/commands.go` | Recipe CRUD, validation |
| `pkg/unit/catalog/store.go` | Memory vs persistent store |
| `pkg/gateway/routes.go` | Catalog route definitions |
| `catalog/embed.go` | Embedded recipe loading |

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Whether the `toInt()` VRAM bug was confirmed
- Actual scoring outputs for each match query
