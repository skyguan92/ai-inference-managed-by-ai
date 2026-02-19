package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway/proto/pb"
)

// ExecuteStreamServer defines the interface for streaming responses
type ExecuteStreamServer interface {
	Context() context.Context
	Send(*pb.Chunk) error
}

// WatchResourceServer defines the interface for resource watching
type WatchResourceServer interface {
	Context() context.Context
	Send(*pb.ResourceUpdate) error
}

// GRPCServer implements the AIMAService gRPC interface
type GRPCServer struct {
	gateway *Gateway
	pb.UnimplementedAIMAServiceServer
}

// NewGRPCServer creates a new gRPC server instance
func NewGRPCServer(gateway *Gateway) *GRPCServer {
	return &GRPCServer{
		gateway: gateway,
	}
}

// Gateway returns the underlying gateway instance
func (s *GRPCServer) Gateway() *Gateway {
	return s.gateway
}

// Execute handles unary gRPC requests
func (s *GRPCServer) Execute(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	// Convert protobuf request to gateway request
	gatewayReq, err := s.convertRequest(req)
	if err != nil {
		return s.buildErrorResponse(err), nil
	}

	// Call gateway handler
	resp := s.gateway.Handle(ctx, gatewayReq)

	// Convert gateway response to protobuf response
	return s.convertResponse(resp), nil
}

// ExecuteStream handles streaming gRPC requests
func (s *GRPCServer) ExecuteStream(req *pb.Request, stream ExecuteStreamServer) error {
	ctx := stream.Context()

	// Convert protobuf request to gateway request
	gatewayReq, err := s.convertRequest(req)
	if err != nil {
		return s.sendErrorChunk(stream, err)
	}

	// Check if streaming is supported
	if gatewayReq.Type != TypeCommand {
		return s.sendErrorChunk(stream, fmt.Errorf("streaming only supports commands"))
	}

	// Get streaming response from gateway
	streamChan, err := s.gateway.HandleStream(ctx, gatewayReq)
	if err != nil {
		return s.sendErrorChunk(stream, err)
	}

	// Forward chunks to gRPC stream
	for chunk := range streamChan {
		pbChunk := &pb.Chunk{
			Done:  chunk.Done,
			Error: s.convertErrorInfo(chunk.Error),
		}

		if chunk.Data != nil {
			dataBytes, _ := json.Marshal(chunk.Data)
			pbChunk.Data = dataBytes
		}

		if chunk.Metadata != nil {
			metaBytes, _ := json.Marshal(chunk.Metadata)
			pbChunk.Metadata = metaBytes
		}

		if err := stream.Send(pbChunk); err != nil {
			return err
		}

		if chunk.Done {
			break
		}
	}

	return nil
}

// WatchResource watches a resource for changes
func (s *GRPCServer) WatchResource(req *pb.ResourceRequest, stream WatchResourceServer) error {
	ctx := stream.Context()

	// Get the resource from registry
	res := s.gateway.registry.GetResource(req.URI)
	if res == nil {
		res = s.gateway.registry.GetResourceWithFactory(req.URI)
	}
	if res == nil {
		return fmt.Errorf("resource not found: %s", req.URI)
	}

	// Start watching - all Resource implementations support Watch
	updateChan, err := res.Watch(ctx)
	if err != nil {
		return fmt.Errorf("failed to watch resource: %w", err)
	}

	// Forward updates to gRPC stream
	for update := range updateChan {
		pbUpdate := &pb.ResourceUpdate{
			URI:       update.URI,
			Operation: update.Operation,
			Timestamp: update.Timestamp.Unix(),
		}

		if update.Data != nil {
			dataBytes, _ := json.Marshal(update.Data)
			pbUpdate.Data = dataBytes
		}

		if err := stream.Send(pbUpdate); err != nil {
			return err
		}
	}

	return nil
}

// convertRequest converts a protobuf request to a gateway request
func (s *GRPCServer) convertRequest(req *pb.Request) (*Request, error) {
	var input map[string]any
	if len(req.Input) > 0 {
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return nil, fmt.Errorf("invalid input JSON: %w", err)
		}
	}

	var opts RequestOptions
	if req.Options != nil {
		opts = RequestOptions{
			TraceID: req.Options.TraceID,
			Async:   req.Options.Async,
		}
		if req.Options.TimeoutMs > 0 {
			opts.Timeout = time.Duration(req.Options.TimeoutMs) * time.Millisecond
		}
	}

	return &Request{
		Type:    req.Type,
		Unit:    req.Unit,
		Input:   input,
		Options: opts,
	}, nil
}

// convertResponse converts a gateway response to a protobuf response
func (s *GRPCServer) convertResponse(resp *Response) *pb.Response {
	var data []byte
	if resp.Data != nil {
		data, _ = json.Marshal(resp.Data)
	}

	var meta *pb.ResponseMeta
	if resp.Meta != nil {
		meta = &pb.ResponseMeta{
			RequestID:  resp.Meta.RequestID,
			DurationMs: resp.Meta.Duration,
			TraceID:    resp.Meta.TraceID,
		}
		if resp.Meta.Pagination != nil {
			meta.Page = int32(resp.Meta.Pagination.Page)
			meta.PerPage = int32(resp.Meta.Pagination.PerPage)
			meta.Total = int32(resp.Meta.Pagination.Total)
		}
	}

	return &pb.Response{
		Success: resp.Success,
		Data:    data,
		Error:   s.convertErrorInfo(resp.Error),
		Meta:    meta,
	}
}

// convertErrorInfo converts a gateway error to protobuf error
func (s *GRPCServer) convertErrorInfo(err *ErrorInfo) *pb.ErrorInfo {
	if err == nil {
		return nil
	}

	var details []byte
	if err.Details != nil {
		details, _ = json.Marshal(err.Details)
	}

	return &pb.ErrorInfo{
		Code:    err.Code,
		Message: err.Message,
		Details: details,
	}
}

// buildErrorResponse builds a protobuf error response
func (s *GRPCServer) buildErrorResponse(err error) *pb.Response {
	errInfo := ToErrorInfo(err)
	return &pb.Response{
		Success: false,
		Error: &pb.ErrorInfo{
			Code:    errInfo.Code,
			Message: errInfo.Message,
		},
	}
}

// sendErrorChunk sends an error chunk in the stream
func (s *GRPCServer) sendErrorChunk(stream ExecuteStreamServer, err error) error {
	errInfo := ToErrorInfo(err)
	return stream.Send(&pb.Chunk{
		Done:  true,
		Error: s.convertErrorInfo(errInfo),
	})
}
