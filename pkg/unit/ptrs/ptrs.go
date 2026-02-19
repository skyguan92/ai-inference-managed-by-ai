// Package ptrs provides tiny helper functions that take a value of a primitive
// type and return a pointer to it. These helpers eliminate the repeated
// single-line pattern
//
//	func ptrFoo(v Foo) *Foo { return &v }
//
// that was duplicated across many packages in pkg/unit.
package ptrs

// Int returns a pointer to v.
func Int(v int) *int { return &v }

// Int32 returns a pointer to v.
func Int32(v int32) *int32 { return &v }

// Int64 returns a pointer to v.
func Int64(v int64) *int64 { return &v }

// Float32 returns a pointer to v.
func Float32(v float32) *float32 { return &v }

// Float64 returns a pointer to v.
func Float64(v float64) *float64 { return &v }

// String returns a pointer to v.
func String(v string) *string { return &v }

// Bool returns a pointer to v.
func Bool(v bool) *bool { return &v }
