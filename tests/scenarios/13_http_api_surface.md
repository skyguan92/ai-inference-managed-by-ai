# Scenario 13: HTTP REST API Completeness

**Difficulty**: Medium
**Estimated Duration**: 8 minutes
**Prerequisites**: Scenario 1 completed (binary built), HTTP server running, Docker available
**Tier**: 2 — API Surface

## User Story

"I'm building a web dashboard that talks to AIMA's REST API. I need to verify that all 13 domain endpoints work, that error handling is consistent, and that the API is robust against bad input — invalid JSON, missing fields, huge payloads, and concurrent requests."

## Success Criteria

1. [ ] All 13 domains respond to at least one endpoint (device, model, engine, inference, resource, service, app, pipeline, alert, remote, catalog, skill, agent)
2. [ ] Invalid JSON body returns HTTP 400 with a structured error (not `{}` or 500)
3. [ ] Empty body to `inference.chat` returns a clear error message (not a silent empty response)
4. [ ] Unknown route returns HTTP 404 with an error code
5. [ ] Responses include `Content-Type: application/json` header
6. [ ] Body size limit enforced (or gap confirmed: no `MaxBytesReader`)
7. [ ] Concurrent requests (10 simultaneous `model.list`) all succeed
8. [ ] `/health` endpoint returns HTTP 200

## Environment Setup

```bash
# Start AIMA HTTP server with Docker available
/tmp/aima start &
AIMA_PID=$!
sleep 2
```

### Test 1: All 13 Domains Respond

```bash
# Test each domain's list/detect endpoint
echo "=== Device ===" && curl -s -o /dev/null -w "%{http_code}" http://localhost:9090/api/device/detect
echo "=== Model ===" && curl -s -o /dev/null -w "%{http_code}" http://localhost:9090/api/model/list
echo "=== Engine ===" && curl -s -o /dev/null -w "%{http_code}" http://localhost:9090/api/engine/list
echo "=== Inference ===" && curl -s -o /dev/null -w "%{http_code}" http://localhost:9090/api/inference/list
echo "=== Resource ===" && curl -s -o /dev/null -w "%{http_code}" http://localhost:9090/api/resource/list
echo "=== Service ===" && curl -s -o /dev/null -w "%{http_code}" http://localhost:9090/api/service/list
echo "=== App ===" && curl -s -o /dev/null -w "%{http_code}" http://localhost:9090/api/app/list
echo "=== Pipeline ===" && curl -s -o /dev/null -w "%{http_code}" http://localhost:9090/api/pipeline/list
echo "=== Alert ===" && curl -s -o /dev/null -w "%{http_code}" http://localhost:9090/api/alert/list
echo "=== Remote ===" && curl -s -o /dev/null -w "%{http_code}" http://localhost:9090/api/remote/list
echo "=== Catalog ===" && curl -s -o /dev/null -w "%{http_code}" http://localhost:9090/api/catalog/list
echo "=== Skill ===" && curl -s -o /dev/null -w "%{http_code}" http://localhost:9090/api/skill/list
echo "=== Agent ===" && curl -s -o /dev/null -w "%{http_code}" http://localhost:9090/api/agent/list

# Expected: all return 200 (or at least not 404/500)
# Record which domains return errors vs empty successful responses
```

### Test 2: Invalid JSON → 400

```bash
curl -s -w "\nHTTP %{http_code}\n" -X POST http://localhost:9090/api/model/create \
  -H "Content-Type: application/json" \
  -d '{"this is not valid json'
# Expected: HTTP 400 with {"error": ..., "code": ...}
# BUG WATCH: bodyInputMapper may return {} instead of 400 on JSON decode error
```

```bash
# Also test with completely wrong content type
curl -s -w "\nHTTP %{http_code}\n" -X POST http://localhost:9090/api/model/create \
  -H "Content-Type: text/plain" \
  -d 'just plain text'
# Expected: HTTP 400 or 415 (Unsupported Media Type)
```

### Test 3: Empty Body to inference.chat

```bash
curl -s -w "\nHTTP %{http_code}\n" -X POST http://localhost:9090/api/inference/chat \
  -H "Content-Type: application/json" \
  -d '{}'
# Expected: clear error about missing required fields (message, model_id)
# Not: silent empty response or 200 with null
```

