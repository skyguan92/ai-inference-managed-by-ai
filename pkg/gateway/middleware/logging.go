package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	_, _ = rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     0,
			}

			var requestBody []byte
			if r.Body != nil {
				requestBody, _ = io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewBuffer(requestBody))
			}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			requestID := unit.GetRequestID(r.Context())
			traceID := unit.GetTraceID(r.Context())

			logAttrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.statusCode),
				slog.Duration("duration", duration),
				slog.String("remote_addr", r.RemoteAddr),
			}

			if requestID != "" {
				logAttrs = append(logAttrs, slog.String("request_id", requestID))
			}
			if traceID != "" {
				logAttrs = append(logAttrs, slog.String("trace_id", traceID))
			}

			logLevel := slog.LevelInfo
			if rw.statusCode >= 400 {
				logLevel = slog.LevelWarn
			}
			if rw.statusCode >= 500 {
				logLevel = slog.LevelError
			}

			if logger != nil {
				logger.LogAttrs(r.Context(), logLevel, "HTTP request", logAttrs...)
			}
		})
	}
}
