package gateway

import (
	"encoding/json"
	"net/http"
	"strings"
)

// bodyDecodeErrKey is a sentinel key used by bodyInputMapper to propagate JSON
// decode errors through the InputMapper interface without changing its signature.
const bodyDecodeErrKey = "__body_decode_error__"

type Route struct {
	Method      string
	Path        string
	Unit        string
	Type        string
	InputMapper func(r *http.Request, pathParams map[string]string) map[string]any
}

type Router struct {
	routes             []Route
	gateway            *Gateway
	pathParamExtractor *pathParamExtractor
}

func NewRouter(gateway *Gateway) *Router {
	return &Router{
		routes:             defaultRoutes(),
		gateway:            gateway,
		pathParamExtractor: newPathParamExtractor(),
	}
}

func (r *Router) AddRoute(route Route) {
	r.routes = append(r.routes, route)
}

func (r *Router) Routes() []Route {
	return r.routes
}

// corsHeaders writes CORS headers to every response so that browser clients
// can call the REST API without requiring a separate proxy.
func corsHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-Trace-ID")
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Bug #46: handle OPTIONS preflight and add CORS headers to all responses.
	corsHeaders(w)
	if req.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Bug #47: HEAD falls back to GET.
	lookupMethod := req.Method
	isHEAD := req.Method == http.MethodHead
	if isHEAD {
		lookupMethod = http.MethodGet
	}

	for _, route := range r.routes {
		if route.Method != lookupMethod {
			continue
		}

		pathParams, ok := r.pathParamExtractor.match(route.Path, req.URL.Path)
		if !ok {
			continue
		}

		if isHEAD {
			// For HEAD, delegate to GET handler but suppress body via http.ResponseWriter wrapper.
			r.handleRoute(w, req, route, pathParams)
			return
		}
		r.handleRoute(w, req, route, pathParams)
		return
	}

	// Bug #48: distinguish "path exists but wrong method" (405) from "path not found" (404).
	var allowedMethods []string
	for _, route := range r.routes {
		_, ok := r.pathParamExtractor.match(route.Path, req.URL.Path)
		if ok {
			allowedMethods = append(allowedMethods, route.Method)
		}
	}
	if len(allowedMethods) > 0 {
		w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
		writeJSONError(w, http.StatusMethodNotAllowed, ErrCodeInvalidRequest, "method not allowed: "+req.Method)
		return
	}

	writeJSONError(w, http.StatusNotFound, ErrCodeUnitNotFound, "route not found: "+req.Method+" "+req.URL.Path)
}

func (r *Router) handleRoute(w http.ResponseWriter, httpReq *http.Request, route Route, pathParams map[string]string) {
	ctx := httpReq.Context()

	// Bug #45: limit request body size for mutating methods.
	method := httpReq.Method
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		httpReq.Body = http.MaxBytesReader(w, httpReq.Body, 10<<20)
	}

	traceID := httpReq.Header.Get(HeaderTraceID)

	input := map[string]any{}
	if route.InputMapper != nil {
		input = route.InputMapper(httpReq, pathParams)
	}

	// Bug #43: detect JSON decode errors signalled by bodyInputMapper.
	if errMsg, ok := input[bodyDecodeErrKey].(string); ok {
		writeJSONError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "invalid JSON body: "+errMsg)
		return
	}

	for k, v := range pathParams {
		input[k] = v
	}

	req := &Request{
		Type:  route.Type,
		Unit:  route.Unit,
		Input: input,
		Options: RequestOptions{
			TraceID: traceID,
		},
	}

	resp := r.gateway.Handle(ctx, req)

	// Bug #44: use the existing errorToStatusCode logic (already in http_adapter.go)
	// via writeResponse, which already maps error codes to HTTP statuses.
	NewHTTPAdapter(r.gateway).writeResponse(w, resp)
}

