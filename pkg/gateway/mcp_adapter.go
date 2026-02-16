package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	MCPVersion       = "2024-11-05"
	JSONRPC          = "2.0"
	MCPServerName    = "aima"
	MCPServerVersion = "0.1.0"
)

const (
	MCPErrorCodeParseError       = -32700
	MCPErrorCodeInvalidRequest   = -32600
	MCPErrorCodeMethodNotFound   = -32601
	MCPErrorCodeInvalidParams    = -32602
	MCPErrorCodeInternalError    = -32603
	MCPErrorCodeToolNotFound     = -32001
	MCPErrorCodeToolExecution    = -32002
	MCPErrorCodeResourceNotFound = -32003
)

type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type MCPResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *MCPError `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *MCPError) Error() string {
	return fmt.Sprintf("MCP error [%d]: %s", e.Code, e.Message)
}

type MCPInitializeParams struct {
	ProtocolVersion string                `json:"protocolVersion"`
	ClientInfo      MCPImplementationInfo `json:"clientInfo"`
	Capabilities    MCPClientCapabilities `json:"capabilities"`
}

type MCPInitializeResult struct {
	ProtocolVersion string                `json:"protocolVersion"`
	ServerInfo      MCPImplementationInfo `json:"serverInfo"`
	Capabilities    MCPServerCapabilities `json:"capabilities"`
	Instructions    string                `json:"instructions,omitempty"`
}

type MCPImplementationInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type MCPClientCapabilities struct {
	Roots    *MCPRootsCapability    `json:"roots,omitempty"`
	Sampling *MCPSamplingCapability `json:"sampling,omitempty"`
}

type MCPRootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type MCPSamplingCapability struct{}

type MCPServerCapabilities struct {
	Tools     *MCPToolsCapability     `json:"tools,omitempty"`
	Resources *MCPResourcesCapability `json:"resources,omitempty"`
	Prompts   *MCPPromptsCapability   `json:"prompts,omitempty"`
}

type MCPToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type MCPResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

type MCPPromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type MCPAdapter struct {
	gateway     *Gateway
	initialized bool
	clientInfo  *MCPImplementationInfo
}

func NewMCPAdapter(gateway *Gateway) *MCPAdapter {
	return &MCPAdapter{
		gateway: gateway,
	}
}

func (a *MCPAdapter) HandleRequest(ctx context.Context, req *MCPRequest) *MCPResponse {
	if req.JSONRPC != JSONRPC {
		return a.errorResponse(req.ID, MCPErrorCodeInvalidRequest, "invalid JSON-RPC version")
	}

	if req.Method == "" {
		return a.errorResponse(req.ID, MCPErrorCodeInvalidRequest, "method is required")
	}

	switch req.Method {
	case "initialize":
		return a.handleInitialize(ctx, req)
	case "notifications/initialized":
		a.handleInitialized(req)
		return nil
	case "ping":
		return a.handlePing(ctx, req)
	case "tools/list":
		return a.handleToolsList(ctx, req)
	case "tools/call":
		return a.handleToolsCall(ctx, req)
	case "resources/list":
		return a.handleResourcesList(ctx, req)
	case "resources/read":
		return a.handleResourcesRead(ctx, req)
	case "prompts/list":
		return a.handlePromptsList(ctx, req)
	case "shutdown":
		return a.handleShutdown(ctx, req)
	default:
		return a.errorResponse(req.ID, MCPErrorCodeMethodNotFound, "method not found: "+req.Method)
	}
}

func (a *MCPAdapter) handleInitialize(ctx context.Context, req *MCPRequest) *MCPResponse {
	var params MCPInitializeParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return a.errorResponse(req.ID, MCPErrorCodeInvalidParams, "invalid initialize params: "+err.Error())
		}
	}

	a.clientInfo = &params.ClientInfo
	a.initialized = true

	result := &MCPInitializeResult{
		ProtocolVersion: MCPVersion,
		ServerInfo: MCPImplementationInfo{
			Name:    MCPServerName,
			Version: MCPServerVersion,
		},
		Capabilities: MCPServerCapabilities{
			Tools: &MCPToolsCapability{
				ListChanged: false,
			},
			Resources: &MCPResourcesCapability{
				Subscribe:   false,
				ListChanged: false,
			},
			Prompts: &MCPPromptsCapability{
				ListChanged: false,
			},
		},
		Instructions: "AIMA MCP Server - Manage AI inference infrastructure",
	}

	return a.successResponse(req.ID, result)
}

func (a *MCPAdapter) handleInitialized(req *MCPRequest) {
	a.initialized = true
}

func (a *MCPAdapter) handlePing(ctx context.Context, req *MCPRequest) *MCPResponse {
	return a.successResponse(req.ID, map[string]any{})
}

func (a *MCPAdapter) handleShutdown(ctx context.Context, req *MCPRequest) *MCPResponse {
	a.initialized = false
	return a.successResponse(req.ID, nil)
}

func (a *MCPAdapter) successResponse(id any, result any) *MCPResponse {
	return &MCPResponse{
		JSONRPC: JSONRPC,
		ID:      id,
		Result:  result,
	}
}

func (a *MCPAdapter) errorResponse(id any, code int, message string) *MCPResponse {
	return &MCPResponse{
		JSONRPC: JSONRPC,
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
		},
	}
}

func (a *MCPAdapter) errorResponseWithData(id any, code int, message string, data any) *MCPResponse {
	return &MCPResponse{
		JSONRPC: JSONRPC,
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

func (a *MCPAdapter) Gateway() *Gateway {
	return a.gateway
}

func (a *MCPAdapter) IsInitialized() bool {
	return a.initialized
}

func unitNameToToolName(unitName string) string {
	return strings.ReplaceAll(unitName, ".", "_")
}

func toolNameToUnitName(toolName string) string {
	parts := strings.Split(toolName, "_")
	if len(parts) < 2 {
		return toolName
	}
	return parts[0] + "." + strings.Join(parts[1:], "_")
}

func isCommand(unitName string, registry *Gateway) bool {
	if registry == nil || registry.Registry() == nil {
		return true
	}
	return registry.Registry().GetCommand(unitName) != nil
}
