package unit

import (
	"context"
	"time"
)

type Schema struct {
	Type       string           `json:"type"`
	Properties map[string]Field `json:"properties,omitempty"`
	Items      *Schema          `json:"items,omitempty"`
	Required   []string         `json:"required,omitempty"`
	Optional   []string         `json:"optional,omitempty"`

	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`

	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	MinLength *int     `json:"minLength,omitempty"`
	MaxLength *int     `json:"maxLength,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	Enum      []any    `json:"enum,omitempty"`
	Default   any      `json:"default,omitempty"`
	Examples  []any    `json:"examples,omitempty"`
}

type Field struct {
	Schema
	Name string `json:"name"`
}

type Example struct {
	Input       any    `json:"input"`
	Output      any    `json:"output"`
	Description string `json:"description,omitempty"`
}

type ResourceUpdate struct {
	URI       string    `json:"uri"`
	Timestamp time.Time `json:"timestamp"`
	Operation string    `json:"operation"`
	Data      any       `json:"data,omitempty"`
	Error     error     `json:"error,omitempty"`
}

type Command interface {
	Name() string
	Domain() string
	InputSchema() Schema
	OutputSchema() Schema
	Execute(ctx context.Context, input any) (output any, err error)
	Description() string
	Examples() []Example
}

type Query interface {
	Name() string
	Domain() string
	InputSchema() Schema
	OutputSchema() Schema
	Execute(ctx context.Context, input any) (output any, err error)
	Description() string
	Examples() []Example
}

type Event interface {
	Type() string
	Domain() string
	Payload() any
	Timestamp() time.Time
	CorrelationID() string
}

type Resource interface {
	URI() string
	Domain() string
	Schema() Schema
	Get(ctx context.Context) (any, error)
	Watch(ctx context.Context) (<-chan ResourceUpdate, error)
}

// ResourceFactory creates Resource instances dynamically based on URI patterns.
// It is used to handle dynamic URI patterns like asms://model/{id}.
type ResourceFactory interface {
	// CanCreate returns true if this factory can create a resource for the given URI
	CanCreate(uri string) bool
	// Create creates a new Resource instance for the given URI
	Create(uri string) (Resource, error)
	// Pattern returns the URI pattern this factory handles (e.g., "asms://model/*")
	Pattern() string
}
