-- Basic table to store telemetry events
CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(255) NOT NULL,
    timestamp TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    data JSONB,
    received_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP -- Track when the server got it
);

-- Optional: Add an index if you query by type or timestamp often
-- CREATE INDEX idx_events_event_type ON events(event_type);
-- CREATE INDEX idx_events_timestamp ON events(timestamp);