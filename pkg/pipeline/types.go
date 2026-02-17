package pipeline

import (
	"context"
	"time"
)

// Event represents a change data capture event
type Event struct {
	ID         string                 `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	Operation  string                 `json:"operation"` // insert, update, delete
	Source     string                 `json:"source"`
	Database   string                 `json:"database"`
	Collection string                 `json:"collection"`
	Data       map[string]interface{} `json:"data"`
	Before     map[string]interface{} `json:"before,omitempty"` // for updates
}

// Source defines the interface for data sources
type Source interface {
	// Connect establishes connection to the source
	Connect(ctx context.Context) error
	// Read returns a channel that emits change events
	Read(ctx context.Context) (<-chan Event, <-chan error)
	// Close closes the source connection
	Close() error
}

// Sink defines the interface for data sinks
type Sink interface {
	// Connect establishes connection to the sink
	Connect(ctx context.Context) error
	// Write writes events to the sink
	Write(ctx context.Context, events <-chan Event) <-chan error
	// Close closes the sink connection
	Close() error
}

// Transformer defines the interface for data transformation
type Transformer interface {
	// Transform transforms an event
	Transform(event Event) (Event, error)
}
