package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type MCPResourcesListResult struct {
	Resources []MCPResource `json:"resources"`
}

type MCPResourceReadParams struct {
	URI string `json:"uri"`
}

type MCPResourceReadResult struct {
	Contents []MCPResourceContent `json:"contents"`
}

type MCPResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     []byte `json:"blob,omitempty"`
}

func (a *MCPAdapter) handleResourcesList(ctx context.Context, req *MCPRequest) *MCPResponse {
	resources := a.ListResources()
	result := &MCPResourcesListResult{
		Resources: resources,
	}
	return a.successResponse(req.ID, result)
}

func (a *MCPAdapter) handleResourcesRead(ctx context.Context, req *MCPRequest) *MCPResponse {
	var params MCPResourceReadParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return a.errorResponse(req.ID, MCPErrorCodeInvalidParams, "invalid resource read params: "+err.Error())
		}
	}

	if params.URI == "" {
		return a.errorResponse(req.ID, MCPErrorCodeInvalidParams, "resource URI is required")
	}

	result, err := a.ReadResource(ctx, params.URI)
	if err != nil {
		return a.errorResponseWithData(req.ID, MCPErrorCodeResourceNotFound, err.Error(), nil)
	}

	return a.successResponse(req.ID, result)
}

func (a *MCPAdapter) ListResources() []MCPResource {
	var resources []MCPResource

	if a.gateway == nil || a.gateway.Registry() == nil {
		return resources
	}

	registry := a.gateway.Registry()

	for _, res := range registry.ListResources() {
		mcpResource := a.resourceToMCPResource(res)
		resources = append(resources, mcpResource)
	}

	return resources
}

func (a *MCPAdapter) resourceToMCPResource(res unit.Resource) MCPResource {
	return MCPResource{
		URI:         res.URI(),
		Name:        res.URI(),
		Description: res.Schema().Description,
		MimeType:    "application/json",
	}
}

func (a *MCPAdapter) ReadResource(ctx context.Context, uri string) (*MCPResourceReadResult, error) {
	if a.gateway == nil {
		return nil, fmt.Errorf("resource not found: %s", uri)
	}

	registry := a.gateway.Registry()
	if registry == nil {
		return nil, fmt.Errorf("resource not found: %s", uri)
	}

	res := registry.GetResource(uri)
	if res == nil {
		return nil, fmt.Errorf("resource not found: %s", uri)
	}

	data, err := res.Get(ctx)
	if err != nil {
		return nil, err
	}

	var contentText string
	if bytes, ok := data.([]byte); ok {
		contentText = string(bytes)
	} else {
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			contentText = string(jsonBytes)
		} else {
			contentText = string(jsonBytes)
		}
	}

	result := &MCPResourceReadResult{
		Contents: []MCPResourceContent{
			{
				URI:      uri,
				MimeType: "application/json",
				Text:     contentText,
			},
		},
	}

	return result, nil
}
