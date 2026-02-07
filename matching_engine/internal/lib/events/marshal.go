package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// MarshalEvent wraps the event data in an envelope with metadata and serializes to JSON.
// This function is safe to call from a background goroutine.
func MarshalEvent(eventData any, eventType EventType) ([]byte, error) {
	// Marshal the specific event data
	dataBytes, err := json.Marshal(eventData)
	if err != nil {
		return nil, err
	}

	// Wrap in envelope with metadata
	envelope := Event{
		EventID:   uuid.NewString(),
		Timestamp: time.Now(),
		Type:      eventType,
		Data:      dataBytes,
	}

	// Marshal the entire envelope
	return json.Marshal(envelope)
}
