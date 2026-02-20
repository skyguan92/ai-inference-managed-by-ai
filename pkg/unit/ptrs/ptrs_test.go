package ptrs

import (
	"testing"
)

func TestInt(t *testing.T) {
	v := 42
	p := Int(v)
	if p == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *p != 42 {
		t.Errorf("expected *p == 42, got %d", *p)
	}
}

func TestInt32(t *testing.T) {
	v := int32(100)
	p := Int32(v)
	if p == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *p != 100 {
		t.Errorf("expected *p == 100, got %d", *p)
	}
}

func TestInt64(t *testing.T) {
	v := int64(9999)
	p := Int64(v)
	if p == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *p != 9999 {
		t.Errorf("expected *p == 9999, got %d", *p)
	}
}

func TestFloat32(t *testing.T) {
	v := float32(3.14)
	p := Float32(v)
	if p == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *p != v {
		t.Errorf("expected *p == %v, got %v", v, *p)
	}
}

func TestFloat64(t *testing.T) {
	v := float64(2.718)
	p := Float64(v)
	if p == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *p != v {
		t.Errorf("expected *p == %v, got %v", v, *p)
	}
}

func TestString(t *testing.T) {
	v := "hello"
	p := String(v)
	if p == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *p != "hello" {
		t.Errorf("expected *p == 'hello', got %q", *p)
	}
}

func TestBool(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		p := Bool(true)
		if p == nil {
			t.Fatal("expected non-nil pointer")
		}
		if !*p {
			t.Error("expected *p == true")
		}
	})

	t.Run("false", func(t *testing.T) {
		p := Bool(false)
		if p == nil {
			t.Fatal("expected non-nil pointer")
		}
		if *p {
			t.Error("expected *p == false")
		}
	})
}

func TestPointersAreIndependent(t *testing.T) {
	// Verify that modifying the original doesn't affect the pointer
	v := 10
	p := Int(v)
	v = 20
	_ = v // intentional: v is reassigned to prove pointer independence
	if *p != 10 {
		t.Errorf("expected pointer to be independent: *p = %d, want 10", *p)
	}
}
