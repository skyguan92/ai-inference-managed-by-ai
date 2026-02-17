package cache

import (
	"context"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	ctx := context.Background()

	c := New()
	if c == nil {
		t.Fatal("New() returned nil")
	}

	size := c.Size(ctx)
	if size != 0 {
		t.Errorf("expected size 0, got %d", size)
	}
}

func TestNewWithOptions(t *testing.T) {
	c := New(WithTTL(time.Hour), WithMaxSize(100))
	if c == nil {
		t.Fatal("New() returned nil")
	}

	ctx := context.Background()
	c.Set(ctx, "key1", "value1", 0)
	size := c.Size(ctx)
	if size != 1 {
		t.Errorf("expected size 1, got %d", size)
	}
}

func TestCache_Get_Set(t *testing.T) {
	ctx := context.Background()
	c := New()

	c.Set(ctx, "key1", "value1", time.Hour)
	val, found := c.Get(ctx, "key1")
	if !found {
		t.Error("expected to find key1")
	}
	if val != "value1" {
		t.Errorf("expected value1, got %v", val)
	}

	val, found = c.Get(ctx, "nonexistent")
	if found {
		t.Error("expected not to find nonexistent key")
	}
	if val != nil {
		t.Errorf("expected nil, got %v", val)
	}
}

func TestCache_Delete(t *testing.T) {
	ctx := context.Background()
	c := New()

	c.Set(ctx, "key1", "value1", time.Hour)
	size := c.Size(ctx)
	if size != 1 {
		t.Fatalf("expected size 1, got %d", size)
	}

	c.Delete(ctx, "key1")
	size = c.Size(ctx)
	if size != 0 {
		t.Errorf("expected size 0 after delete, got %d", size)
	}

	val, found := c.Get(ctx, "key1")
	if found {
		t.Error("expected not to find key1 after delete")
	}
	if val != nil {
		t.Errorf("expected nil, got %v", val)
	}
}

func TestCache_Expiration(t *testing.T) {
	ctx := context.Background()
	c := New()

	c.Set(ctx, "key1", "value1", 50*time.Millisecond)

	val, found := c.Get(ctx, "key1")
	if !found {
		t.Error("expected to find key1 before expiration")
	}
	if val != "value1" {
		t.Errorf("expected value1, got %v", val)
	}

	time.Sleep(100 * time.Millisecond)

	val, found = c.Get(ctx, "key1")
	if found {
		t.Error("expected not to find key1 after expiration")
	}
	if val != nil {
		t.Errorf("expected nil, got %v", val)
	}
}

func TestCache_NoExpiration(t *testing.T) {
	ctx := context.Background()
	c := New()

	c.Set(ctx, "key1", "value1", 0)

	time.Sleep(50 * time.Millisecond)

	val, found := c.Get(ctx, "key1")
	if !found {
		t.Error("expected to find key1 without expiration")
	}
	if val != "value1" {
		t.Errorf("expected value1, got %v", val)
	}
}

func TestCache_Clear(t *testing.T) {
	ctx := context.Background()
	c := New()

	c.Set(ctx, "key1", "value1", time.Hour)
	c.Set(ctx, "key2", "value2", time.Hour)
	c.Set(ctx, "key3", "value3", time.Hour)

	size := c.Size(ctx)
	if size != 3 {
		t.Fatalf("expected size 3, got %d", size)
	}

	c.Clear(ctx)
	size = c.Size(ctx)
	if size != 0 {
		t.Errorf("expected size 0 after clear, got %d", size)
	}
}

func TestCache_Size(t *testing.T) {
	ctx := context.Background()
	c := New()

	size := c.Size(ctx)
	if size != 0 {
		t.Errorf("expected size 0, got %d", size)
	}

	c.Set(ctx, "key1", "value1", time.Hour)
	size = c.Size(ctx)
	if size != 1 {
		t.Errorf("expected size 1, got %d", size)
	}

	c.Set(ctx, "key2", "value2", time.Hour)
	size = c.Size(ctx)
	if size != 2 {
		t.Errorf("expected size 2, got %d", size)
	}

	c.Delete(ctx, "key1")
	size = c.Size(ctx)
	if size != 1 {
		t.Errorf("expected size 1 after delete, got %d", size)
	}
}
