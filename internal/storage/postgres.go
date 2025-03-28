package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Event represents the structure of the telemetry data we expect and store.
type Event struct {
	EventType string          `json:"event_type"`
	Timestamp time.Time       `json:"timestamp"` // Expect ISO 8601 format
	Data      json.RawMessage `json:"data"`      // Store arbitrary JSON
}

// Storer defines the interface for storing events.
type Storer interface {
	StoreEvent(ctx context.Context, event Event) error
	Close()
}

// PostgresStore implements the Storer interface using PostgreSQL.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a new PostgresStore and establishes a connection pool.
func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	// Example pool settings (adjust based on expected load)
	// config.MaxConns = 15
	// config.MinConns = 2
	// config.MaxConnIdleTime = time.Minute * 5
	// config.MaxConnLifetime = time.Hour
	// config.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Ping the database to ensure connectivity on startup
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close() // Close pool if ping fails
		return nil, fmt.Errorf("unable to ping database on startup: %w", err)
	}

	slog.Info("Successfully connected to PostgreSQL and connection pool established")
	return &PostgresStore{pool: pool}, nil
}

// StoreEvent inserts an event into the database.
func (s *PostgresStore) StoreEvent(ctx context.Context, event Event) error {
	query := `INSERT INTO events (event_type, timestamp, data) VALUES ($1, $2, $3)`

	// Ensure timestamp is in UTC for consistency, default to Now() if zero
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	} else {
		event.Timestamp = event.Timestamp.UTC()
	}


	// Use context with a timeout for the query
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second) // Example query timeout
	defer cancel()

	commandTag, err := s.pool.Exec(queryCtx, query, event.EventType, event.Timestamp, event.Data)
	if err != nil {
		// Consider checking for specific pgx errors if needed
		return fmt.Errorf("unable to insert event: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return fmt.Errorf("expected 1 row to be affected by insert, but got %d", commandTag.RowsAffected())
	}

	return nil
}

// Close closes the database connection pool.
func (s *PostgresStore) Close() {
	slog.Info("Closing PostgreSQL connection pool...")
	s.pool.Close()
	slog.Info("PostgreSQL connection pool closed.")
}