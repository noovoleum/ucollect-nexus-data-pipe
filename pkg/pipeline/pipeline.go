package pipeline

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// Pipeline represents a data pipeline from source to sink
type Pipeline struct {
	name        string
	source      Source
	sink        Sink
	transformer Transformer
	logger      *log.Logger
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
	}
}

// Run starts the pipeline
func (p *Pipeline) Run(ctx context.Context) error {
	p.logger.Printf("Starting pipeline: %s", p.name)

	// Connect source
	if err := p.source.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect source: %w", err)
	}
	defer p.source.Close()

	// Connect sink
	if err := p.sink.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect sink: %w", err)
	}
	defer p.sink.Close()

	// Start reading from source
	events, sourceErrors := p.source.Read(ctx)

	// Transform events if transformer is provided
	transformedEvents := make(chan Event)
	go func() {
		defer close(transformedEvents)
		for event := range events {
			if p.transformer != nil {
				transformed, err := p.transformer.Transform(event)
				if err != nil {
					p.logger.Printf("Error transforming event: %v", err)
					continue
				}
				event = transformed
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
		}
	}()

	go func() {
		defer wg.Done()
		for err := range sinkErrors {
			p.logger.Printf("Sink error: %v", err)
		}
	}()

	wg.Wait()
	p.logger.Printf("Pipeline stopped: %s", p.name)
	return nil
}
