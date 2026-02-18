package source

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/IEatCodeDaily/data-pipe/pkg/pipeline"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBSource implements the Source interface for MongoDB
type MongoDBSource struct {
	uri        string
	database   string
	collection string
	client     *mongo.Client
	logger     *log.Logger
}

// InitialSyncConfig contains configuration for initial sync
type InitialSyncConfig struct {
	Enabled        bool
	TimestampField string
	FromTimestamp  interface{}
	BatchSize      int
}

// NewMongoDBSource creates a new MongoDB source
func NewMongoDBSource(uri, database, collection string, logger *log.Logger) *MongoDBSource {
	if logger == nil {
		logger = log.Default()
	}
	return &MongoDBSource{
		uri:        uri,
		database:   database,
		collection: collection,
		logger:     logger,
	}
}

// Connect establishes connection to MongoDB
func (m *MongoDBSource) Connect(ctx context.Context) error {
	m.logger.Printf("Connecting to MongoDB: %s", m.uri)

	clientOptions := options.Client().ApplyURI(m.uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	m.client = client
	m.logger.Println("Successfully connected to MongoDB")
	return nil
}

// Read reads change events from MongoDB using change streams
func (m *MongoDBSource) Read(ctx context.Context) (<-chan pipeline.Event, <-chan error) {
	events := make(chan pipeline.Event)
	errors := make(chan error)

	go func() {
		defer close(events)
		defer close(errors)

		collection := m.client.Database(m.database).Collection(m.collection)

		// Create a change stream
		pipeline := mongo.Pipeline{}
		opts := options.ChangeStream().SetFullDocument(options.UpdateLookup)

		m.logger.Printf("Starting change stream for %s.%s", m.database, m.collection)
		stream, err := collection.Watch(ctx, pipeline, opts)
		if err != nil {
			errors <- fmt.Errorf("failed to create change stream: %w", err)
			return
		}
		defer stream.Close(ctx)

		for stream.Next(ctx) {
			var changeDoc bson.M
			if err := stream.Decode(&changeDoc); err != nil {
				errors <- fmt.Errorf("failed to decode change event: %w", err)
				continue
			}

			event := m.convertChangeEvent(changeDoc)
			events <- event
		}

		if err := stream.Err(); err != nil {
			errors <- fmt.Errorf("change stream error: %w", err)
		}
	}()

	return events, errors
}

// convertChangeEvent converts MongoDB change stream event to pipeline event
func (m *MongoDBSource) convertChangeEvent(changeDoc bson.M) pipeline.Event {
	event := pipeline.Event{
		Source:     "mongodb",
		Database:   m.database,
		Collection: m.collection,
		Timestamp:  time.Now(),
	}

	if id, ok := changeDoc["_id"]; ok {
		event.ID = fmt.Sprintf("%v", id)
	}

	if opType, ok := changeDoc["operationType"].(string); ok {
		event.Operation = opType
	}

	if fullDoc, ok := changeDoc["fullDocument"].(bson.M); ok {
		event.Data = convertBSONToMap(fullDoc)
	}

	if updateDesc, ok := changeDoc["updateDescription"].(bson.M); ok {
		if updatedFields, ok := updateDesc["updatedFields"].(bson.M); ok {
			if event.Data == nil {
				event.Data = make(map[string]interface{})
			}
			for k, v := range convertBSONToMap(updatedFields) {
				event.Data[k] = v
			}
		}
	}

	return event
}

// convertBSONToMap converts BSON document to map
func convertBSONToMap(doc bson.M) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range doc {
		result[k] = v
	}
	return result
}

// Close closes the MongoDB connection
func (m *MongoDBSource) Close() error {
	if m.client != nil {
		m.logger.Println("Closing MongoDB connection")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return m.client.Disconnect(ctx)
	}
	return nil
}

// PerformInitialSync performs initial synchronization of existing data
func (m *MongoDBSource) PerformInitialSync(ctx context.Context, config InitialSyncConfig) (<-chan pipeline.Event, <-chan error) {
	events := make(chan pipeline.Event)
	errors := make(chan error)

	go func() {
		defer close(events)
		defer close(errors)

		collection := m.client.Database(m.database).Collection(m.collection)

		// Build query filter
		filter := bson.M{}
		if config.TimestampField != "" && config.FromTimestamp != nil {
			filter[config.TimestampField] = bson.M{"$gte": config.FromTimestamp}
			m.logger.Printf("Starting initial sync from timestamp: %v on field: %s", config.FromTimestamp, config.TimestampField)
		} else {
			m.logger.Printf("Starting full initial sync for %s.%s", m.database, m.collection)
		}

		// Set batch size
		batchSize := config.BatchSize
		if batchSize <= 0 {
			batchSize = 1000
		}

		// Query with cursor
		opts := options.Find().SetBatchSize(int32(batchSize))
		if config.TimestampField != "" {
			// Sort by timestamp field to ensure ordered processing
			opts.SetSort(bson.D{bson.E{Key: config.TimestampField, Value: 1}})
		}

		cursor, err := collection.Find(ctx, filter, opts)
		if err != nil {
			errors <- fmt.Errorf("failed to query MongoDB for initial sync: %w", err)
			return
		}
		defer cursor.Close(ctx)

		count := 0
		for cursor.Next(ctx) {
			var doc bson.M
			if err := cursor.Decode(&doc); err != nil {
				errors <- fmt.Errorf("failed to decode document: %w", err)
				continue
			}

			// Convert to pipeline event
			event := pipeline.Event{
				ID:         fmt.Sprintf("%v", doc["_id"]),
				Timestamp:  time.Now(),
				Operation:  "insert", // Initial sync is treated as insert
				Source:     "mongodb",
				Database:   m.database,
				Collection: m.collection,
				Data:       convertBSONToMap(doc),
			}

			events <- event
			count++

			if count%1000 == 0 {
				m.logger.Printf("Initial sync progress: %d documents synced", count)
			}
		}

		if err := cursor.Err(); err != nil {
			errors <- fmt.Errorf("cursor error during initial sync: %w", err)
			return
		}

		m.logger.Printf("Initial sync completed: %d documents synced", count)
	}()

	return events, errors
}

// GetLatestTimestamp retrieves the latest timestamp from the collection
func (m *MongoDBSource) GetLatestTimestamp(ctx context.Context, timestampField string) (interface{}, error) {
	if timestampField == "" {
		return nil, fmt.Errorf("timestamp field is required")
	}

	collection := m.client.Database(m.database).Collection(m.collection)

	opts := options.FindOne().SetSort(bson.D{bson.E{Key: timestampField, Value: -1}})
	var result bson.M
	err := collection.FindOne(ctx, bson.M{}, opts).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Collection is empty
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest timestamp: %w", err)
	}

	timestamp, ok := result[timestampField]
	if !ok {
		return nil, fmt.Errorf("timestamp field '%s' not found in document", timestampField)
	}

	return timestamp, nil
}
