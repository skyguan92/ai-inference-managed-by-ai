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
