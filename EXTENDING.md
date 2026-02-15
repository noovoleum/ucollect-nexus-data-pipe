# Extending the Data Pipeline

This guide shows how to add new source and sink connectors to the data pipeline.

## Adding a New Source Connector

To add a new source connector (e.g., Convex database), implement the `Source` interface:

```go
type Source interface {
    Connect(ctx context.Context) error
    Read(ctx context.Context) (<-chan Event, <-chan error)
    Close() error
}
```

### Example: Convex Source

Create a new file `pkg/source/convex.go`:

```go
package source

import (
    "context"
    "log"
    
    "github.com/IEatCodeDaily/data-pipe/pkg/pipeline"
)

type ConvexSource struct {
    endpoint string
    apiKey   string
    table    string
    logger   *log.Logger
}

func NewConvexSource(endpoint, apiKey, table string, logger *log.Logger) *ConvexSource {
    if logger == nil {
        logger = log.Default()
    }
    return &ConvexSource{
        endpoint: endpoint,
        apiKey:   apiKey,
        table:    table,
        logger:   logger,
    }
}

func (c *ConvexSource) Connect(ctx context.Context) error {
    c.logger.Printf("Connecting to Convex: %s", c.endpoint)
    // Implement connection logic
    return nil
}

func (c *ConvexSource) Read(ctx context.Context) (<-chan pipeline.Event, <-chan error) {
    events := make(chan pipeline.Event)
    errors := make(chan error)
    
    go func() {
        defer close(events)
        defer close(errors)
        
        // Implement change stream/subscription logic
        // Convert Convex changes to pipeline.Event and send to events channel
    }()
    
    return events, errors
}

func (c *ConvexSource) Close() error {
    c.logger.Println("Closing Convex connection")
    return nil
}
```

### Register the Source

Update `cmd/data-pipe/main.go` to support the new source:

```go
// Create source
var src pipeline.Source
switch cfg.Source.Type {
case "mongodb":
    uri := cfg.Source.GetString("uri")
    database := cfg.Source.GetString("database")
    collection := cfg.Source.GetString("collection")
    src = source.NewMongoDBSource(uri, database, collection, logger)
case "convex":
    endpoint := cfg.Source.GetString("endpoint")
    apiKey := cfg.Source.GetString("api_key")
    table := cfg.Source.GetString("table")
    src = source.NewConvexSource(endpoint, apiKey, table, logger)
default:
    logger.Fatalf("Unsupported source type: %s", cfg.Source.Type)
}
```

### Configuration Example

```json
{
  "pipeline": {
    "name": "convex-to-postgresql"
  },
  "source": {
    "type": "convex",
    "settings": {
      "endpoint": "https://my-project.convex.cloud",
      "api_key": "your-api-key",
      "table": "users"
    }
  },
  "sink": {
    "type": "postgresql",
    "settings": {
      "connection_string": "host=localhost port=5432 user=postgres password=postgres dbname=mydb sslmode=disable",
      "table": "users"
    }
  }
}
```

## Adding a New Sink Connector

To add a new sink connector (e.g., ClickHouse), implement the `Sink` interface:

```go
type Sink interface {
    Connect(ctx context.Context) error
    Write(ctx context.Context, events <-chan Event) <-chan error
    Close() error
}
```

### Example: ClickHouse Sink

Create a new file `pkg/sink/clickhouse.go`:

```go
package sink

import (
    "context"
    "fmt"
    "log"
    "regexp"
    
    "github.com/IEatCodeDaily/data-pipe/pkg/pipeline"
)

// Validate table name to prevent SQL injection
var validTableName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]{0,62}$`)

type ClickHouseSink struct {
    connStr   string
    table     string
    logger    *log.Logger
    batchSize int
}

func NewClickHouseSink(connStr, table string, logger *log.Logger) *ClickHouseSink {
    if logger == nil {
        logger = log.Default()
    }
    return &ClickHouseSink{
        connStr:   connStr,
        table:     table,
        logger:    logger,
        batchSize: 1000, // ClickHouse benefits from larger batches
    }
}

