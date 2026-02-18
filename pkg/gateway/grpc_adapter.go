package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway/proto/pb"
)

// GRPCAdapter wraps GRPCServer and provides additional adapter functionality
type GRPCAdapter struct {
	server  *GRPCServer
	gateway *Gateway
	options GRPCAdapterOptions
}

// GRPCAdapterOptions contains configuration options for the gRPC adapter
type GRPCAdapterOptions struct {
	Address        string
	MaxMessageSize int
	EnableTLS      bool
	TLSCertFile    string
	TLSKeyFile     string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
}

// DefaultGRPCAdapterOptions returns default options
func DefaultGRPCAdapterOptions() GRPCAdapterOptions {
	return GRPCAdapterOptions{
		Address:        ":9091",
		MaxMessageSize: 100 * 1024 * 1024, // 100MB
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
	}
}

// NewGRPCAdapter creates a new gRPC adapter
func NewGRPCAdapter(gateway *Gateway, opts ...GRPCAdapterOption) *GRPCAdapter {
	options := DefaultGRPCAdapterOptions()
	for _, opt := range opts {
		opt(&options)
	}

	server := NewGRPCServer(gateway)

	return &GRPCAdapter{
		server:  server,
		gateway: gateway,
		options: options,
	}
}

// GRPCAdapterOption is a functional option for configuring the gRPC adapter
type GRPCAdapterOption func(*GRPCAdapterOptions)

// WithAddress sets the listen address
func WithAddress(addr string) GRPCAdapterOption {
	return func(o *GRPCAdapterOptions) {
		o.Address = addr
	}
}

// WithMaxMessageSize sets the maximum message size
func WithMaxMessageSize(size int) GRPCAdapterOption {
	return func(o *GRPCAdapterOptions) {
		o.MaxMessageSize = size
	}
}

// WithTLS enables TLS
func WithTLS(certFile, keyFile string) GRPCAdapterOption {
	return func(o *GRPCAdapterOptions) {
		o.EnableTLS = true
		o.TLSCertFile = certFile
		o.TLSKeyFile = keyFile
	}
}

// Execute performs a unary request through the adapter
func (a *GRPCAdapter) Execute(ctx context.Context, reqType, unit string, input map[string]any) (*pb.Response, error) {
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal input: %w", err)
	}

	req := &pb.Request{
		Type:  reqType,
		Unit:  unit,
		Input: inputBytes,
	}

	return a.server.Execute(ctx, req)
}

// ExecuteStream performs a streaming request through the adapter
func (a *GRPCAdapter) ExecuteStream(ctx context.Context, reqType, unit string, input map[string]any) (<-chan *pb.Chunk, error) {
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal input: %w", err)
	}

	req := &pb.Request{
		Type:  reqType,
		Unit:  unit,
		Input: inputBytes,
	}

	// Create output channel
	outputChan := make(chan *pb.Chunk, 10)

	// Create stream server implementation
	stream := &adapterStream{
		ctx:    ctx,
		output: outputChan,
	}

	// Run in goroutine
	go func() {
		defer close(outputChan)
		if err := a.server.ExecuteStream(req, stream); err != nil {
			outputChan <- &pb.Chunk{
				Done: true,
				Error: &pb.ErrorInfo{
					Code:    "STREAM_ERROR",
					Message: err.Error(),
				},
			}
		}
	}()

	return outputChan, nil
}

// adapterStream implements ExecuteStreamServer for adapter
type adapterStream struct {
	ctx    context.Context
	output chan *pb.Chunk
}

func (s *adapterStream) Context() context.Context {
	return s.ctx
}

func (s *adapterStream) Send(chunk *pb.Chunk) error {
	select {
	case s.output <- chunk:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

// Gateway returns the underlying gateway
func (a *GRPCAdapter) Gateway() *Gateway {
	return a.gateway
}

// Server returns the underlying gRPC server
func (a *GRPCAdapter) Server() *GRPCServer {
	return a.server
}
