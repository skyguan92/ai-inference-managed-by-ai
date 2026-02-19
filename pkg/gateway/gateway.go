package gateway

import (
	"context"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/workflow"
)

const (
	TypeCommand  = "command"
	TypeQuery    = "query"
	TypeResource = "resource"
	TypeWorkflow = "workflow"

	DefaultTimeout = 30 * time.Second
)

type Request struct {
	Type    string         `json:"type"`
	Unit    string         `json:"unit"`
	Input   map[string]any `json:"input,omitempty"`
	Options RequestOptions `json:"options,omitempty"`
}

type RequestOptions struct {
	Timeout time.Duration `json:"timeout,omitempty"`
	Async   bool          `json:"async,omitempty"`
	TraceID string        `json:"trace_id,omitempty"`
}

type Response struct {
	Success bool          `json:"success"`
	Data    any           `json:"data,omitempty"`
	Error   *ErrorInfo    `json:"error,omitempty"`
	Meta    *ResponseMeta `json:"meta,omitempty"`
}

type ResponseMeta struct {
	RequestID  string      `json:"request_id"`
	Duration   int64       `json:"duration_ms"`
	TraceID    string      `json:"trace_id,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

type Pagination struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

type Gateway struct {
	registry       *unit.Registry
	workflowEngine *workflow.WorkflowEngine
	requestTimeout time.Duration
}

type GatewayOption func(*Gateway)

func WithTimeout(timeout time.Duration) GatewayOption {
	return func(g *Gateway) {
		g.requestTimeout = timeout
	}
}

func WithWorkflowEngine(engine *workflow.WorkflowEngine) GatewayOption {
	return func(g *Gateway) {
		g.workflowEngine = engine
	}
}

func NewGateway(registry *unit.Registry, opts ...GatewayOption) *Gateway {
	if registry == nil {
		registry = unit.NewRegistry()
	}

	g := &Gateway{
		registry:       registry,
		requestTimeout: DefaultTimeout,
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

func (g *Gateway) Handle(ctx context.Context, req *Request) *Response {
	start := time.Now()
	requestID := unit.GenerateRequestID()

	resp := &Response{
		Meta: &ResponseMeta{
			RequestID: requestID,
		},
	}
	defer func() {
		resp.Meta.Duration = time.Since(start).Milliseconds()
	}()

	if err := g.validateRequest(req); err != nil {
		resp.Success = false
		resp.Error = err
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
		resp.Success = false
		resp.Error = ToErrorInfo(err)
		return resp
	}

	resp.Success = true
	resp.Data = result
	return resp
}

func (g *Gateway) validateRequest(req *Request) *ErrorInfo {
	if req == nil {
		return NewErrorInfo(ErrCodeInvalidRequest, "request is nil")
	}

	switch req.Type {
	case TypeCommand, TypeQuery, TypeResource, TypeWorkflow:
	default:
		return NewErrorInfo(ErrCodeInvalidRequest, "invalid request type: "+req.Type)
	}

	if req.Unit == "" {
		return NewErrorInfo(ErrCodeInvalidRequest, "unit is required")
	}

	return nil
}

func (g *Gateway) execute(ctx context.Context, req *Request) (any, error) {
	switch req.Type {
	case TypeCommand:
		return g.executeCommand(ctx, req)
	case TypeQuery:
		return g.executeQuery(ctx, req)
	case TypeResource:
		return g.executeResource(ctx, req)
	case TypeWorkflow:
		return g.executeWorkflow(ctx, req)
	default:
		return nil, NewErrorInfo(ErrCodeInvalidRequest, "unknown request type: "+req.Type)
	}
}

func (g *Gateway) executeCommand(ctx context.Context, req *Request) (any, error) {
	cmd := g.registry.GetCommand(req.Unit)
	if cmd == nil {
		return nil, NewErrorInfo(ErrCodeUnitNotFound, "command not found: "+req.Unit)
	}

	result, err := cmd.Execute(ctx, req.Input)
	if err != nil {
		return nil, NewErrorInfoWithDetails(ErrCodeExecutionFailed, "command execution failed", err.Error())
	}

	return result, nil
}

func (g *Gateway) executeQuery(ctx context.Context, req *Request) (any, error) {
	q := g.registry.GetQuery(req.Unit)
	if q == nil {
		return nil, NewErrorInfo(ErrCodeUnitNotFound, "query not found: "+req.Unit)
	}

	result, err := q.Execute(ctx, req.Input)
	if err != nil {
		return nil, NewErrorInfoWithDetails(ErrCodeExecutionFailed, "query execution failed", err.Error())
	}

	return result, nil
}

func (g *Gateway) executeResource(ctx context.Context, req *Request) (any, error) {
	// First try direct lookup
	res := g.registry.GetResource(req.Unit)
	if res == nil {
		// Try with factory (for dynamic URIs like asms://model/{id})
		res = g.registry.GetResourceWithFactory(req.Unit)
	}
	if res == nil {
		return nil, NewErrorInfo(ErrCodeResourceNotFound, "resource not found: "+req.Unit)
	}

	result, err := res.Get(ctx)
	if err != nil {
		return nil, NewErrorInfoWithDetails(ErrCodeExecutionFailed, "resource get failed", err.Error())
	}

	return result, nil
}

func (g *Gateway) executeWorkflow(ctx context.Context, req *Request) (any, error) {
	if g.workflowEngine == nil {
		return nil, NewErrorInfo(ErrCodeInternalError, "workflow engine not configured")
	}

	def, err := g.workflowEngine.GetWorkflow(ctx, req.Unit)
	if err != nil {
		return nil, NewErrorInfoWithDetails(ErrCodeUnitNotFound, "workflow not found: "+req.Unit, err.Error())
	}

	result, err := g.workflowEngine.Execute(ctx, def, req.Input)
	if err != nil {
		return nil, NewErrorInfoWithDetails(ErrCodeExecutionFailed, "workflow execution failed", err.Error())
	}

	return result, nil
}

func (g *Gateway) Registry() *unit.Registry {
	return g.registry
}

func (g *Gateway) Timeout() time.Duration {
	return g.requestTimeout
}

// StreamResponse represents a single chunk in a streaming response
type StreamResponse struct {
	Data     any        `json:"data,omitempty"`
	Metadata any        `json:"metadata,omitempty"`
	Done     bool       `json:"done,omitempty"`
	Error    *ErrorInfo `json:"error,omitempty"`
}

// HandleStream executes a streaming command and returns a channel of chunks
// This method is used for Server-Sent Events (SSE) streaming
func (g *Gateway) HandleStream(ctx context.Context, req *Request) (<-chan StreamResponse, error) {
	if err := g.validateRequest(req); err != nil {
		return nil, err
	}

	if req.Type != TypeCommand {
		return nil, NewErrorInfo(ErrCodeInvalidRequest, "streaming only supports commands")
	}

	// Check if command supports streaming
	cmd := g.registry.GetCommand(req.Unit)
	if cmd == nil {
		return nil, NewErrorInfo(ErrCodeUnitNotFound, "command not found: "+req.Unit)
	}

	streamingCmd, ok := cmd.(unit.StreamingCommand)
	if !ok || !streamingCmd.SupportsStreaming() {
		return nil, NewErrorInfo(ErrCodeInvalidRequest, "command does not support streaming: "+req.Unit)
	}

	requestID := unit.GenerateRequestID()
	ctx = unit.WithRequestID(ctx, requestID)

	timeout := req.Options.Timeout
	if timeout <= 0 {
		timeout = g.requestTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)

	// Create output channel
	stream := make(chan StreamResponse, 10)

	// Run command in goroutine
	go func() {
		defer close(stream)
		defer cancel()

		// Internal channel to receive chunks from command
		unitStream := make(chan unit.StreamChunk, 10)

		// Execute streaming command in separate goroutine
		errChan := make(chan error, 1)
		go func() {
			errChan <- streamingCmd.ExecuteStream(ctx, req.Input, unitStream)
			close(unitStream)
		}()

		// Forward chunks from unit stream to gateway stream
		for chunk := range unitStream {
			resp := StreamResponse{
				Data:     chunk.Data,
				Metadata: chunk.Metadata,
			}
			if chunk.Type == "done" {
				resp.Done = true
			}
			select {
			case stream <- resp:
			case <-ctx.Done():
				return
			}
		}

		// Check for execution error
		if err := <-errChan; err != nil {
			select {
			case stream <- StreamResponse{
				Error: ToErrorInfo(err),
				Done:  true,
			}:
			case <-ctx.Done():
			}
			return
		}

		// Send done signal
		select {
		case stream <- StreamResponse{Done: true}:
		case <-ctx.Done():
		}
	}()

	return stream, nil
}
