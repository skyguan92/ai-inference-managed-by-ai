package gateway

import (
	"encoding/json"
	"io"
	"net/http"
)

const (
	ContentTypeJSON = "application/json"
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
