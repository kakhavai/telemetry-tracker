package storage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore implements the storer interface using PostgreSQL.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a new PostgresStore and establishes a connection pool.
func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	slog.Info("Connected to PostgreSQL successfully")
	return &PostgresStore{pool: pool}, nil
}

// StoreEvent inserts an event into the database.
func (s *PostgresStore) StoreEvent(ctx context.Context, event Event) error {
	query := `INSERT INTO events (event_type, timestamp, data) VALUES ($1, $2, $3)`

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	} else {
		event.Timestamp = event.Timestamp.UTC()
	}

	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmdTag, err := s.pool.Exec(queryCtx, query, event.EventType, event.Timestamp, event.Data)
	if err != nil {
		return fmt.Errorf("unable to insert event: %w", err)
	}
	if cmdTag.RowsAffected() != 1 {
		return fmt.Errorf("expected 1 row affected, got %d", cmdTag.RowsAffected())
	}
	return nil
}

// Close closes the database connection pool.
func (s *PostgresStore) Close() {
	slog.Info("Closing PostgreSQL connection pool")
	s.pool.Close()
	slog.Info("PostgreSQL connection pool closed")
}
