package storage

import (
	"encoding/json"
	"time"
)

// Event represents the structure of the telemetry data we expect and store.
type Event struct {
	EventType string          `json:"event_type"`
	Timestamp time.Time       `json:"timestamp"` // Expect ISO 8601 format
	Data      json.RawMessage `json:"data"`      // Store arbitrary JSON
}
