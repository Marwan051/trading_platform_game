package clients

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Marwan051/tradding_platform_game/matching_engine/internal/lib/events"
	"github.com/Marwan051/tradding_platform_game/matching_engine/internal/lib/types"
	glide "github.com/valkey-io/valkey-glide/go/v2"
	"github.com/valkey-io/valkey-glide/go/v2/config"
	"github.com/valkey-io/valkey-glide/go/v2/models"
)

type eventPayload struct {
	data      []byte
	eventType types.EventType
}

type ValkeyClient struct {
	client     *glide.Client
	streamName string
	eventChan  chan eventPayload
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	logger     *slog.Logger
	maxRetries int
}

func NewValkeyClient(host string, port int, streamName string, bufferSize int) (*ValkeyClient, error) {
	clientConfig := config.NewClientConfiguration().WithAddress(&config.NodeAddress{
		Host: host,
		Port: port,
	})

	glideClient, err := glide.NewClient(clientConfig)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	vc := &ValkeyClient{
		client:     glideClient,
		streamName: streamName,
		eventChan:  make(chan eventPayload, bufferSize),
		ctx:        ctx,
		cancel:     cancel,
		logger:     slog.Default(),
		maxRetries: 3,
	}

	vc.wg.Add(1)
	go vc.worker()

	return vc, nil
}

// Pushes events to the event chan and waits for the worker to fulfil the requests
func (vc *ValkeyClient) Publish(ctx context.Context, eventData any, eventType types.EventType) error {
	data, err := events.MarshalEvent(eventData, eventType)
	if err != nil {
		return err
	}
	payload := eventPayload{
		data:      data,
		eventType: eventType,
	}
	select {
	case vc.eventChan <- payload:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-vc.ctx.Done():
		return vc.ctx.Err()
	}
}

func (vc *ValkeyClient) Close(ctx context.Context) error {
	// Signal shutdown (prevents new Publish calls from succeeding)
	vc.cancel()

	// Close channel to signal worker to stop after draining
	close(vc.eventChan)

	// Wait for worker to finish processing all buffered events
	done := make(chan struct{})
	go func() {
		vc.wg.Wait()
		close(done)
	}()

	// Wait for worker completion or context timeout
	select {
	case <-done:
		// Worker finished, close the underlying client
		vc.client.Close()
		return nil
	case <-ctx.Done():
		// Timeout - worker still processing, return error
		// (worker will continue in background until done)
		return ctx.Err()
	}
}

func (vc *ValkeyClient) worker() {
	defer vc.wg.Done()

	for payload := range vc.eventChan {
		// Check if context is cancelled before processing
		select {
		case <-vc.ctx.Done():
			vc.logger.Info("worker shutting down, context cancelled")
			return
		default:
		}

		// Try to publish with retry logic
		if err := vc.publishWithRetry(payload); err != nil {
			vc.logger.Error("failed to publish event after retries",
				slog.String("stream", vc.streamName),
				slog.Int("event_type", int(payload.eventType)),
				slog.String("error", err.Error()),
				slog.Int("max_retries", vc.maxRetries),
			)
			// Event is dropped - could add to DLQ in production
		}
	}

	vc.logger.Info("worker finished processing all events")
}

func (vc *ValkeyClient) publishWithRetry(payload eventPayload) error {
	var lastErr error
	backoff := 50 * time.Millisecond

	for attempt := 0; attempt <= vc.maxRetries; attempt++ {
		// Check context before each retry
		if vc.ctx.Err() != nil {
			return fmt.Errorf("context cancelled during retry: %w", vc.ctx.Err())
		}

		_, err := vc.client.XAdd(vc.ctx, vc.streamName, []models.FieldValue{
			{Field: "type", Value: fmt.Sprintf("%d", payload.eventType)},
			{Field: "data", Value: string(payload.data)},
		})

		if err == nil {
			if attempt > 0 {
				vc.logger.Info("event published after retry",
					slog.Int("attempt", attempt+1),
					slog.Int("event_type", int(payload.eventType)),
				)
			}
			return nil
		}

		lastErr = err

		// Don't retry on last attempt
		if attempt < vc.maxRetries {
			vc.logger.Warn("failed to publish event, retrying",
				slog.Int("attempt", attempt+1),
				slog.Int("event_type", int(payload.eventType)),
				slog.String("error", err.Error()),
				slog.Duration("backoff", backoff),
			)

			// Exponential backoff with context check
			select {
			case <-time.After(backoff):
				backoff *= 2
			case <-vc.ctx.Done():
				return fmt.Errorf("context cancelled during backoff: %w", vc.ctx.Err())
			}
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", vc.maxRetries+1, lastErr)
}
