// Package logger provides structured logging for the AIMA application.
// It wraps log/slog to provide a consistent logging interface across
// the codebase with support for JSON output, request correlation,
// and configurable log levels.
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
)

// contextKey is a private type for context keys in this package.
type contextKey int

const (
	requestIDKey contextKey = iota
	traceIDKey
	unitKey
)

var (
	defaultLogger *slog.Logger
	once          sync.Once
	mu            sync.RWMutex
)

// Config holds logger configuration.
type Config struct {
	// Level is the minimum log level (debug, info, warn, error).
	Level string
	// Format is the output format (json, text).
	Format string
	// Output is the writer to log to (defaults to os.Stderr).
	Output io.Writer
	// AddSource adds source file:line to log entries.
	AddSource bool
}

// Init initializes the default logger with the given configuration.
// It is safe to call multiple times; only the first call takes effect.
// Use Reset() followed by Init() to reconfigure.
func Init(cfg Config) {
	mu.Lock()
	defer mu.Unlock()
	once.Do(func() {
		initLogger(cfg)
	})
}

// Reset resets the default logger so Init can be called again.
// This is primarily for testing. It is safe to call concurrently.
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	once = sync.Once{}
	defaultLogger = nil
}

func initLogger(cfg Config) {
	output := cfg.Output
	if output == nil {
		output = os.Stderr
	}

	level := parseLevel(cfg.Level)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	var handler slog.Handler
	switch cfg.Format {
	case "json":
		handler = slog.NewJSONHandler(output, opts)
	default:
		handler = slog.NewTextHandler(output, opts)
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Default returns the default logger instance.
// If Init() has not been called, returns a basic text logger on stderr.
func Default() *slog.Logger {
	mu.RLock()
	l := defaultLogger
	mu.RUnlock()
	if l == nil {
		return slog.Default()
	}
	return l
}

// WithContext returns a logger enriched with context values
// (request_id, trace_id, unit) if they are present.
func WithContext(ctx context.Context) *slog.Logger {
	l := Default()

	if rid, ok := ctx.Value(requestIDKey).(string); ok && rid != "" {
		l = l.With("request_id", rid)
	}
	if tid, ok := ctx.Value(traceIDKey).(string); ok && tid != "" {
		l = l.With("trace_id", tid)
	}
	if u, ok := ctx.Value(unitKey).(string); ok && u != "" {
		l = l.With("unit", u)
	}

	return l
}

// SetRequestID adds a request ID to the context.
func SetRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// SetTraceID adds a trace ID to the context.
func SetTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey, id)
}

// SetUnit adds a unit name to the context.
func SetUnit(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, unitKey, name)
}

// GetRequestID extracts the request ID from the context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// GetTraceID extracts the trace ID from the context.
func GetTraceID(ctx context.Context) string {
	if id, ok := ctx.Value(traceIDKey).(string); ok {
		return id
	}
	return ""
}

// Convenience functions that delegate to the default logger.

func Debug(msg string, args ...any) { Default().Debug(msg, args...) }
func Info(msg string, args ...any)  { Default().Info(msg, args...) }
func Warn(msg string, args ...any)  { Default().Warn(msg, args...) }
func Error(msg string, args ...any) { Default().Error(msg, args...) }
