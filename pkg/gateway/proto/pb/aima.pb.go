// Code generated manually for AIMA gRPC adapter.
// This file contains the protobuf message definitions and gRPC service interface.

package pb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// RequestOptions contains options for the request
type RequestOptions struct {
	TimeoutMs int32  `json:"timeout_ms,omitempty"`
	Async     bool   `json:"async,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
}

// Request represents a gRPC request to AIMA
type Request struct {
	Type    string          `json:"type,omitempty"`
	Unit    string          `json:"unit,omitempty"`
	Input   []byte          `json:"input,omitempty"`
	Options *RequestOptions `json:"options,omitempty"`
}

// ErrorInfo represents error information in the response
type ErrorInfo struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Details []byte `json:"details,omitempty"`
}

// ResponseMeta contains metadata about the response
type ResponseMeta struct {
	RequestID  string `json:"request_id,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
	TraceID    string `json:"trace_id,omitempty"`
	Page       int32  `json:"page,omitempty"`
	PerPage    int32  `json:"per_page,omitempty"`
	Total      int32  `json:"total,omitempty"`
}

// Response represents a gRPC response from AIMA
type Response struct {
	Success bool          `json:"success,omitempty"`
	Data    []byte        `json:"data,omitempty"`
	Error   *ErrorInfo    `json:"error,omitempty"`
	Meta    *ResponseMeta `json:"meta,omitempty"`
}

// Chunk represents a streaming response chunk
type Chunk struct {
	Data     []byte     `json:"data,omitempty"`
	Metadata []byte     `json:"metadata,omitempty"`
	Done     bool       `json:"done,omitempty"`
	Error    *ErrorInfo `json:"error,omitempty"`
}

// ResourceRequest represents a request to watch a resource
type ResourceRequest struct {
	URI string `json:"uri,omitempty"`
}

// ResourceUpdate represents a resource update event
type ResourceUpdate struct {
	URI       string `json:"uri,omitempty"`
	Operation string `json:"operation,omitempty"`
	Data      []byte `json:"data,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

// AIMAServiceServer is the server API for AIMAService service
type AIMAServiceServer interface {
	Execute(ctx context.Context, req *Request) (*Response, error)
	ExecuteStream(req *Request, stream AIMAService_ExecuteStreamServer) error
	WatchResource(req *ResourceRequest, stream AIMAService_WatchResourceServer) error
}

// UnimplementedAIMAServiceServer must be embedded to have forward compatible implementations
type UnimplementedAIMAServiceServer struct{}

func (UnimplementedAIMAServiceServer) Execute(ctx context.Context, req *Request) (*Response, error) {
	return nil, fmt.Errorf("method Execute not implemented")
}

func (UnimplementedAIMAServiceServer) ExecuteStream(req *Request, stream AIMAService_ExecuteStreamServer) error {
	return fmt.Errorf("method ExecuteStream not implemented")
}

func (UnimplementedAIMAServiceServer) WatchResource(req *ResourceRequest, stream AIMAService_WatchResourceServer) error {
	return fmt.Errorf("method WatchResource not implemented")
}

// AIMAService_ExecuteStreamServer is the server API for streaming response
type AIMAService_ExecuteStreamServer interface {
	Send(*Chunk) error
	Context() context.Context
}

// AIMAService_WatchResourceServer is the server API for resource watching
type AIMAService_WatchResourceServer interface {
	Send(*ResourceUpdate) error
	Context() context.Context
}

// Server is a minimal gRPC server implementation for testing
type Server struct {
	impl AIMAServiceServer
}

// NewServer creates a new gRPC server
func NewServer(impl AIMAServiceServer) *Server {
	return &Server{impl: impl}
}

// RegisterAIMAServiceServer registers the AIMAServiceServer with a gRPC server
// This is a stub implementation for when real gRPC is not available
func RegisterAIMAServiceServer(s *Server, srv AIMAServiceServer) {
	s.impl = srv
}

// grpc package types for compatibility
type ServerOption interface{}

// ServerRegistrar is used to register services
var ServerRegistrar = &serverRegistrar{}

type serverRegistrar struct{}

func (r *serverRegistrar) RegisterService(desc interface{}, impl interface{}) {}

// FromGatewayRequest converts a gateway request to protobuf request
func FromGatewayRequest(reqType, unit string, input map[string]any, opts *RequestOptions) (*Request, error) {
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal input: %w", err)
	}
	return &Request{
		Type:    reqType,
		Unit:    unit,
		Input:   inputBytes,
		Options: opts,
	}, nil
}

// ToGatewayResponse converts a protobuf response to gateway response
func ToGatewayResponse(resp *Response) (map[string]any, *ErrorInfo, *ResponseMeta) {
	var data map[string]any
	if resp.Data != nil {
		_ = json.Unmarshal(resp.Data, &data)
	}
	return data, resp.Error, resp.Meta
}

// NewResourceUpdate creates a new resource update
func NewResourceUpdate(uri, operation string, data any) (*ResourceUpdate, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal data: %w", err)
	}
	return &ResourceUpdate{
		URI:       uri,
		Operation: operation,
		Data:      dataBytes,
		Timestamp: time.Now().Unix(),
	}, nil
}
