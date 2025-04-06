package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kakhavain/telemetry-tracker/internal/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// PostgresStore implements the storer interface using PostgreSQL.
type PostgresStore struct {
	pool *pgxpool.Pool
	obs  observability.Provider
}

// NewPostgresStore creates a new PostgresStore and establishes a connection pool.
func NewPostgresStore(ctx context.Context, dsn string, obs observability.Provider) (*PostgresStore, error) {
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

	obs.Logger().Info("Connected to PostgreSQL successfully")
	return &PostgresStore{pool: pool, obs: obs}, nil
}

// StoreEvent inserts an event into the database.
func (s *PostgresStore) StoreEvent(ctx context.Context, event Event) error {
	ctx, span := s.obs.Tracer().Start(ctx, "StoreEvent")
	defer span.End()

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
		span.RecordError(err)
		span.SetStatus(codes.Error, "DB insert failed")
		return fmt.Errorf("unable to insert event: %w", err)
	}
	if cmdTag.RowsAffected() != 1 {
		err := fmt.Errorf("expected 1 row affected, got %d", cmdTag.RowsAffected())
		span.RecordError(err)
		span.SetStatus(codes.Error, "Unexpected rows affected")
		return err
	}

	span.SetAttributes(
		attribute.String("event_type", event.EventType),
		attribute.String("timestamp", event.Timestamp.String()),
	)

	return nil
}

// Close closes the database connection pool.
func (s *PostgresStore) Close() {
	s.obs.Logger().Info("Closing PostgreSQL connection pool")
	s.pool.Close()
	s.obs.Logger().Info("PostgreSQL connection pool closed")
}
