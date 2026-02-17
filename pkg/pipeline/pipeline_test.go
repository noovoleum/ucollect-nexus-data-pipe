package pipeline

import (
	"context"
	"testing"
	"time"
)

// MockSource is a mock implementation of Source for testing
type MockSource struct {
	events []Event
}

func NewMockSource(events []Event) *MockSource {
	return &MockSource{events: events}
}

func (m *MockSource) Connect(ctx context.Context) error {
	return nil
}

func (m *MockSource) Read(ctx context.Context) (<-chan Event, <-chan error) {
	events := make(chan Event)
	errors := make(chan error)

	go func() {
		defer close(events)
		defer close(errors)

		for _, event := range m.events {
			select {
			case <-ctx.Done():
				return
			case events <- event:
			}
		}
	}()

	return events, errors
}

func (m *MockSource) Close() error {
	return nil
}

// MockSink is a mock implementation of Sink for testing
type MockSink struct {
	received []Event
}

func NewMockSink() *MockSink {
	return &MockSink{received: make([]Event, 0)}
}

func (m *MockSink) Connect(ctx context.Context) error {
	return nil
}

func (m *MockSink) Write(ctx context.Context, events <-chan Event) <-chan error {
	errors := make(chan error)

	go func() {
		defer close(errors)

		for event := range events {
			m.received = append(m.received, event)
		}
	}()

	return errors
}

func (m *MockSink) Close() error {
	return nil
}

// MockTransformer is a mock implementation of Transformer for testing
type MockTransformer struct {
	prefix string
}

func NewMockTransformer(prefix string) *MockTransformer {
	return &MockTransformer{prefix: prefix}
}

func (m *MockTransformer) Transform(event Event) (Event, error) {
	event.ID = m.prefix + event.ID
	return event, nil
}

// TestPipelineBasic tests basic pipeline functionality
func TestPipelineBasic(t *testing.T) {
	// Create mock events
	events := []Event{
		{
			ID:         "1",
			Timestamp:  time.Now(),
			Operation:  "insert",
			Source:     "test",
			Database:   "testdb",
			Collection: "testcol",
			Data:       map[string]interface{}{"name": "test1"},
		},
		{
			ID:         "2",
			Timestamp:  time.Now(),
			Operation:  "update",
			Source:     "test",
			Database:   "testdb",
			Collection: "testcol",
			Data:       map[string]interface{}{"name": "test2"},
		},
	}

	// Create mock source and sink
	source := NewMockSource(events)
	sink := NewMockSink()

	// Create pipeline
	pipeline := New("test-pipeline", source, sink, nil, nil)

	// Run pipeline
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := pipeline.Run(ctx)
	if err != nil {
		t.Fatalf("Pipeline.Run() error = %v", err)
	}

	// Verify events were received
	if len(sink.received) != len(events) {
		t.Errorf("Expected %d events, got %d", len(events), len(sink.received))
	}
}

// TestPipelineWithTransformer tests pipeline with transformer
func TestPipelineWithTransformer(t *testing.T) {
	// Create mock events
	events := []Event{
		{
			ID:        "1",
			Timestamp: time.Now(),
			Operation: "insert",
			Data:      map[string]interface{}{"name": "test1"},
		},
	}

	// Create mock source, sink, and transformer
	source := NewMockSource(events)
	sink := NewMockSink()
	transformer := NewMockTransformer("PREFIX_")

	// Create pipeline
	pipeline := New("test-pipeline", source, sink, transformer, nil)

	// Run pipeline
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := pipeline.Run(ctx)
	if err != nil {
		t.Fatalf("Pipeline.Run() error = %v", err)
	}

	// Verify transformation was applied
	if len(sink.received) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(sink.received))
	}

	if sink.received[0].ID != "PREFIX_1" {
		t.Errorf("Expected ID 'PREFIX_1', got '%s'", sink.received[0].ID)
	}
}
