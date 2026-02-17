package pipeline

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// MetricsRecorder interface for recording pipeline metrics
type MetricsRecorder interface {
	RecordEventProcessed(pipelineName, operation string)
	RecordEventError(pipelineName, component, errorType string)
	RecordProcessingDuration(pipelineName, component string, duration float64)
	SetPipelineRunning(running bool)
	SetSourceConnected(connected bool)
	SetSinkConnected(connected bool)
}

// Pipeline represents a data pipeline from source to sink
type Pipeline struct {
	name            string
	source          Source
	sink            Sink
	transformer     Transformer
	logger          *log.Logger
	metrics         MetricsRecorder
	startTime       time.Time
	mu              sync.RWMutex // protects the fields below
	lastEventTime   time.Time
	sourceConnected bool
	sinkConnected   bool
}

// New creates a new pipeline
func New(name string, source Source, sink Sink, transformer Transformer, logger *log.Logger) *Pipeline {
	if logger == nil {
		logger = log.Default()
	}
	return &Pipeline{
		name:        name,
		source:      source,
		sink:        sink,
		transformer: transformer,
		logger:      logger,
		startTime:   time.Now(),
	}
}

// SetMetrics sets the metrics recorder for the pipeline
func (p *Pipeline) SetMetrics(metrics MetricsRecorder) {
	p.metrics = metrics
}

// IsHealthy returns true if the pipeline is healthy
func (p *Pipeline) IsHealthy() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.sourceConnected && p.sinkConnected
}

// GetStatus returns the current health status of the pipeline
func (p *Pipeline) GetStatus() HealthStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	uptime := time.Since(p.startTime).Seconds()
	
	var lastEventTimeStr string
	if !p.lastEventTime.IsZero() {
		lastEventTimeStr = p.lastEventTime.Format(time.RFC3339)
	}
	
	return HealthStatus{
		Healthy:          p.sourceConnected && p.sinkConnected,
		PipelineRunning:  p.sourceConnected && p.sinkConnected,
		SourceConnected:  p.sourceConnected,
		SinkConnected:    p.sinkConnected,
		LastEventTime:    lastEventTimeStr,
		UptimeSeconds:    int64(uptime),
	}
}

// HealthStatus represents the health status of the pipeline
type HealthStatus struct {
	Healthy          bool   `json:"healthy"`
	PipelineRunning  bool   `json:"pipeline_running"`
	SourceConnected  bool   `json:"source_connected"`
	SinkConnected    bool   `json:"sink_connected"`
	LastEventTime    string `json:"last_event_time,omitempty"`
	UptimeSeconds    int64  `json:"uptime_seconds"`
}

// Run starts the pipeline
func (p *Pipeline) Run(ctx context.Context) error {
	p.logger.Printf("Starting pipeline: %s", p.name)
	
	// Set pipeline status to running
	if p.metrics != nil {
		p.metrics.SetPipelineRunning(true)
		defer p.metrics.SetPipelineRunning(false)
	}

	// Connect source
	startTime := time.Now()
	if err := p.source.Connect(ctx); err != nil {
		if p.metrics != nil {
			p.metrics.RecordEventError(p.name, "source", "connection_error")
			p.metrics.SetSourceConnected(false)
		}
		return fmt.Errorf("failed to connect source: %w", err)
	}
	p.mu.Lock()
	p.sourceConnected = true
	p.mu.Unlock()
	if p.metrics != nil {
		p.metrics.SetSourceConnected(true)
		p.metrics.RecordProcessingDuration(p.name, "source_connect", time.Since(startTime).Seconds())
	}
	defer func() {
		p.source.Close()
		p.mu.Lock()
		p.sourceConnected = false
		p.mu.Unlock()
		if p.metrics != nil {
			p.metrics.SetSourceConnected(false)
		}
	}()

	// Connect sink
	startTime = time.Now()
	if err := p.sink.Connect(ctx); err != nil {
		if p.metrics != nil {
			p.metrics.RecordEventError(p.name, "sink", "connection_error")
			p.metrics.SetSinkConnected(false)
		}
		return fmt.Errorf("failed to connect sink: %w", err)
	}
	p.mu.Lock()
	p.sinkConnected = true
	p.mu.Unlock()
	if p.metrics != nil {
		p.metrics.SetSinkConnected(true)
		p.metrics.RecordProcessingDuration(p.name, "sink_connect", time.Since(startTime).Seconds())
	}
	defer func() {
		p.sink.Close()
		p.mu.Lock()
		p.sinkConnected = false
		p.mu.Unlock()
		if p.metrics != nil {
			p.metrics.SetSinkConnected(false)
		}
	}()

	// Start reading from source
	events, sourceErrors := p.source.Read(ctx)

	// Transform events if transformer is provided
	transformedEvents := make(chan Event)
	go func() {
		defer close(transformedEvents)
		for event := range events {
			eventStartTime := time.Now()
			p.mu.Lock()
			p.lastEventTime = eventStartTime
			p.mu.Unlock()
			
			if p.transformer != nil {
				transformed, err := p.transformer.Transform(event)
				if err != nil {
					p.logger.Printf("Error transforming event: %v", err)
					if p.metrics != nil {
						p.metrics.RecordEventError(p.name, "transformer", "transform_error")
					}
					continue
				}
				event = transformed
				if p.metrics != nil {
					p.metrics.RecordProcessingDuration(p.name, "transform", time.Since(eventStartTime).Seconds())
				}
			}
			
			// Record event processed by operation type
			if p.metrics != nil {
				p.metrics.RecordEventProcessed(p.name, event.Operation)
			}
			
			transformedEvents <- event
		}
	}()

	// Write to sink
	sinkErrors := p.sink.Write(ctx, transformedEvents)

	// Handle errors
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for err := range sourceErrors {
			p.logger.Printf("Source error: %v", err)
			if p.metrics != nil {
				p.metrics.RecordEventError(p.name, "source", "read_error")
			}
		}
	}()

	go func() {
		defer wg.Done()
		for err := range sinkErrors {
			p.logger.Printf("Sink error: %v", err)
			if p.metrics != nil {
				p.metrics.RecordEventError(p.name, "sink", "write_error")
			}
		}
	}()

	wg.Wait()
	p.logger.Printf("Pipeline stopped: %s", p.name)
	return nil
}
