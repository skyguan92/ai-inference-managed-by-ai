package gateway

import (
	"context"
	"sync"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

// OptimizedGateway is a performance-optimized version of Gateway.
// It uses object pooling to reduce GC pressure in high-throughput scenarios.
type OptimizedGateway struct {
	*Gateway
	responsePool *sync.Pool
}

// NewOptimizedGateway creates a new optimized gateway.
func NewOptimizedGateway(registry *unit.Registry, opts ...GatewayOption) *OptimizedGateway {
	g := NewGateway(registry, opts...)
	
	return &OptimizedGateway{
		Gateway: g,
		responsePool: &sync.Pool{
			New: func() any {
				return &Response{
					Meta: &ResponseMeta{},
				}
			},
		},
	}
}

// Handle processes a request with optimized memory usage.
func (g *OptimizedGateway) Handle(ctx context.Context, req *Request) *Response {
	start := time.Now()
	requestID := unit.GenerateRequestID()

	// Get response from pool
	resp := g.responsePool.Get().(*Response)
	resp.Success = false
	resp.Data = nil
	resp.Error = nil
	resp.Meta.RequestID = requestID
	resp.Meta.Duration = 0
	resp.Meta.TraceID = ""

	if err := g.validateRequest(req); err != nil {
		resp.Error = err
		resp.Meta.Duration = time.Since(start).Milliseconds()
		return resp
	}

	traceID := req.Options.TraceID
	if traceID == "" {
		traceID = unit.GenerateTraceID()
	}
	resp.Meta.TraceID = traceID

	ctx = unit.WithRequestID(ctx, requestID)
	ctx = unit.WithTraceID(ctx, traceID)
	ctx = unit.WithStartTime(ctx, start)

	timeout := req.Options.Timeout
	if timeout <= 0 {
		timeout = g.requestTimeout
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := g.execute(ctx, req)
	if err != nil {
		resp.Error = ToErrorInfo(err)
		resp.Meta.Duration = time.Since(start).Milliseconds()
		return resp
	}

	resp.Success = true
	resp.Data = result
	resp.Meta.Duration = time.Since(start).Milliseconds()
	return resp
}

// ReleaseResponse returns a response to the pool for reuse.
// Callers should call this when done with the response to reduce GC pressure.
func (g *OptimizedGateway) ReleaseResponse(resp *Response) {
	if resp == nil {
		return
	}
	// Clear fields to avoid memory leaks
	resp.Data = nil
	resp.Error = nil
	resp.Success = false
	if resp.Meta != nil {
		resp.Meta.RequestID = ""
		resp.Meta.TraceID = ""
		resp.Meta.Duration = 0
		resp.Meta.Pagination = nil
	}
	g.responsePool.Put(resp)
}
