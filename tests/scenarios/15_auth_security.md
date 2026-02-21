# Scenario 15: Auth Middleware Wiring

**Difficulty**: Easy
**Estimated Duration**: 5 minutes
**Prerequisites**: Scenario 1 completed (binary built), HTTP server running
**Tier**: 2 — API Surface

## User Story

"I'm deploying AIMA on a shared network and I need to ensure only authorized users can access it. The docs mention 3-tier authentication (Optional/Recommended/Forced) with Bearer tokens. I want to verify: is auth actually enforced? Can I configure API keys? What happens when I send requests without a token?"

## Success Criteria

1. [ ] Without auth enabled, all endpoints work without any token (baseline)
2. [ ] With API key configured, POST requests without Bearer token return 401
3. [ ] Correct Bearer token in Authorization header → request succeeds (200)
4. [ ] GET queries (Optional auth level) work without token even when auth is enabled
5. [ ] `remote.exec` (Forced auth level) requires token regardless of configuration
6. [ ] Rate limiting gap confirmed: `TokenBucketLimiter` code exists but is never instantiated
7. [ ] Auth enable gap confirmed: no TOML config field to enable auth — only settable in code

## Environment Setup

```bash
# Test 1 first: baseline without auth
/tmp/aima start &
AIMA_PID=$!
sleep 2
```

### Test 1: Baseline — No Auth

```bash
# All endpoints should work without any auth
curl -s -w "\nHTTP %{http_code}\n" http://localhost:9090/api/model/list
curl -s -w "\nHTTP %{http_code}\n" -X POST http://localhost:9090/api/model/create \
  -H "Content-Type: application/json" \
  -d '{"name": "test-model", "source": "ollama", "repo": "tinyllama"}'
# Expected: both return 200
```

### Test 2: Try to Enable Auth

```bash
kill $AIMA_PID 2>/dev/null
sleep 1

# Try setting auth via config
cat > /tmp/aima-auth.toml << 'EOF'
[auth]
enabled = true
api_keys = ["test-key-12345", "backup-key-67890"]
EOF

AIMA_CONFIG=/tmp/aima-auth.toml /tmp/aima start &
AIMA_PID=$!
sleep 2

# BUG WATCH: The [auth] section may not be recognized in config.go
# Check if auth is actually enabled
curl -s -w "\nHTTP %{http_code}\n" -X POST http://localhost:9090/api/model/create \
  -H "Content-Type: application/json" \
  -d '{"name": "test-model", "source": "ollama", "repo": "tinyllama"}'
# If auth is properly enabled: 401
# If auth config field doesn't exist: 200 (auth not enabled despite config)
```

### Test 3: Try Auth via Environment Variables

```bash
kill $AIMA_PID 2>/dev/null
sleep 1

# Try env var approach
AIMA_AUTH_ENABLED=true \
AIMA_API_KEYS="env-key-11111,env-key-22222" \
/tmp/aima start &
AIMA_PID=$!
sleep 2

# Test without token
curl -s -w "\nHTTP %{http_code}\n" -X POST http://localhost:9090/api/model/create \
  -H "Content-Type: application/json" \
  -d '{"name": "test-model"}'
# Expected: 401 if env vars work, 200 if they don't

# Test with correct Bearer token
curl -s -w "\nHTTP %{http_code}\n" -X POST http://localhost:9090/api/model/create \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer env-key-11111" \
  -d '{"name": "test-model"}'
# Expected: 200 if auth works
```

### Test 4: Auth Levels — Optional (GET queries)

```bash
# GET endpoints with Optional auth level should work without token
curl -s -w "\nHTTP %{http_code}\n" http://localhost:9090/api/model/list
curl -s -w "\nHTTP %{http_code}\n" http://localhost:9090/api/engine/list
curl -s -w "\nHTTP %{http_code}\n" http://localhost:9090/api/service/list
# Expected: all return 200 even without Bearer token
# (Optional level = auth checked but not required)
```

### Test 5: Auth Levels — Forced (remote.exec)

```bash
# remote.exec should ALWAYS require auth (Forced level)
curl -s -w "\nHTTP %{http_code}\n" -X POST http://localhost:9090/api/remote/exec \
  -H "Content-Type: application/json" \
  -d '{"command": "echo test"}'
# Expected: 401 (Forced auth — requires token regardless)

# Even with token, remote.exec should validate
curl -s -w "\nHTTP %{http_code}\n" -X POST http://localhost:9090/api/remote/exec \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer env-key-11111" \
  -d '{"command": "echo test"}'
# Expected: 200 (authorized)
```

### Test 6: Rate Limiting

```bash
# Check if rate limiting is active
for i in $(seq 1 50); do
  curl -s -o /dev/null -w "%{http_code} " http://localhost:9090/api/model/list
done
echo ""
# Expected: all 200 (rate limiting not enforced)
# Verify: TokenBucketLimiter exists in code but is never instantiated
```

### Test 7: Auth Middleware Code Analysis

```bash
# Check if auth middleware is wired into HTTP handlers
grep -n "auth" /home/qujing/projects/ai-inference-managed-by-ai/pkg/cli/start.go
grep -n "AuthMiddleware" /home/qujing/projects/ai-inference-managed-by-ai/pkg/gateway/server.go
grep -n "EnableAuth" /home/qujing/projects/ai-inference-managed-by-ai/pkg/config/config.go

# Expected findings:
# - auth middleware code exists in pkg/gateway/middleware/auth.go
# - BUT it's not wired into the HTTP handler chain
# - No EnableAuth field in config.go
```

## Cleanup

```bash
kill $AIMA_PID 2>/dev/null
rm -f /tmp/aima-auth.toml
```

## Hints for the Operator

- Auth middleware: `pkg/gateway/middleware/auth.go`
- Auth levels defined: Optional (queries), Recommended (commands), Forced (remote.exec, delete)
- Config: `pkg/config/config.go` (check for AuthConfig or EnableAuth field)
- HTTP server setup: `pkg/cli/start.go` (where middleware is registered)
- Rate limiter: `pkg/gateway/middleware/` (TokenBucketLimiter)

## Known Pitfalls

- **HIGH CONFIDENCE BUG — No TOML field for auth**: The `EnableAuth` flag exists in the middleware code but there's no corresponding field in the config.go struct. Auth can only be enabled by modifying Go code directly, not via configuration.
- **Auth middleware NOT wired**: The auth middleware code exists and is tested, but it's not registered in the HTTP handler chain in `pkg/cli/start.go`. All requests bypass auth entirely.
- **Forced routes permanently locked**: If auth IS enabled but no API keys are configured, `Forced` level routes (like `remote.exec`) become permanently inaccessible — they require a valid token but there are no valid tokens.
- **Rate limiting never instantiated**: `TokenBucketLimiter` exists as code but is never created or registered as middleware. Rate limiting is effectively disabled.
- **Bearer token format**: The middleware expects `Authorization: Bearer <token>`. Other formats (Basic, API-Key header) may not be supported.

## Key Code to Inspect

| File | What to Check |
|------|---------------|
| `pkg/gateway/middleware/auth.go` | Auth levels, token validation, EnableAuth field |
| `pkg/config/config.go` | Config struct — is there an auth section? |
| `pkg/cli/start.go` | HTTP handler chain — is auth middleware registered? |
| `pkg/gateway/server.go` | Middleware registration order |
| `pkg/gateway/middleware/auth_test.go` | Tests exist but middleware may not be wired |

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Whether auth can be enabled via ANY mechanism (config, env var, CLI flag)
- Whether rate limiting code exists but is never used
- Full list of auth-level mappings found in code (which endpoints are Optional/Recommended/Forced)