### Test 4: Unknown Route → 404

```bash
curl -s -w "\nHTTP %{http_code}\n" http://localhost:9090/api/nonexistent/endpoint
# Expected: HTTP 404 with structured error
```

```bash
curl -s -w "\nHTTP %{http_code}\n" http://localhost:9090/completely/wrong/path
# Expected: HTTP 404
```

### Test 5: Content-Type Header

```bash
curl -s -D - http://localhost:9090/api/model/list | head -20
# Expected: Content-Type: application/json in response headers
```

### Test 6: Body Size Limit

```bash
# Generate a 10MB payload
python3 -c "import json; print(json.dumps({'data': 'x' * 10_000_000}))" | \
  curl -s -w "\nHTTP %{http_code}\n" -X POST http://localhost:9090/api/model/create \
  -H "Content-Type: application/json" \
  -d @-
# Expected: HTTP 413 (Payload Too Large) or similar rejection
# BUG WATCH: No http.MaxBytesReader — server may accept and OOM on huge payloads
```

### Test 7: Concurrent Requests

```bash
# Fire 10 concurrent model.list requests
for i in $(seq 1 10); do
  curl -s -o /dev/null -w "%{http_code}\n" http://localhost:9090/api/model/list &
done
wait
# Expected: all 10 return 200
```

### Test 8: Health Endpoint

```bash
curl -s -w "\nHTTP %{http_code}\n" http://localhost:9090/health
# Expected: HTTP 200 with some status info
```

## Additional Probing

```bash
# Test OPTIONS (CORS preflight)
curl -s -D - -X OPTIONS http://localhost:9090/api/model/list \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: POST" | head -20
# BUG WATCH: CORS middleware exists but may not be wired

# Test method not allowed
curl -s -w "\nHTTP %{http_code}\n" -X DELETE http://localhost:9090/api/model/list
# Expected: 405 Method Not Allowed

# Test HEAD request
curl -s -o /dev/null -w "%{http_code}" -I http://localhost:9090/api/model/list
# Expected: 200 with headers but no body
```

## Cleanup

```bash
kill $AIMA_PID 2>/dev/null
```

## Hints for the Operator

- Route definitions: `pkg/gateway/routes.go` (RegisterRoutes function)
- HTTP server setup: `pkg/gateway/server.go`
- Body parsing: `bodyInputMapper` function in routes.go
- CORS middleware: `pkg/gateway/middleware/cors.go` (if it exists)
- Error response format: check if errors are consistently structured

## Known Pitfalls

- **HIGH CONFIDENCE BUG — `bodyInputMapper` returns `{}`**: On JSON decode error, `bodyInputMapper` in `pkg/gateway/routes.go:221` returns an empty map `{}` instead of an error. This means malformed JSON gets silently accepted as an empty input, leading to confusing downstream errors instead of a clean 400.
- **No `http.MaxBytesReader`**: The server likely has no body size limit. A malicious client could send a multi-GB payload and OOM the server. Check for `MaxBytesReader` usage.
- **CORS not wired**: CORS middleware code may exist but not be registered in the HTTP handler chain. Cross-origin requests from a web dashboard will fail silently.
- **InputMapper mismatches**: Some routes may have mismatches between the URL parameter names and the InputMapper key mappings. This was previously fixed for several domains but new ones (catalog, skill, agent) may still have issues.
- **Missing Content-Type**: Some responses may not include `Content-Type: application/json` header, causing issues for frontend clients.

## Key Code to Inspect

| File | What to Check |
|------|---------------|
| `pkg/gateway/routes.go` | Route definitions, bodyInputMapper, error handling |
| `pkg/gateway/server.go` | Server config, MaxBytesReader, WriteTimeout |
| `pkg/gateway/middleware/` | CORS, auth, rate limiting middleware |
| `pkg/gateway/handler.go` | Request handling, error response format |

## Reporting

After executing this scenario, report:
- For each success criterion: PASS or FAIL with evidence (command output)
- Any bugs discovered (use Bug Report Format from README)
- Time taken
- Complete list of domains that work vs. those that return errors
- Whether the `bodyInputMapper` bug was confirmed
- Whether CORS headers are present in responses
