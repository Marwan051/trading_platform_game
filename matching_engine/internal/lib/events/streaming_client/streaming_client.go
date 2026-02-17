package streamingclient

import (
	"context"

	"github.com/Marwan051/tradding_platform_game/matching_engine/internal/lib/types"
)

// StreamingClient defines the interface for publishing events asynchronously.
// Implementations should handle serialization, buffering, and delivery to the underlying stream.
type StreamingClient interface {
	// IsHealthy returns the state of the event streamer if it is healthy or not.
	IsHealthy(ctx context.Context) (bool, error)

	// Publish sends an event asynchronously. Returns immediately without blocking.
	// The event will be serialized and sent to the stream by a background worker.
	Publish(ctx context.Context, eventData any, eventType types.EventType) error

	// Close gracefully shuts down the client, flushing any pending events.
	// Blocks until all buffered events are published or ctx is cancelled.
	Close(ctx context.Context) error
}