func defaultRoutes() []Route {
	return []Route{
		{Method: http.MethodPost, Path: "/api/v2/models/pull", Unit: "model.pull", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/models/create", Unit: "model.create", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodDelete, Path: "/api/v2/models/{id}", Unit: "model.delete", Type: TypeCommand, InputMapper: modelIDInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/models", Unit: "model.list", Type: TypeQuery, InputMapper: queryInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/models/{id}", Unit: "model.get", Type: TypeQuery, InputMapper: modelIDInputMapper},

		{Method: http.MethodPost, Path: "/api/v2/inference/chat", Unit: "inference.chat", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/inference/complete", Unit: "inference.complete", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/inference/embed", Unit: "inference.embed", Type: TypeCommand, InputMapper: bodyInputMapper},

		{Method: http.MethodGet, Path: "/api/v2/devices", Unit: "device.detect", Type: TypeCommand, InputMapper: emptyInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/devices/{id}", Unit: "device.info", Type: TypeQuery, InputMapper: deviceIDInputMapper},

		{Method: http.MethodGet, Path: "/api/v2/engines", Unit: "engine.list", Type: TypeQuery, InputMapper: queryInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/engines/{name}", Unit: "engine.get", Type: TypeQuery, InputMapper: nameInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/engines/start", Unit: "engine.start", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/engines/stop", Unit: "engine.stop", Type: TypeCommand, InputMapper: bodyInputMapper},

		{Method: http.MethodGet, Path: "/api/v2/resource/status", Unit: "resource.status", Type: TypeQuery, InputMapper: emptyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/resource/allocate", Unit: "resource.allocate", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/resource/release", Unit: "resource.release", Type: TypeCommand, InputMapper: bodyInputMapper},

		{Method: http.MethodGet, Path: "/api/v2/services", Unit: "service.list", Type: TypeQuery, InputMapper: queryInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/services", Unit: "service.create", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/services/{id}", Unit: "service.get", Type: TypeQuery, InputMapper: serviceIDInputMapper},
		{Method: http.MethodDelete, Path: "/api/v2/services/{id}", Unit: "service.delete", Type: TypeCommand, InputMapper: serviceIDInputMapper},

		{Method: http.MethodGet, Path: "/api/v2/apps", Unit: "app.list", Type: TypeQuery, InputMapper: queryInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/apps", Unit: "app.install", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/apps/{id}", Unit: "app.get", Type: TypeQuery, InputMapper: appIDInputMapper},

		// Catalog domain
		{Method: http.MethodPost, Path: "/api/v2/catalog/recipes", Unit: "catalog.create_recipe", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/catalog/recipes/validate", Unit: "catalog.validate_recipe", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/catalog/recipes/{id}/apply", Unit: "catalog.apply_recipe", Type: TypeCommand, InputMapper: recipeIDInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/catalog/recipes/match", Unit: "catalog.match", Type: TypeQuery, InputMapper: queryInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/catalog/recipes", Unit: "catalog.list", Type: TypeQuery, InputMapper: queryInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/catalog/recipes/{id}/status", Unit: "catalog.check_status", Type: TypeQuery, InputMapper: recipeIDInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/catalog/recipes/{id}", Unit: "catalog.get", Type: TypeQuery, InputMapper: recipeIDInputMapper},

		// Skill domain
		{Method: http.MethodPost, Path: "/api/v2/skills", Unit: "skill.add", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodDelete, Path: "/api/v2/skills/{id}", Unit: "skill.remove", Type: TypeCommand, InputMapper: skillIDInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/skills/{id}/enable", Unit: "skill.enable", Type: TypeCommand, InputMapper: skillIDInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/skills/{id}/disable", Unit: "skill.disable", Type: TypeCommand, InputMapper: skillIDInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/skills", Unit: "skill.list", Type: TypeQuery, InputMapper: queryInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/skills/search", Unit: "skill.search", Type: TypeQuery, InputMapper: queryInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/skills/{id}", Unit: "skill.get", Type: TypeQuery, InputMapper: skillIDInputMapper},

		// Agent domain
		{Method: http.MethodPost, Path: "/api/v2/agent/chat", Unit: "agent.chat", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/agent/reset", Unit: "agent.reset", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/agent/status", Unit: "agent.status", Type: TypeQuery, InputMapper: emptyInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/agent/history", Unit: "agent.history", Type: TypeQuery, InputMapper: queryInputMapper},

		// Alert domain
		{Method: http.MethodPost, Path: "/api/v2/alerts/rules", Unit: "alert.create_rule", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPut, Path: "/api/v2/alerts/rules/{id}", Unit: "alert.update_rule", Type: TypeCommand, InputMapper: bodyWithIDMapper},
		{Method: http.MethodDelete, Path: "/api/v2/alerts/rules/{id}", Unit: "alert.delete_rule", Type: TypeCommand, InputMapper: ruleIDInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/alerts/{id}/ack", Unit: "alert.acknowledge", Type: TypeCommand, InputMapper: alertIDInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/alerts/{id}/resolve", Unit: "alert.resolve", Type: TypeCommand, InputMapper: alertIDInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/alerts/rules", Unit: "alert.list_rules", Type: TypeQuery, InputMapper: queryInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/alerts/history", Unit: "alert.history", Type: TypeQuery, InputMapper: queryInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/alerts/active", Unit: "alert.active", Type: TypeQuery, InputMapper: emptyInputMapper},

		// Pipeline domain
		{Method: http.MethodPost, Path: "/api/v2/pipelines", Unit: "pipeline.create", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodDelete, Path: "/api/v2/pipelines/{id}", Unit: "pipeline.delete", Type: TypeCommand, InputMapper: pipelineIDInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/pipelines/{id}/run", Unit: "pipeline.run", Type: TypeCommand, InputMapper: pipelineIDInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/pipelines/{id}/cancel", Unit: "pipeline.cancel", Type: TypeCommand, InputMapper: runIDInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/pipelines", Unit: "pipeline.list", Type: TypeQuery, InputMapper: queryInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/pipelines/{id}", Unit: "pipeline.get", Type: TypeQuery, InputMapper: pipelineIDInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/pipelines/{id}/status", Unit: "pipeline.status", Type: TypeQuery, InputMapper: runIDInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/pipelines/validate", Unit: "pipeline.validate", Type: TypeQuery, InputMapper: bodyInputMapper},

		// inference — additional modalities
		{Method: http.MethodPost, Path: "/api/v2/inference/transcribe", Unit: "inference.transcribe", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/inference/synthesize", Unit: "inference.synthesize", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/inference/generate-image", Unit: "inference.generate_image", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/inference/generate-video", Unit: "inference.generate_video", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/inference/rerank", Unit: "inference.rerank", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/inference/detect", Unit: "inference.detect", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/inference/voices", Unit: "inference.voices", Type: TypeQuery, InputMapper: emptyInputMapper},

		// model — additional operations
		{Method: http.MethodPost, Path: "/api/v2/models/import", Unit: "model.import", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/models/{id}/verify", Unit: "model.verify", Type: TypeCommand, InputMapper: modelIDInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/models/search", Unit: "model.search", Type: TypeQuery, InputMapper: queryInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/models/{id}/estimate-resources", Unit: "model.estimate_resources", Type: TypeQuery, InputMapper: modelIDInputMapper},

		// engine — additional operations
		{Method: http.MethodPost, Path: "/api/v2/engines/install", Unit: "engine.install", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/engines/{name}/restart", Unit: "engine.restart", Type: TypeCommand, InputMapper: nameInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/engines/{name}/features", Unit: "engine.features", Type: TypeQuery, InputMapper: nameInputMapper},

		// resource — additional queries and update
		{Method: http.MethodGet, Path: "/api/v2/resource/budget", Unit: "resource.budget", Type: TypeQuery, InputMapper: emptyInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/resource/allocations", Unit: "resource.allocations", Type: TypeQuery, InputMapper: emptyInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/resource/can-allocate", Unit: "resource.can_allocate", Type: TypeQuery, InputMapper: queryInputMapper},
		{Method: http.MethodPut, Path: "/api/v2/resource/slots/{id}", Unit: "resource.update_slot", Type: TypeCommand, InputMapper: slotIDInputMapper},

		// device — metrics, health, power limit
		{Method: http.MethodGet, Path: "/api/v2/devices/{id}/metrics", Unit: "device.metrics", Type: TypeQuery, InputMapper: deviceIDInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/devices/{id}/health", Unit: "device.health", Type: TypeQuery, InputMapper: deviceIDInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/devices/{id}/power-limit", Unit: "device.set_power_limit", Type: TypeCommand, InputMapper: deviceIDBodyMapper},

		// service — lifecycle and status
		{Method: http.MethodPost, Path: "/api/v2/services/{id}/scale", Unit: "service.scale", Type: TypeCommand, InputMapper: serviceIDBodyMapper},
		{Method: http.MethodPost, Path: "/api/v2/services/{id}/start", Unit: "service.start", Type: TypeCommand, InputMapper: serviceIDInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/services/{id}/stop", Unit: "service.stop", Type: TypeCommand, InputMapper: serviceIDBodyMapper},
		{Method: http.MethodGet, Path: "/api/v2/services/{id}/recommend", Unit: "service.recommend", Type: TypeQuery, InputMapper: serviceIDInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/services/{id}/status", Unit: "service.status", Type: TypeQuery, InputMapper: serviceIDInputMapper},

		// app — lifecycle, logs and templates
		{Method: http.MethodDelete, Path: "/api/v2/apps/{id}", Unit: "app.uninstall", Type: TypeCommand, InputMapper: appIDInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/apps/{id}/start", Unit: "app.start", Type: TypeCommand, InputMapper: appIDInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/apps/{id}/stop", Unit: "app.stop", Type: TypeCommand, InputMapper: appIDBodyMapper},
		{Method: http.MethodGet, Path: "/api/v2/apps/{id}/logs", Unit: "app.logs", Type: TypeQuery, InputMapper: appIDInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/apps/templates", Unit: "app.templates", Type: TypeQuery, InputMapper: emptyInputMapper},

		// Remote domain
		{Method: http.MethodPost, Path: "/api/v2/remote/enable", Unit: "remote.enable", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/remote/disable", Unit: "remote.disable", Type: TypeCommand, InputMapper: emptyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/remote/exec", Unit: "remote.exec", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/remote/status", Unit: "remote.status", Type: TypeQuery, InputMapper: emptyInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/remote/audit", Unit: "remote.audit", Type: TypeQuery, InputMapper: queryInputMapper},
	}
}

func bodyInputMapper(r *http.Request, _ map[string]string) map[string]any {
	if r.Body == nil {
		return map[string]any{}
	}
	var input map[string]any
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		// Bug #43: signal decode failure via sentinel key so handleRoute can
		// return HTTP 400 instead of silently using an empty input map.
		return map[string]any{bodyDecodeErrKey: err.Error()}
	}
	if input == nil {
		return map[string]any{}
	}
	return input
}

func queryInputMapper(r *http.Request, _ map[string]string) map[string]any {
	input := map[string]any{}
	for k, v := range r.URL.Query() {
		if len(v) == 1 {
			input[k] = v[0]
		} else {
			input[k] = v
		}
	}
	return input
}

func idInputMapper(_ *http.Request, pathParams map[string]string) map[string]any {
	return map[string]any{
		"id": pathParams["id"],
	}
}

func nameInputMapper(_ *http.Request, pathParams map[string]string) map[string]any {
	return map[string]any{
		"name": pathParams["name"],
	}
}

func emptyInputMapper(_ *http.Request, _ map[string]string) map[string]any {
	return map[string]any{}
}

func bodyWithIDMapper(r *http.Request, pathParams map[string]string) map[string]any {
	input := bodyInputMapper(r, pathParams)
	if id, ok := pathParams["id"]; ok {
		input["rule_id"] = id
	}
	return input
}

func recipeIDInputMapper(_ *http.Request, pathParams map[string]string) map[string]any {
	return map[string]any{
		"recipe_id": pathParams["id"],
	}
}

func skillIDInputMapper(_ *http.Request, pathParams map[string]string) map[string]any {
	return map[string]any{
		"skill_id": pathParams["id"],
	}
}

func modelIDInputMapper(_ *http.Request, pathParams map[string]string) map[string]any {
	return map[string]any{
		"model_id": pathParams["id"],
	}
}

func pipelineIDInputMapper(_ *http.Request, pathParams map[string]string) map[string]any {
	return map[string]any{
		"pipeline_id": pathParams["id"],
	}
}

func runIDInputMapper(_ *http.Request, pathParams map[string]string) map[string]any {
	return map[string]any{
		"run_id": pathParams["id"],
	}
}

func ruleIDInputMapper(_ *http.Request, pathParams map[string]string) map[string]any {
	return map[string]any{
		"rule_id": pathParams["id"],
	}
}

func alertIDInputMapper(_ *http.Request, pathParams map[string]string) map[string]any {
	return map[string]any{
		"alert_id": pathParams["id"],
	}
}

func deviceIDInputMapper(_ *http.Request, pathParams map[string]string) map[string]any {
	return map[string]any{
		"device_id": pathParams["id"],
	}
}

func deviceIDBodyMapper(r *http.Request, pathParams map[string]string) map[string]any {
	input := bodyInputMapper(r, pathParams)
	if id, ok := pathParams["id"]; ok {
		input["device_id"] = id
	}
	return input
}

func serviceIDInputMapper(_ *http.Request, pathParams map[string]string) map[string]any {
	return map[string]any{
		"service_id": pathParams["id"],
	}
}

func serviceIDBodyMapper(r *http.Request, pathParams map[string]string) map[string]any {
	input := bodyInputMapper(r, pathParams)
	if id, ok := pathParams["id"]; ok {
		input["service_id"] = id
	}
	return input
}

func appIDInputMapper(_ *http.Request, pathParams map[string]string) map[string]any {
	return map[string]any{
		"app_id": pathParams["id"],
	}
}

func appIDBodyMapper(r *http.Request, pathParams map[string]string) map[string]any {
	input := bodyInputMapper(r, pathParams)
	if id, ok := pathParams["id"]; ok {
		input["app_id"] = id
	}
	return input
}

func slotIDInputMapper(_ *http.Request, pathParams map[string]string) map[string]any {
	return map[string]any{
		"slot_id": pathParams["id"],
	}
}

type pathParamExtractor struct{}

func newPathParamExtractor() *pathParamExtractor {
	return &pathParamExtractor{}
}

func (p *pathParamExtractor) match(pattern, path string) (map[string]string, bool) {
	params := make(map[string]string)

	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	if len(patternParts) != len(pathParts) {
		return nil, false
	}

	for i := 0; i < len(patternParts); i++ {
		pp := patternParts[i]
		pt := pathParts[i]

		if strings.HasPrefix(pp, "{") && strings.HasSuffix(pp, "}") {
			paramName := pp[1 : len(pp)-1]
			params[paramName] = pt
		} else if pp != pt {
			return nil, false
		}
	}

	return params, true
}
