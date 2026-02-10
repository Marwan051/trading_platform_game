package streamingclient

import (
	"context"
)

// StreamingClient defines the interface for publishing events asynchronously.
// Implementations should handle serialization, buffering, and delivery to the underlying stream.
type StreamingClient interface {
	// Stream continuously reads events from the stream of the matching engine and persists them.
	Stream(ctx context.Context) error

	// Close gracefully shuts down the client, flushing any pending events.
	// Blocks until all buffered events are published or ctx is cancelled.
	Close(ctx context.Context) error
}
