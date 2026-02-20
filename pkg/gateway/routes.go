package gateway

import (
	"encoding/json"
	"net/http"
	"strings"
)

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

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	for _, route := range r.routes {
		if route.Method != req.Method {
			continue
		}

		pathParams, ok := r.pathParamExtractor.match(route.Path, req.URL.Path)
		if !ok {
			continue
		}

		r.handleRoute(w, req, route, pathParams)
		return
	}

	writeJSONError(w, http.StatusNotFound, ErrCodeUnitNotFound, "route not found: "+req.Method+" "+req.URL.Path)
}

func (r *Router) handleRoute(w http.ResponseWriter, httpReq *http.Request, route Route, pathParams map[string]string) {
	ctx := httpReq.Context()

	traceID := httpReq.Header.Get(HeaderTraceID)

	input := map[string]any{}
	if route.InputMapper != nil {
		input = route.InputMapper(httpReq, pathParams)
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

	NewHTTPAdapter(r.gateway).writeResponse(w, resp)
}

func defaultRoutes() []Route {
	return []Route{
		{Method: http.MethodPost, Path: "/api/v2/models/pull", Unit: "model.pull", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/models/create", Unit: "model.create", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodDelete, Path: "/api/v2/models/{id}", Unit: "model.delete", Type: TypeCommand, InputMapper: idInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/models", Unit: "model.list", Type: TypeQuery, InputMapper: queryInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/models/{id}", Unit: "model.get", Type: TypeQuery, InputMapper: idInputMapper},

		{Method: http.MethodPost, Path: "/api/v2/inference/chat", Unit: "inference.chat", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/inference/complete", Unit: "inference.complete", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/inference/embed", Unit: "inference.embed", Type: TypeCommand, InputMapper: bodyInputMapper},

		{Method: http.MethodGet, Path: "/api/v2/devices", Unit: "device.detect", Type: TypeCommand, InputMapper: emptyInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/devices/{id}", Unit: "device.info", Type: TypeQuery, InputMapper: idInputMapper},

		{Method: http.MethodGet, Path: "/api/v2/engines", Unit: "engine.list", Type: TypeQuery, InputMapper: queryInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/engines/{name}", Unit: "engine.get", Type: TypeQuery, InputMapper: nameInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/engines/start", Unit: "engine.start", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/engines/stop", Unit: "engine.stop", Type: TypeCommand, InputMapper: bodyInputMapper},

		{Method: http.MethodGet, Path: "/api/v2/resource/status", Unit: "resource.status", Type: TypeQuery, InputMapper: emptyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/resource/allocate", Unit: "resource.allocate", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/resource/release", Unit: "resource.release", Type: TypeCommand, InputMapper: bodyInputMapper},

		{Method: http.MethodGet, Path: "/api/v2/services", Unit: "service.list", Type: TypeQuery, InputMapper: queryInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/services", Unit: "service.create", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/services/{id}", Unit: "service.get", Type: TypeQuery, InputMapper: idInputMapper},
		{Method: http.MethodDelete, Path: "/api/v2/services/{id}", Unit: "service.delete", Type: TypeCommand, InputMapper: idInputMapper},

		{Method: http.MethodGet, Path: "/api/v2/apps", Unit: "app.list", Type: TypeQuery, InputMapper: queryInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/apps", Unit: "app.install", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/apps/{id}", Unit: "app.get", Type: TypeQuery, InputMapper: idInputMapper},

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
		{Method: http.MethodDelete, Path: "/api/v2/alerts/rules/{id}", Unit: "alert.delete_rule", Type: TypeCommand, InputMapper: idInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/alerts/{id}/ack", Unit: "alert.acknowledge", Type: TypeCommand, InputMapper: idInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/alerts/{id}/resolve", Unit: "alert.resolve", Type: TypeCommand, InputMapper: idInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/alerts/rules", Unit: "alert.list_rules", Type: TypeQuery, InputMapper: queryInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/alerts/history", Unit: "alert.history", Type: TypeQuery, InputMapper: queryInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/alerts/active", Unit: "alert.active", Type: TypeQuery, InputMapper: emptyInputMapper},

		// Pipeline domain
		{Method: http.MethodPost, Path: "/api/v2/pipelines", Unit: "pipeline.create", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodDelete, Path: "/api/v2/pipelines/{id}", Unit: "pipeline.delete", Type: TypeCommand, InputMapper: idInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/pipelines/{id}/run", Unit: "pipeline.run", Type: TypeCommand, InputMapper: idInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/pipelines/{id}/cancel", Unit: "pipeline.cancel", Type: TypeCommand, InputMapper: idInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/pipelines", Unit: "pipeline.list", Type: TypeQuery, InputMapper: queryInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/pipelines/{id}", Unit: "pipeline.get", Type: TypeQuery, InputMapper: idInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/pipelines/{id}/status", Unit: "pipeline.status", Type: TypeQuery, InputMapper: idInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/pipelines/validate", Unit: "pipeline.validate", Type: TypeCommand, InputMapper: bodyInputMapper},

		// Remote domain
		{Method: http.MethodPost, Path: "/api/v2/remote/enable", Unit: "remote.enable", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/remote/disable", Unit: "remote.disable", Type: TypeCommand, InputMapper: emptyInputMapper},
		{Method: http.MethodPost, Path: "/api/v2/remote/exec", Unit: "remote.exec", Type: TypeCommand, InputMapper: bodyInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/remote/status", Unit: "remote.status", Type: TypeQuery, InputMapper: emptyInputMapper},
		{Method: http.MethodGet, Path: "/api/v2/remote/audit", Unit: "remote.audit", Type: TypeQuery, InputMapper: queryInputMapper},
	}
}

func bodyInputMapper(r *http.Request, _ map[string]string) map[string]any {
	var input map[string]any
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			return map[string]any{}
		}
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
