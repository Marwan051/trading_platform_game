package clients

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
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
	client          *glide.Client
	isclientHealthy atomic.Bool
	streamName      string
	eventChan       chan eventPayload
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	closeOnce       sync.Once
	logger          *slog.Logger
	maxRetries      int
	enqueueTimeout  time.Duration
	xaddTimeout     time.Duration
}

type ValkeyOptions struct {
	ValkeyHost             string
	ValkeyPort             int
	ValkeyStreamName       string
	ValkeyRequestTimeoutMs int
}

func NewValkeyClient(host string, port int, streamName string, bufferSize int, requestTimeout int) (*ValkeyClient, error) {
	clientConfig := config.NewClientConfiguration().WithAddress(&config.NodeAddress{
		Host: host,
		Port: port,
	}).WithRequestTimeout(time.Duration(requestTimeout) * time.Millisecond)

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
	// xaddTimeout comes from the provided requestTimeout (ms); fallback to 5s if zero
	vc.xaddTimeout = time.Duration(requestTimeout) * time.Millisecond
	if vc.xaddTimeout == 0 {
		vc.xaddTimeout = 2 * time.Second
	}
	// enqueueTimeout uses the same configured request timeout by default
	vc.enqueueTimeout = vc.xaddTimeout
	vc.isclientHealthy.Store(true)

	vc.wg.Add(1)
	go vc.worker()

	return vc, nil
}

// SetHealthy updates the client health status (thread-safe)
func (vc *ValkeyClient) SetHealthy(healthy bool) {
	vc.isclientHealthy.Store(healthy)
}

// GetHealthy returns the client health status (thread-safe)
func (vc *ValkeyClient) GetHealthy() bool {
	return vc.isclientHealthy.Load()
}

func (vc *ValkeyClient) IsHealthy(ctx context.Context) (bool, error) {
	resp, err := vc.client.Ping(ctx)
	if err != nil {
		vc.SetHealthy(false)
		return false, err
	}
	if resp == "PONG" {
		vc.SetHealthy(true)
		return true, nil
	}
	vc.SetHealthy(false)
	return false, errors.New("Response message not expected")
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
	// Attempt to enqueue with a bounded timeout to avoid blocking producers indefinitely
	sendFn := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = errors.New("event channel closed")
			}
		}()

		select {
		case vc.eventChan <- payload:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-vc.ctx.Done():
			return vc.ctx.Err()
		case <-time.After(vc.enqueueTimeout):
			return errors.New("enqueue timeout")
		}
	}

	return sendFn()
}

func (vc *ValkeyClient) Close(ctx context.Context) error {
	// Use Once to prevent panic on double-close
	vc.closeOnce.Do(func() {
		// cancel first so Publish stops accepting new events
		vc.cancel()
		// closing the channel signals the worker to drain then exit
		close(vc.eventChan)
	})

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
		// Keep trying to publish this event until it succeeds
		for {
			// Wait for client to be healthy before attempting to publish.
			// If context is cancelled (shutdown), waitForHealthy returns immediately.
			vc.waitForHealthy()

			// During shutdown: attempt once using a fresh context (not vc.ctx which is
			// already cancelled) so XAdd can actually reach Valkey.
			if vc.ctx.Err() != nil {
				drainCtx, drainCancel := context.WithTimeout(context.Background(), vc.xaddTimeout)
				if err := vc.publishWithRetry(drainCtx, payload); err != nil {
					vc.logger.Warn("dropping event during shutdown after failed publish",
						slog.Int("event_type", int(payload.eventType)),
						slog.String("error", err.Error()),
					)
				}
				drainCancel()
				break // move to next buffered event
			}

			// Try to publish with retry logic
			err := vc.publishWithRetry(vc.ctx, payload)
			if err == nil {
				// Success - move to next event
				break
			}

			// Failed - mark unhealthy and loop to wait and retry
			vc.logger.Error("failed to publish event after retries, will retry after health recovery",
				slog.String("stream", vc.streamName),
				slog.Int("event_type", int(payload.eventType)),
				slog.String("error", err.Error()),
				slog.Int("max_retries", vc.maxRetries),
			)
			vc.SetHealthy(false)
			// Continue inner loop - will wait for health and retry same event
		}
	}

	vc.logger.Info("worker finished processing all events")
}

// waitForHealthy blocks until the client becomes healthy
// External health checker will update isclientHealthy flag
func (vc *ValkeyClient) waitForHealthy() {
	checkInterval := 100 * time.Millisecond

	for !vc.GetHealthy() {
		// Check if context is cancelled
		select {
		case <-vc.ctx.Done():
			vc.logger.Info("stopping health wait, context cancelled")
			return
		case <-time.After(checkInterval):
			// Continue checking
			if !vc.GetHealthy() {
				vc.logger.Debug("waiting for client to become healthy")
			}
		}
	}
}

func (vc *ValkeyClient) publishWithRetry(ctx context.Context, payload eventPayload) error {
	var lastErr error
	backoff := 50 * time.Millisecond

	for attempt := 0; attempt <= vc.maxRetries; attempt++ {
		// Check context before each retry
		if ctx.Err() != nil {
			return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
		}

		// Use a per-attempt timeout for the XAdd network call so a single hung attempt
		// doesn't block retries indefinitely.
		attemptCtx, cancel := context.WithTimeout(ctx, vc.xaddTimeout)
		_, err := vc.client.XAdd(attemptCtx, vc.streamName, []models.FieldValue{
			{Field: "type", Value: fmt.Sprintf("%d", payload.eventType)},
			{Field: "data", Value: string(payload.data)},
		})
		cancel()

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
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during backoff: %w", ctx.Err())
			}
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", vc.maxRetries+1, lastErr)
}
