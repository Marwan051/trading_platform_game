package streamingclient

import (
	"context"

	"github.com/Marwan051/tradding_platform_game/matching_engine/internal/lib/events"
)

// StreamingClient defines the interface for publishing events asynchronously.
// Implementations should handle serialization, buffering, and delivery to the underlying stream.
type StreamingClient interface {
	// Publish sends an event asynchronously. Returns immediately without blocking.
	// The event will be serialized and sent to the stream by a background worker.
	Publish(ctx context.Context, eventData any, eventType events.EventType) error

	// Close gracefully shuts down the client, flushing any pending events.
	// Blocks until all buffered events are published or ctx is cancelled.
	Close(ctx context.Context) error
}
