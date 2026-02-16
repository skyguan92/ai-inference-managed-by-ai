package unit

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()
	requestID := "req_test123"

	newCtx := WithRequestID(ctx, requestID)

	if GetRequestID(newCtx) != requestID {
		t.Errorf("GetRequestID() = %q, want %q", GetRequestID(newCtx), requestID)
	}

	if GetRequestID(ctx) != "" {
		t.Errorf("GetRequestID() on original context should return empty string")
	}
}

func TestWithTraceID(t *testing.T) {
	ctx := context.Background()
	traceID := "trc_test123"

	newCtx := WithTraceID(ctx, traceID)

	if GetTraceID(newCtx) != traceID {
		t.Errorf("GetTraceID() = %q, want %q", GetTraceID(newCtx), traceID)
	}

	if GetTraceID(ctx) != "" {
		t.Errorf("GetTraceID() on original context should return empty string")
	}
}

func TestWithUserID(t *testing.T) {
	ctx := context.Background()
	userID := "user_abc123"

	newCtx := WithUserID(ctx, userID)

	if GetUserID(newCtx) != userID {
		t.Errorf("GetUserID() = %q, want %q", GetUserID(newCtx), userID)
	}

	if GetUserID(ctx) != "" {
		t.Errorf("GetUserID() on original context should return empty string")
	}
}

func TestWithStartTime(t *testing.T) {
	ctx := context.Background()
	startTime := time.Date(2026, 2, 16, 10, 30, 0, 0, time.UTC)

	newCtx := WithStartTime(ctx, startTime)

	if !GetStartTime(newCtx).Equal(startTime) {
		t.Errorf("GetStartTime() = %v, want %v", GetStartTime(newCtx), startTime)
	}

	if !GetStartTime(ctx).IsZero() {
		t.Errorf("GetStartTime() on original context should return zero time")
	}
}

func TestWithMetadata(t *testing.T) {
	ctx := context.Background()
	meta := map[string]any{
		"key1": "value1",
		"key2": 123,
	}

	newCtx := WithMetadata(ctx, meta)

	result := GetMetadata(newCtx)
	if result == nil {
		t.Fatal("GetMetadata() returned nil")
	}

	if result["key1"] != "value1" {
		t.Errorf("result[\"key1\"] = %v, want %v", result["key1"], "value1")
	}

	if result["key2"] != 123 {
		t.Errorf("result[\"key2\"] = %v, want %v", result["key2"], 123)
	}

	if GetMetadata(ctx) != nil {
		t.Errorf("GetMetadata() on original context should return nil")
	}
}

func TestGenerateRequestID(t *testing.T) {
	id1 := GenerateRequestID()
	id2 := GenerateRequestID()

	if !strings.HasPrefix(id1, "req_") {
		t.Errorf("GenerateRequestID() = %q, should have prefix 'req_'", id1)
	}

	if id1 == id2 {
		t.Errorf("GenerateRequestID() should generate unique IDs, got same: %q", id1)
	}

	if len(id1) < 10 {
		t.Errorf("GenerateRequestID() = %q, seems too short", id1)
	}
}

func TestGenerateTraceID(t *testing.T) {
	id1 := GenerateTraceID()
	id2 := GenerateTraceID()

	if !strings.HasPrefix(id1, "trc_") {
		t.Errorf("GenerateTraceID() = %q, should have prefix 'trc_'", id1)
	}

	if id1 == id2 {
		t.Errorf("GenerateTraceID() should generate unique IDs, got same: %q", id1)
	}

	if len(id1) < 10 {
		t.Errorf("GenerateTraceID() = %q, seems too short", id1)
	}
}

func TestMultipleContextValues(t *testing.T) {
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req_123")
	ctx = WithTraceID(ctx, "trc_456")
	ctx = WithUserID(ctx, "user_789")
	ctx = WithStartTime(ctx, time.Now())

	if GetRequestID(ctx) != "req_123" {
		t.Errorf("GetRequestID() = %q, want %q", GetRequestID(ctx), "req_123")
	}

	if GetTraceID(ctx) != "trc_456" {
		t.Errorf("GetTraceID() = %q, want %q", GetTraceID(ctx), "trc_456")
	}

	if GetUserID(ctx) != "user_789" {
		t.Errorf("GetUserID() = %q, want %q", GetUserID(ctx), "user_789")
	}
}

func TestGetMetadataReturnsNilForWrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), MetadataKey, "not a map")

	result := GetMetadata(ctx)
	if result != nil {
		t.Errorf("GetMetadata() should return nil for wrong type, got %v", result)
	}
}

func TestGetStringsWithWrongType(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, RequestIDKey, 123)
	ctx = context.WithValue(ctx, TraceIDKey, []string{"a", "b"})
	ctx = context.WithValue(ctx, UserIDKey, struct{ Name string }{"test"})

	if GetRequestID(ctx) != "" {
		t.Errorf("GetRequestID() should return empty string for wrong type")
	}

	if GetTraceID(ctx) != "" {
		t.Errorf("GetTraceID() should return empty string for wrong type")
	}

	if GetUserID(ctx) != "" {
		t.Errorf("GetUserID() should return empty string for wrong type")
	}
}

func TestGetStartTimeWithWrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), StartTimeKey, "not a time")

	result := GetStartTime(ctx)
	if !result.IsZero() {
		t.Errorf("GetStartTime() should return zero time for wrong type, got %v", result)
	}
}
