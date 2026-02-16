package sink

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/IEatCodeDaily/data-pipe/pkg/pipeline"
	_ "github.com/lib/pq"
)

// Valid table name pattern (alphanumeric, underscore, max 63 chars for PostgreSQL)
var validTableName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]{0,62}$`)

// PostgreSQLSink implements the Sink interface for PostgreSQL
type PostgreSQLSink struct {
	connStr   string
	table     string
	db        *sql.DB
	logger    *log.Logger
	batchSize int
}

// NewPostgreSQLSink creates a new PostgreSQL sink
func NewPostgreSQLSink(connStr, table string, logger *log.Logger) *PostgreSQLSink {
	if logger == nil {
		logger = log.Default()
	}
	return &PostgreSQLSink{
		connStr:   connStr,
		table:     table,
		logger:    logger,
		batchSize: 100,
	}
}

// Connect establishes connection to PostgreSQL
func (p *PostgreSQLSink) Connect(ctx context.Context) error {
	p.logger.Println("Connecting to PostgreSQL")

	// Validate table name to prevent SQL injection
	if !validTableName.MatchString(p.table) {
		return fmt.Errorf("invalid table name: %s (must be alphanumeric with underscores, starting with letter or underscore)", p.table)
	}

	db, err := sql.Open("postgres", p.connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	p.db = db
	p.logger.Println("Successfully connected to PostgreSQL")
	return nil
}

// Write writes events to PostgreSQL
func (p *PostgreSQLSink) Write(ctx context.Context, events <-chan pipeline.Event) <-chan error {
	errors := make(chan error)

	go func() {
		defer close(errors)

		batch := make([]pipeline.Event, 0, p.batchSize)

		for event := range events {
			batch = append(batch, event)

			if len(batch) >= p.batchSize {
				if err := p.writeBatch(ctx, batch); err != nil {
					errors <- err
				}
				batch = batch[:0]
			}
		}

		// Write remaining events
		if len(batch) > 0 {
			if err := p.writeBatch(ctx, batch); err != nil {
				errors <- err
			}
		}
	}()

	return errors
}

// writeBatch writes a batch of events to PostgreSQL
func (p *PostgreSQLSink) writeBatch(ctx context.Context, events []pipeline.Event) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(); rbErr != nil && rbErr != sql.ErrTxDone {
			p.logger.Printf("Warning: failed to rollback transaction: %v", rbErr)
		}
	}()

	for _, event := range events {
		if err := p.writeEvent(ctx, tx, event); err != nil {
			return fmt.Errorf("failed to write event: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	p.logger.Printf("Wrote %d events to PostgreSQL", len(events))
	return nil
}

// writeEvent writes a single event to PostgreSQL
func (p *PostgreSQLSink) writeEvent(ctx context.Context, tx *sql.Tx, event pipeline.Event) error {
	switch event.Operation {
	case "insert":
		return p.insertEvent(ctx, tx, event)
	case "update", "replace":
		return p.upsertEvent(ctx, tx, event)
	case "delete":
		return p.deleteEvent(ctx, tx, event)
	default:
		p.logger.Printf("Unknown operation type: %s", event.Operation)
		return nil
	}
}

// insertEvent inserts a new record
func (p *PostgreSQLSink) insertEvent(ctx context.Context, tx *sql.Tx, event pipeline.Event) error {
	if len(event.Data) == 0 {
		return nil
	}

	columns := make([]string, 0, len(event.Data))
	placeholders := make([]string, 0, len(event.Data))
	values := make([]interface{}, 0, len(event.Data))

	i := 1
	for key, value := range event.Data {
		columns = append(columns, key)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		values = append(values, value)
		i++
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (_id) DO UPDATE SET %s",
		p.table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
		p.buildUpdateClause(columns),
	)

	_, err := tx.ExecContext(ctx, query, values...)
	return err
}

// upsertEvent updates or inserts a record
func (p *PostgreSQLSink) upsertEvent(ctx context.Context, tx *sql.Tx, event pipeline.Event) error {
	return p.insertEvent(ctx, tx, event) // Same as insert with upsert logic
}

// deleteEvent deletes a record
func (p *PostgreSQLSink) deleteEvent(ctx context.Context, tx *sql.Tx, event pipeline.Event) error {
	if id, ok := event.Data["_id"]; ok {
		query := fmt.Sprintf("DELETE FROM %s WHERE _id = $1", p.table)
		_, err := tx.ExecContext(ctx, query, id)
		return err
	}
	return nil
}

// buildUpdateClause builds the SET clause for upsert
func (p *PostgreSQLSink) buildUpdateClause(columns []string) string {
	updates := make([]string, 0, len(columns))
	for _, col := range columns {
		if col != "_id" {
			updates = append(updates, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
		}
	}
	return strings.Join(updates, ", ")
}

// Close closes the PostgreSQL connection
func (p *PostgreSQLSink) Close() error {
	if p.db != nil {
		p.logger.Println("Closing PostgreSQL connection")
		return p.db.Close()
	}
	return nil
}

// GetLatestTimestamp retrieves the latest timestamp from the table
func (p *PostgreSQLSink) GetLatestTimestamp(ctx context.Context, timestampField string) (interface{}, error) {
	if timestampField == "" {
		return nil, fmt.Errorf("timestamp field is required")
	}

	// Validate field name to prevent SQL injection
	if !validTableName.MatchString(timestampField) {
		return nil, fmt.Errorf("invalid timestamp field name: %s", timestampField)
	}

	query := fmt.Sprintf("SELECT %s FROM %s ORDER BY %s DESC LIMIT 1", timestampField, p.table, timestampField)

	var timestamp interface{}
	err := p.db.QueryRowContext(ctx, query).Scan(&timestamp)
	if err != nil {
		if err == sql.ErrNoRows {
			// Table is empty
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest timestamp: %w", err)
	}

	return timestamp, nil
}

// IsTableEmpty checks if the target table is empty
func (p *PostgreSQLSink) IsTableEmpty(ctx context.Context) (bool, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s LIMIT 1", p.table)

	var count int
	err := p.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if table is empty: %w", err)
	}

	return count == 0, nil
}
