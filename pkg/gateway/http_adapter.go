package gateway

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	ContentTypeJSON = "application/json"
	ContentTypeSSE  = "text/event-stream"
	HeaderRequestID = "X-Request-ID"
	HeaderTraceID   = "X-Trace-ID"
)

type HTTPAdapter struct {
	gateway *Gateway
}

func NewHTTPAdapter(gateway *Gateway) *HTTPAdapter {
	return &HTTPAdapter{
		gateway: gateway,
	}
}

func (a *HTTPAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, ErrCodeInvalidRequest, "method not allowed")
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "" && contentType != ContentTypeJSON {
		writeJSONError(w, http.StatusUnsupportedMediaType, ErrCodeInvalidRequest, "content-type must be application/json")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	var req Request
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "invalid JSON body: "+err.Error())
			return
		}
	}

	traceID := r.Header.Get(HeaderTraceID)
	if traceID != "" {
		req.Options.TraceID = traceID
	}

	// Check if streaming is requested
	if isStreamingRequest(&req) {
		a.handleStreamRequest(ctx, w, r, &req)
		return
	}

	resp := a.gateway.Handle(ctx, &req)

	a.writeResponse(w, resp)
}

func (a *HTTPAdapter) writeResponse(w http.ResponseWriter, resp *Response) {
	w.Header().Set("Content-Type", ContentTypeJSON)

	if resp.Meta != nil {
		if resp.Meta.RequestID != "" {
			w.Header().Set(HeaderRequestID, resp.Meta.RequestID)
		}
		if resp.Meta.TraceID != "" {
			w.Header().Set(HeaderTraceID, resp.Meta.TraceID)
		}
	}

	statusCode := http.StatusOK
	if !resp.Success {
		statusCode = errorToStatusCode(resp.Error)
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

func errorToStatusCode(err *ErrorInfo) int {
	if err == nil {
		return http.StatusInternalServerError
	}

	switch err.Code {
	case ErrCodeInvalidRequest, ErrCodeValidationFailed:
		return http.StatusBadRequest
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrCodeRateLimited:
		return http.StatusTooManyRequests
	case ErrCodeUnitNotFound, ErrCodeResourceNotFound:
		return http.StatusNotFound
	case ErrCodeTimeout:
		return http.StatusRequestTimeout
	default:
		return http.StatusInternalServerError
	}
}

func writeJSONError(w http.ResponseWriter, statusCode int, code string, message string) {
	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(statusCode)

	requestID := generateRequestIDSimple()
	w.Header().Set(HeaderRequestID, requestID)

	resp := &Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
		Meta: &ResponseMeta{
			RequestID: requestID,
		},
	}
	json.NewEncoder(w).Encode(resp)
}

func generateRequestIDSimple() string {
	return "req_" + randomHex(8)
}

func randomHex(n int) string {
	const hexChars = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = hexChars[(i*7+n*13)%16]
	}
	return string(b)
}

func (a *HTTPAdapter) Gateway() *Gateway {
	return a.gateway
}

// isStreamingRequest checks if the request requires streaming response
func isStreamingRequest(req *Request) bool {
	if req == nil {
		return false
	}
	// Check explicit stream flag in input
	if req.Input != nil {
		if stream, ok := req.Input["stream"].(bool); ok {
			return stream
		}
	}
	return false
}

// handleStreamRequest handles streaming requests using Server-Sent Events (SSE)
func (a *HTTPAdapter) handleStreamRequest(ctx context.Context, w http.ResponseWriter, r *http.Request, req *Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", ContentTypeSSE)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Flush headers
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Get streaming response channel
	stream, err := a.gateway.HandleStream(ctx, req)
	if err != nil {
		writeSSEError(w, err)
		return
	}

	// Create buffered writer for SSE
	writer := bufio.NewWriter(w)

	// Stream chunks to client
	for resp := range stream {
		if resp.Error != nil {
			writeSSEEvent(writer, "error", resp.Error)
			writer.Flush()
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			return
		}

		if resp.Done {
			// Send [DONE] marker in OpenAI-compatible format
			writeSSEData(writer, "[DONE]")
			writer.Flush()
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			return
		}

		// Format data according to SSE spec with OpenAI-compatible JSON
		data := formatSSEData(resp)
		writeSSEData(writer, data)
		writer.Flush()

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

// writeSSEData writes a data event in SSE format
func writeSSEData(w *bufio.Writer, data string) {
	fmt.Fprintf(w, "data: %s\n\n", data)
}

// writeSSEEvent writes a named event in SSE format
func writeSSEEvent(w *bufio.Writer, event string, data any) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(jsonData))
}

// writeSSEError writes an error event and closes the stream
func writeSSEError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", ContentTypeSSE)
	w.WriteHeader(http.StatusOK)

	writer := bufio.NewWriter(w)
	var errInfo *ErrorInfo
	if e, ok := err.(*ErrorInfo); ok {
		errInfo = e
	} else {
		errInfo = ToErrorInfo(err)
	}
	writeSSEEvent(writer, "error", errInfo)
	writer.Flush()
}

// formatSSEData formats the stream response as SSE data in OpenAI-compatible format
func formatSSEData(resp StreamResponse) string {
	// OpenAI-compatible SSE format
	chunk := map[string]any{
		"object": "chat.completion.chunk",
		"choices": []map[string]any{
			{
				"index": 0,
				"delta": map[string]any{
					"content": resp.Data,
				},
			},
		},
	}

	if resp.Metadata != nil {
		if metadata, ok := resp.Metadata.(map[string]any); ok {
			if finishReason, ok := metadata["finish_reason"].(string); ok && finishReason != "" {
				chunk["choices"].([]map[string]any)[0]["finish_reason"] = finishReason
			}
			if model, ok := metadata["model"].(string); ok {
				chunk["model"] = model
			}
			if id, ok := metadata["id"].(string); ok {
				chunk["id"] = id
			}
		}
	}

	jsonBytes, _ := json.Marshal(chunk)
	return string(jsonBytes)
}