func (c *ClickHouseSink) Connect(ctx context.Context) error {
    c.logger.Println("Connecting to ClickHouse")
    
    // Validate table name to prevent SQL injection
    if !validTableName.MatchString(c.table) {
        return fmt.Errorf("invalid table name: %s", c.table)
    }
    
    // Implement ClickHouse connection logic
    return nil
}

func (c *ClickHouseSink) Write(ctx context.Context, events <-chan pipeline.Event) <-chan error {
    errors := make(chan error)
    
    go func() {
        defer close(errors)
        
        batch := make([]pipeline.Event, 0, c.batchSize)
        
        for event := range events {
            batch = append(batch, event)
            
            if len(batch) >= c.batchSize {
                if err := c.writeBatch(ctx, batch); err != nil {
                    errors <- err
                }
                batch = batch[:0]
            }
        }
        
        // Write remaining events
        if len(batch) > 0 {
            if err := c.writeBatch(ctx, batch); err != nil {
                errors <- err
            }
        }
    }()
    
    return errors
}

func (c *ClickHouseSink) writeBatch(ctx context.Context, events []pipeline.Event) error {
    // Implement batch insert logic for ClickHouse
    c.logger.Printf("Writing %d events to ClickHouse", len(events))
    return nil
}

func (c *ClickHouseSink) Close() error {
    c.logger.Println("Closing ClickHouse connection")
    return nil
}
```

### Register the Sink

Update `cmd/data-pipe/main.go`:

```go
// Create sink
var snk pipeline.Sink
switch cfg.Sink.Type {
case "postgresql":
    connStr := cfg.Sink.GetString("connection_string")
    table := cfg.Sink.GetString("table")
    snk = sink.NewPostgreSQLSink(connStr, table, logger)
case "clickhouse":
    connStr := cfg.Sink.GetString("connection_string")
    table := cfg.Sink.GetString("table")
    snk = sink.NewClickHouseSink(connStr, table, logger)
default:
    logger.Fatalf("Unsupported sink type: %s", cfg.Sink.Type)
}
```

## Adding Custom Transformers

Implement the `Transformer` interface:

```go
type Transformer interface {
    Transform(event Event) (Event, error)
}
```

### Example: Field Mapping Transformer

```go
package transform

import (
    "github.com/IEatCodeDaily/data-pipe/pkg/pipeline"
)

type FieldMapper struct {
    mappings map[string]string // old_field -> new_field
}

func NewFieldMapper(mappings map[string]string) *FieldMapper {
    return &FieldMapper{mappings: mappings}
}

func (f *FieldMapper) Transform(event pipeline.Event) (pipeline.Event, error) {
    newData := make(map[string]interface{})
    
    for oldField, newField := range f.mappings {
        if value, exists := event.Data[oldField]; exists {
            newData[newField] = value
        }
    }
    
    // Copy unmapped fields
    for key, value := range event.Data {
        if _, isMapped := f.mappings[key]; !isMapped {
            newData[key] = value
        }
    }
    
    event.Data = newData
    return event, nil
}
```

## Testing New Connectors

Always add tests for new connectors:

```go
package source

import (
    "context"
    "testing"
)

func TestConvexSourceConnect(t *testing.T) {
    source := NewConvexSource("https://test.convex.cloud", "test-key", "test_table", nil)
    
    ctx := context.Background()
    err := source.Connect(ctx)
    
    // Add assertions
    if err != nil {
        t.Errorf("Connect() error = %v", err)
    }
}
```

## Security Considerations

When adding new connectors, always:

1. **Validate inputs**: Use regex patterns to validate table/collection names
2. **Use parameterized queries**: Never concatenate user input into SQL/queries
3. **Handle credentials securely**: Don't log sensitive information
4. **Add connection timeouts**: Use context deadlines for all operations
5. **Test error cases**: Include tests for malicious inputs

## Dependencies

When adding dependencies for new connectors:

```bash
go get github.com/clickhouse/clickhouse-go/v2
go mod tidy
```

Make sure to run security checks on new dependencies.
