package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/IEatCodeDaily/data-pipe/pkg/config"
	"github.com/IEatCodeDaily/data-pipe/pkg/pipeline"
	"github.com/IEatCodeDaily/data-pipe/pkg/sink"
	"github.com/IEatCodeDaily/data-pipe/pkg/source"
	"github.com/IEatCodeDaily/data-pipe/pkg/transform"
)

func main() {
	configPath := flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	logger := log.New(os.Stdout, "[data-pipe] ", log.LstdFlags)

	// Load configuration
	cfg, err := config.LoadFromFile(*configPath)
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	logger.Printf("Loaded configuration for pipeline: %s", cfg.Pipeline.Name)

	// Create source
	var src pipeline.Source
	switch cfg.Source.Type {
	case "mongodb":
		uri := cfg.Source.GetString("uri")
		database := cfg.Source.GetString("database")
		collection := cfg.Source.GetString("collection")
		src = source.NewMongoDBSource(uri, database, collection, logger)
	default:
		logger.Fatalf("Unsupported source type: %s", cfg.Source.Type)
	}

	// Create sink
	var snk pipeline.Sink
	switch cfg.Sink.Type {
	case "postgresql":
		connStr := cfg.Sink.GetString("connection_string")
		table := cfg.Sink.GetString("table")
		snk = sink.NewPostgreSQLSink(connStr, table, logger)
	default:
		logger.Fatalf("Unsupported sink type: %s", cfg.Sink.Type)
	}

	// Create transformer
	var transformer pipeline.Transformer

	if cfg.Transformer.Type != "" {
		switch cfg.Transformer.Type {
		case "fieldmapper":
			// Parse field mapper configuration
			if _, ok := cfg.Transformer.Settings["mappings"]; !ok {
				logger.Fatalf("fieldmapper transformer requires 'mappings' configuration")
			}

			// Convert settings to JSON and parse into FieldMapperConfig
			settingsJSON, err := json.Marshal(cfg.Transformer.Settings)
			if err != nil {
				logger.Fatalf("Failed to marshal transformer settings: %v", err)
			}

			var fmConfig transform.FieldMapperConfig
			if err := json.Unmarshal(settingsJSON, &fmConfig); err != nil {
				logger.Fatalf("Failed to parse fieldmapper configuration: %v", err)
			}

			fm, err := transform.NewFieldMapperWithLogger(fmConfig, logger)
			if err != nil {
				logger.Fatalf("Failed to create field mapper: %v", err)
			}
			transformer = fm
		case "passthrough":
			transformer = transform.NewPassThroughTransformer()
		default:
			logger.Fatalf("Unsupported transformer type: %s", cfg.Transformer.Type)
		}
	} else {
		// Default to passthrough if no transformer configured
		transformer = transform.NewPassThroughTransformer()
	}

	// Create pipeline
	pipe := pipeline.New(cfg.Pipeline.Name, src, snk, transformer, logger)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Println("Received shutdown signal, stopping pipeline...")
		cancel()
	}()

	// Handle initial sync if configured
	if cfg.Pipeline.Sync.InitialSync {
		logger.Println("Initial sync is enabled")

		// Perform initial sync
		if err := performInitialSync(ctx, cfg, src, snk, transformer, logger); err != nil {
			logger.Fatalf("Initial sync failed: %v", err)
		}
	}

	// Run CDC pipeline
	logger.Println("Starting CDC pipeline...")
	if err := pipe.Run(ctx); err != nil {
		logger.Fatalf("Pipeline error: %v", err)
	}

	logger.Println("Pipeline stopped")
	fmt.Println("Goodbye!")
}

// performInitialSync handles the initial synchronization of data
func performInitialSync(ctx context.Context, cfg *config.Config, src pipeline.Source, snk pipeline.Sink, transformer pipeline.Transformer, logger *log.Logger) error {
	// Type assert to access MongoDB-specific methods
	mongoSrc, ok := src.(*source.MongoDBSource)
	if !ok {
		return fmt.Errorf("initial sync is only supported for MongoDB sources")
	}

	pgSink, ok := snk.(*sink.PostgreSQLSink)
	if !ok {
		return fmt.Errorf("initial sync is only supported for PostgreSQL sinks")
	}

	// Ensure connections are established
	if err := mongoSrc.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := pgSink.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Determine initial sync strategy
	var fromTimestamp interface{}

	if cfg.Pipeline.Sync.ForceInitialSync {
		logger.Println("Force initial sync is enabled, syncing all data")
	} else if cfg.Pipeline.Sync.TimestampField != "" {
		// Check if sink table is empty
		isEmpty, err := pgSink.IsTableEmpty(ctx)
		if err != nil {
			return fmt.Errorf("failed to check if sink table is empty: %w", err)
		}

		if isEmpty {
			logger.Println("Sink table is empty, performing full initial sync")
		} else {
			// Get latest timestamp from sink
			ts, err := pgSink.GetLatestTimestamp(ctx, cfg.Pipeline.Sync.TimestampField)
			if err != nil {
				logger.Printf("Warning: failed to get latest timestamp from sink: %v", err)
				logger.Println("Falling back to full initial sync")
			} else if ts != nil {
				fromTimestamp = ts
				logger.Printf("Starting incremental initial sync from timestamp: %v", fromTimestamp)
			} else {
				logger.Println("No timestamp found in sink, performing full initial sync")
			}
		}
	} else {
		logger.Println("No timestamp field configured, performing full initial sync")
	}

	// Prepare initial sync config
	syncConfig := source.InitialSyncConfig{
		Enabled:        true,
		TimestampField: cfg.Pipeline.Sync.TimestampField,
		FromTimestamp:  fromTimestamp,
		BatchSize:      cfg.Pipeline.Sync.BatchSize,
	}

	if syncConfig.BatchSize <= 0 {
		syncConfig.BatchSize = 1000
	}

	// Perform initial sync
	logger.Println("Starting initial sync...")
	events, errors := mongoSrc.PerformInitialSync(ctx, syncConfig)

	// Transform and write events
	transformedEvents := make(chan pipeline.Event)
	go func() {
		defer close(transformedEvents)
		for event := range events {
			if transformer != nil {
				transformed, err := transformer.Transform(event)
				if err != nil {
					logger.Printf("Error transforming event during initial sync: %v", err)
					continue
				}
				event = transformed
			}
			transformedEvents <- event
		}
	}()

	// Write to sink
	sinkErrors := pgSink.Write(ctx, transformedEvents)

	// Handle errors from both channels concurrently
	var wg sync.WaitGroup
	wg.Add(2)
	errorOccurred := false

	go func() {
		defer wg.Done()
		for err := range errors {
			logger.Printf("Initial sync source error: %v", err)
			errorOccurred = true
		}
	}()

	go func() {
		defer wg.Done()
		for err := range sinkErrors {
			logger.Printf("Initial sync sink error: %v", err)
			errorOccurred = true
		}
	}()

	wg.Wait()

	if errorOccurred {
		return fmt.Errorf("errors occurred during initial sync")
	}

	logger.Println("Initial sync completed successfully")
	return nil
}
