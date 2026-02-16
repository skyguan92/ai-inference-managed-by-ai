package unit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

type contextKey string

const (
	RequestIDKey contextKey = "request_id"
	TraceIDKey   contextKey = "trace_id"
	UserIDKey    contextKey = "user_id"
	StartTimeKey contextKey = "start_time"
	MetadataKey  contextKey = "metadata"
)

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

func WithStartTime(ctx context.Context, t time.Time) context.Context {
	return context.WithValue(ctx, StartTimeKey, t)
}

func WithMetadata(ctx context.Context, meta map[string]any) context.Context {
	return context.WithValue(ctx, MetadataKey, meta)
}

func GetRequestID(ctx context.Context) string {
	if v := ctx.Value(RequestIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func GetTraceID(ctx context.Context) string {
	if v := ctx.Value(TraceIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func GetUserID(ctx context.Context) string {
	if v := ctx.Value(UserIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func GetStartTime(ctx context.Context) time.Time {
	if v := ctx.Value(StartTimeKey); v != nil {
		if t, ok := v.(time.Time); ok {
			return t
		}
	}
	return time.Time{}
}

func GetMetadata(ctx context.Context) map[string]any {
	if v := ctx.Value(MetadataKey); v != nil {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return nil
}

func GenerateRequestID() string {
	return fmt.Sprintf("req_%s", generateRandomHex(16))
}

func GenerateTraceID() string {
	return fmt.Sprintf("trc_%s", generateRandomHex(16))
}

func generateRandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		timestamp := time.Now().UnixNano()
		for i := range n {
			b[i] = byte(timestamp >> (i * 8))
		}
	}
	return hex.EncodeToString(b)
}
